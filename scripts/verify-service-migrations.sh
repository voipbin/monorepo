#!/usr/bin/env bash
# Dynamic verification of a service-owned migration set on scratch databases.
#
# usage:
#   scripts/verify-service-migrations.sh <service-dir> [<migrations-subdir>]
#   scripts/verify-service-migrations.sh --self-test
#
# For each engine (mariadb:12.3, mysql:8.0) it starts a throwaway container and checks, on an
# empty database and the service migrations (E2):
#   - the extended normalized dump of the owned tables (columns incl. comment, generation
#     expression and default, indexes, table options, constraints, foreign key rules, check
#     expressions) is identical before and after up -> down -> up
#   - up -> down -> up, status, concurrent start, dirty refusal, force recovery,
#     a database version ahead of the image, a restricted account, special-character passwords,
#     the daemon start order, a database created with the engine default collation (V13)
#   - the migrations package structure check of the service
#   - with HELPER_TESTS=1 the migrationhandler integration tests (including the lock wait case)
# It never touches a real database: only containers created here are used and removed on exit.
#
# DAEMON_BLOCKS=1 is for a service that has no cache to fail on: after migrating it keeps retrying the
# message broker and never exits by itself, so the daemon start check (V1) accepts a daemon that is
# stopped by the timeout, as long as it migrated the empty database first.
#
# Environment (defaults in brackets):
#   ENGINES_OVERRIDE  engines to run [mariadb:12.3 mysql:8.0]; any other value is not evidence
#   HELPER_TESTS      1 runs the migrationhandler integration tests [off]
#   CONTROL_BIN       control binary name [<name>-control]   SERVICE_BIN [<name>-manager]
#   VERSION_TABLE     version table [<name>_schema_migrations]
#   MIGRATE_CMD       control subcommand [migrate]           DSN_VAR [DATABASE_DSN]
#   DAEMON_ENV        space separated KEY=VALUE for the daemon start; @DSN@ is replaced by the
#                     database DSN [REDIS_ADDRESS=127.0.0.1:1 RABBITMQ_ADDRESS=... PROMETHEUS_LISTEN_ADDRESS=127.0.0.1:0]
#   DAEMON_BLOCKS     1 for a service without a cache: the daemon keeps retrying the broker and is stopped
#                     by the timeout, which the daemon start check (V1) accepts [off]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
read -ra ENGINES <<< "${ENGINES_OVERRIDE:-mariadb:12.3 mysql:8.0}"
PASS=rootpw
FAILS=0
SKIPS=0
CONTAINERS=()
WORK="$(mktemp -d)"

cleanup() { for c in "${CONTAINERS[@]:-}"; do [ -n "$c" ] && docker rm -f "$c" >/dev/null 2>&1 || true; done; rm -rf "$WORK"; }
trap cleanup EXIT

ok()   { echo "PASS  $*"; }
fail() { echo "FAIL  $*"; FAILS=$((FAILS+1)); }
skip() { echo "SKIP  $*"; SKIPS=$((SKIPS+1)); }

# Character set prefixes of string literals depend on the session that ran the DDL. They are
# removed in the default, generation and check expression lines only, never in other fields.
norm() {
  sed -E $'/^(DEF|GEN|CHK)\t/ s/(^|[^A-Za-z0-9_])_(utf8mb4|utf8mb3|utf8|latin1|binary|ascii)(\\\\{1,2})?\'/\\1\\3\'/g'
}

# dump_db <database>: extended normalized dump of $TABLES. Needs sql() and is_maria.
dump_db() {
  local db="$1" t chk
  for t in $TABLES; do
    echo "## $t"
    sql -e "SELECT 'COL',ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,EXTRA,IFNULL(CHARACTER_SET_NAME,''),IFNULL(COLLATION_NAME,''),COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY ORDINAL_POSITION"
    sql -e "SELECT 'DEF',COLUMN_NAME,IFNULL(COLUMN_DEFAULT,'<none>') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY ORDINAL_POSITION"
    sql -e "SELECT 'GEN',COLUMN_NAME,GENERATION_EXPRESSION FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' AND GENERATION_EXPRESSION IS NOT NULL AND GENERATION_EXPRESSION<>'' ORDER BY ORDINAL_POSITION"
    sql -e "SELECT 'IDX',INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE,IFNULL(SUB_PART,''),INDEX_TYPE,IFNULL(COLLATION,''),INDEX_COMMENT FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY INDEX_NAME,SEQ_IN_INDEX"
    sql -e "SELECT 'TBL',ENGINE,TABLE_COLLATION,IFNULL(ROW_FORMAT,''),TABLE_COMMENT,CREATE_OPTIONS FROM information_schema.TABLES WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t'"
    sql -e "SELECT 'CON',CONSTRAINT_NAME,CONSTRAINT_TYPE FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY CONSTRAINT_NAME"
    sql -e "SELECT 'KCU',CONSTRAINT_NAME,ORDINAL_POSITION,COLUMN_NAME,REFERENCED_TABLE_NAME,REFERENCED_COLUMN_NAME FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' AND REFERENCED_TABLE_NAME IS NOT NULL ORDER BY CONSTRAINT_NAME,ORDINAL_POSITION"
    sql -e "SELECT 'FK',CONSTRAINT_NAME,UPDATE_RULE,DELETE_RULE,REFERENCED_TABLE_NAME FROM information_schema.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY CONSTRAINT_NAME"
    # MariaDB has TABLE_NAME in CHECK_CONSTRAINTS, MySQL 8.0 has not and needs the join.
    if is_maria; then
      chk="SELECT 'CHK',CONSTRAINT_NAME,CHECK_CLAUSE FROM information_schema.CHECK_CONSTRAINTS WHERE CONSTRAINT_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY CONSTRAINT_NAME"
    else
      chk="SELECT 'CHK',cc.CONSTRAINT_NAME,cc.CHECK_CLAUSE FROM information_schema.CHECK_CONSTRAINTS cc JOIN information_schema.TABLE_CONSTRAINTS tc ON tc.CONSTRAINT_SCHEMA=cc.CONSTRAINT_SCHEMA AND tc.CONSTRAINT_NAME=cc.CONSTRAINT_NAME WHERE tc.TABLE_SCHEMA='$db' AND tc.TABLE_NAME='$t' AND tc.CONSTRAINT_TYPE='CHECK' ORDER BY cc.CONSTRAINT_NAME"
    fi
    sql -e "$chk"
  done | norm
}

# dump_complete <file>: every table of $TABLES has column lines (a missing table or a missing
# privilege would otherwise give a dump that is equal to another empty dump)
dump_complete() {
  local t
  for t in $TABLES; do
    awk -v t="## $t" -F'\t' '$0==t{f=1;next} /^## /{f=0} f&&$1=="COL"{c++} END{exit c>0?0:1}' "$1" || { echo "table $t has no columns in the dump $1 (missing table or no privilege)" >&2; return 1; }
  done
}

is_maria() { sql -e "SELECT VERSION()" | grep -qi mariadb; }

sql() { docker exec -i "$CN" $CLI --protocol=tcp -h127.0.0.1 -uroot -p$PASS -N -B "$@" 2> >(grep -v "Using a password" >&2); }

start_engine() { # start_engine <image> <container>; sets CN, PORT, CLI
  local image="$1" cn="$2"
  if [[ "$image" == mariadb* ]]; then
    docker run -d --name "$cn" -e MARIADB_ROOT_PASSWORD=$PASS -p 127.0.0.1::3306 "$image" >/dev/null
    CLI=mariadb
  else
    docker run -d --name "$cn" -e MYSQL_ROOT_PASSWORD=$PASS -p 127.0.0.1::3306 "$image" >/dev/null
    CLI=mysql
  fi
  CONTAINERS+=("$cn")
  CN="$cn"
  PORT=$(docker port "$cn" 3306/tcp | head -1 | sed 's/.*://')
  for _ in $(seq 1 90); do sql -e 'select 1' >/dev/null 2>&1 && break; sleep 1; done
  sql -e "select 1" >/dev/null
}

# ---------------------------------------------------------------------------------------------
# --self-test: the extended dump queries are exercised on a synthetic schema, because the
# service under test may use none of the elements (generated column, comment, foreign key, check).
# ---------------------------------------------------------------------------------------------
self_test() {
  local image tag cn d base tagname
  TABLES="st_parent st_child"

  # normalization: only the character set prefix of the expression lines is removed
  [ "$(printf 'DEF\tc\t_utf8mb4\x27x\x27\n' | norm)" = "$(printf 'DEF\tc\t\x27x\x27\n')" ] && ok "norm removes a character set prefix in a default" || fail "norm prefix"
  local raw_a raw_b
  raw_a='CHK'$'\t''k'$'\t''(`c` = _latin1\\'"'"'x\\'"'"')'
  raw_b='CHK'$'\t''k'$'\t''(`c` = _utf8mb4\\'"'"'x\\'"'"')'
  [ "$(echo "$raw_a" | norm)" = "$(echo "$raw_b" | norm)" ] && [ "$(echo "$raw_a" | norm)" != "$raw_a" ] && ok "norm makes escaped prefixes of different sessions equal" || fail "norm escaped prefix"
  [ "$(printf 'DEF\tc\t\x27a_b\x27\n' | norm)" != "$(printf 'DEF\tc\t\x27a_c\x27\n' | norm)" ] && ok "norm keeps 'a_b' and 'a_c' different" || fail "norm over-normalizes"
  [ "$(printf 'COL\t1\tc\t_utf8mb4\x27x\x27\n' | norm)" = "$(printf 'COL\t1\tc\t_utf8mb4\x27x\x27\n')" ] && ok "norm leaves other lines unchanged" || fail "norm other lines"

  for image in "${ENGINES[@]}"; do
    tag="${image//[:.]/_}"; cn="verify-selftest-${tag}-$$"
    echo "== self-test engine $image (container $cn)"
    start_engine "$image" "$cn"
    base="CREATE TABLE st_parent (id int NOT NULL PRIMARY KEY);
CREATE TABLE st_child (id int NOT NULL PRIMARY KEY COMMENT 'the id', parent_id int NOT NULL, j json NULL, g int GENERATED ALWAYS AS (id + 1) STORED, s varchar(10) DEFAULT 'x',
  CONSTRAINT st_fk FOREIGN KEY (parent_id) REFERENCES st_parent (id) ON DELETE CASCADE, CONSTRAINT st_chk CHECK (id > 0)) COMMENT='child table';"
    mk() { sql -e "CREATE DATABASE $1 CHARACTER SET utf8 COLLATE utf8_general_ci"; sql -D "$1" -e "$2"; }
    mk st_a "$base"; mk st_b "$base"
    mk st_comment "${base//the id/the ID}"
    mk st_gen "${base//id + 1/id + 2}"
    mk st_check "${base//id > 0/id > 1}"
    mk st_fkrule "${base//ON DELETE CASCADE/ON DELETE RESTRICT}"
    mk st_tblcomment "${base//child table/other table}"
    mk st_default "${base//DEFAULT \'x\'/DEFAULT \'y\'}"
    dump_db st_a > "$WORK/st_a.txt"; dump_db st_b > "$WORK/st_b.txt"
    for tagname in COL DEF GEN IDX TBL CON KCU FK CHK; do
      if grep -q "^$tagname"$'\t' "$WORK/st_a.txt"; then ok "$image self-test dump has $tagname lines"; else fail "$image self-test dump has no $tagname line"; fi
    done
    if grep -q "^COL.*the id" "$WORK/st_a.txt" && grep -q "^TBL.*child table" "$WORK/st_a.txt"; then ok "$image self-test dump holds the comments"; else fail "$image self-test comments"; fi
    if diff -q "$WORK/st_a.txt" "$WORK/st_b.txt" >/dev/null; then ok "$image self-test equal schemas give equal dumps"; else fail "$image self-test equal schemas differ"; fi
    for d in comment gen check fkrule tblcomment default; do
      dump_db "st_$d" > "$WORK/st_$d.txt"
      if diff -q "$WORK/st_a.txt" "$WORK/st_$d.txt" >/dev/null; then fail "$image self-test change '$d' is not visible in the dump"; else ok "$image self-test change '$d' changes the dump"; fi
    done
  done
}

if [ "${1:-}" = "--self-test" ]; then
  self_test
  echo "== engines: ${ENGINES[*]}$([ -n "${ENGINES_OVERRIDE:-}" ] && echo ' (ENGINES_OVERRIDE is set: not evidence)')"
  echo "== failures: $FAILS, skipped: $SKIPS"
  [ "$FAILS" = 0 ]
  exit
fi

SERVICE="${1:?usage: $0 <service-dir, e.g. bin-tag-manager> [<migrations-subdir>] | --self-test}"
MIG_REL="${2:-migrations}"
SVC_DIR="$ROOT/$SERVICE"
MIG_DIR="$SVC_DIR/$MIG_REL"

NAME="${SERVICE#bin-}"; NAME="${NAME%-manager}"
CONTROL="${CONTROL_BIN:-${NAME}-control}"
SERVICE_BIN="${SERVICE_BIN:-${NAME}-manager}"
VERSION_TABLE="${VERSION_TABLE:-${NAME}_schema_migrations}"
MIGRATE_CMD="${MIGRATE_CMD:-migrate}"
DSN_VAR="${DSN_VAR:-DATABASE_DSN}"
DAEMON_ENV="${DAEMON_ENV:-REDIS_ADDRESS=127.0.0.1:1 RABBITMQ_ADDRESS=amqp://guest:guest@127.0.0.1:1 PROMETHEUS_LISTEN_ADDRESS=127.0.0.1:0}"
ls "$MIG_DIR"/*.up.sql >/dev/null 2>&1 || { echo "no migration files in $MIG_DIR"; exit 1; }
# every table created by any up file, with or without IF NOT EXISTS
TABLES=$(grep -ihoE '^[[:space:]]*CREATE TABLE( IF NOT EXISTS)? +`?[a-z0-9_]+' "$MIG_DIR"/*.up.sql | awk '{print $NF}' | tr -d '`' | sort -u)
[ -n "$TABLES" ] || { echo "no tables found in $MIG_DIR"; exit 1; }
FIRST_TABLE=$(echo "$TABLES" | head -1)

# expected version: the highest up file number
N=$(ls "$MIG_DIR"/*.up.sql | sed -E 's#.*/([0-9]{6})_.*#\1#' | sort -n | tail -1 | sed -E 's/^0+//')
[ -n "$N" ] || { echo "no migration files in $MIG_DIR"; exit 1; }
if [ -f "$MIG_DIR/embed.go" ]; then
  grep -q "VersionTable = \"$VERSION_TABLE\"" "$MIG_DIR/embed.go" || { echo "migrations.VersionTable is not $VERSION_TABLE"; exit 1; }
else
  skip "no embed.go in $MIG_REL: the version table name is not checked"
fi
# a destructive migration (marker line) is exempt from the down/up check of V13
DESTRUCTIVE=0
if grep -lq '^-- DESTRUCTIVE:' "$MIG_DIR"/*.up.sql 2>/dev/null; then DESTRUCTIVE=1; fi

(cd "$SVC_DIR" && go build -o "$WORK/" ./cmd/...)

# structure check of the service migrations package
if (cd "$SVC_DIR" && go test "./$MIG_REL/" -count=1 >"$WORK/structure.out" 2>&1) && ! grep -q 'no test files' "$WORK/structure.out"; then ok "structure check of $MIG_REL: $(tail -1 "$WORK/structure.out")"; else cat "$WORK/structure.out"; fail "structure check of $MIG_REL"; fi

for IMAGE in "${ENGINES[@]}"; do
  TAG="${IMAGE//[:.]/_}"
  CN="verify-${NAME}-${TAG}-$$"
  echo "== engine $IMAGE (container $CN)"
  start_engine "$IMAGE" "$CN"

  # charset utf8 like the self-hosted init script
  for db in e2 e3 e4 e5 e6; do sql -e "CREATE DATABASE $db CHARACTER SET utf8 COLLATE utf8_general_ci"; done
  dsn() { echo "root:$PASS@tcp(127.0.0.1:$PORT)/$1?parseTime=true&loc=Local"; }
  ctl() { local db="$1"; shift; env "$DSN_VAR=$(dsn "$db")" "$WORK/$CONTROL" "$MIGRATE_CMD" "$@"; }
  version_of() { ctl "$1" status | awk -F': ' '/Current version/{print $2}'; }

  # V1/V3 empty database: migrate and dump
  ctl e2 up >/dev/null
  dump_db e2 > "$WORK/e2.txt"
  [ "$(version_of e2)" = "$N" ] && ok "$IMAGE V1 version $N" || fail "$IMAGE V1 version"
  if dump_complete "$WORK/e2.txt"; then :; else fail "$IMAGE V3 the dump holds no columns"; fi

  # V4 up -> down -> up
  if ctl e2 down >/dev/null && [ "$(version_of e2)" = "$((N-1))" ]; then
    if [ "$N" = 1 ]; then
      [ "$(sql -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='e2' AND TABLE_NAME='$FIRST_TABLE'")" = 1 ] || fail "$IMAGE V4 the baseline down removed a table"
    fi
    if ctl e2 up >/dev/null && [ "$(version_of e2)" = "$N" ]; then ok "$IMAGE V4 up/down/up ($N -> $((N-1)) -> $N; baseline down keeps the tables)"; else fail "$IMAGE V4 up after down"; fi
  else
    fail "$IMAGE V4 down"
  fi
  if dump_db e2 | diff -q - "$WORK/e2.txt" >/dev/null; then ok "$IMAGE V4 dump unchanged after up/down/up"; else fail "$IMAGE V4 dump"; fi

  # V5 concurrent start on an empty database
  ( ctl e3 up >"$WORK/c1.out" 2>&1; echo $? >"$WORK/c1.rc" ) & ( ctl e3 up >"$WORK/c2.out" 2>&1; echo $? >"$WORK/c2.rc" ) & wait
  if [ "$(cat "$WORK/c1.rc")" = 0 ] && [ "$(cat "$WORK/c2.rc")" = 0 ] && [ "$(version_of e3)" = "$N" ]; then ok "$IMAGE V5 concurrent up: both exit 0, version $N"; else fail "$IMAGE V5 concurrent up ($(cat "$WORK/c1.rc") $(cat "$WORK/c2.rc"))"; fi

  # V8 dirty refused, force recovers
  sql -e "UPDATE e3.$VERSION_TABLE SET dirty=1"
  if ctl e3 up >"$WORK/dirty.out" 2>&1; then fail "$IMAGE V8 dirty accepted"; elif grep -q "Dirty database version $N" "$WORK/dirty.out"; then ok "$IMAGE V8 dirty refused"; else fail "$IMAGE V8 message"; fi
  if ctl e3 force "$N" >/dev/null && ctl e3 up >/dev/null && [ "$(version_of e3)" = "$N" ]; then ok "$IMAGE V8 force $N then up recovers"; else fail "$IMAGE V8 recover"; fi
  if [ "$N" = 1 ]; then
    if ctl e3 force -- -1 >/dev/null && [ "$(version_of e3)" = "0" ] && ctl e3 up >/dev/null && [ "$(version_of e3)" = "1" ]; then ok "$IMAGE V8 force -1 resets and the baseline reapplies"; else fail "$IMAGE V8 force -1"; fi
  else
    if ctl e3 force "$((N-1))" >/dev/null && [ "$(version_of e3)" = "$((N-1))" ] && ctl e3 up >/dev/null && [ "$(version_of e3)" = "$N" ]; then ok "$IMAGE V8 force $((N-1)) then up applies migration $N"; else fail "$IMAGE V8 force $((N-1))"; fi
  fi

  # V9 database ahead of the image
  ctl e3 force "$((N+1))" >/dev/null
  if ctl e3 up >"$WORK/ahead.out" 2>&1 && [ "$(version_of e3)" = "$((N+1))" ] && grep -q "ahead of this image" "$WORK/ahead.out"; then ok "$IMAGE V9 ahead of image: warning logged and migrations skipped"; else cat "$WORK/ahead.out"; fail "$IMAGE V9"; fi
  sql -e "UPDATE e3.$VERSION_TABLE SET dirty=1"
  if ctl e3 up >"$WORK/dirtyahead.out" 2>&1; then
    fail "$IMAGE V9 dirty+ahead accepted"
  elif grep -q "Dirty database version $((N+1))" "$WORK/dirtyahead.out"; then
    ok "$IMAGE V9 dirty and ahead is refused"
  else
    cat "$WORK/dirtyahead.out"; fail "$IMAGE V9 dirty+ahead message"
  fi

  # V6 special-character password and DSN parameters
  SPPW="a+b""%41""c""@/?#"":&"
  sql -e "CREATE USER 'sp'@'%' IDENTIFIED BY '$SPPW'; GRANT ALL ON e4.* TO 'sp'@'%'"
  if env "$DSN_VAR=sp:$SPPW@tcp(127.0.0.1:$PORT)/e4?parseTime=true&loc=Local" "$WORK/$CONTROL" "$MIGRATE_CMD" up >"$WORK/sp.out" 2>&1; then ok "$IMAGE V6 password with % + @ / ? # : & and DSN parameters"; else cat "$WORK/sp.out"; fail "$IMAGE V6"; fi

  # V1 service start: the daemon applies the migrations right after the database connection,
  # before the cache and the message broker. Redis and RabbitMQ are unreachable here on purpose,
  # so the daemon must exit non-zero after migrating.
  DAEMON_KV=()
  for kv in $DAEMON_ENV; do DAEMON_KV+=("${kv//@DSN@/$(dsn e6)}"); done
  rc=0
  # A service without a cache connects the broker next and keeps retrying, so it never exits by itself.
  # DAEMON_BLOCKS=1 accepts that: the daemon is stopped by the timeout (the daemon traps TERM, so the
  # timeout also sends KILL after a grace period) and the check is that it migrated first.
  if [ "${DAEMON_BLOCKS:-}" = 1 ]; then
    env "$DSN_VAR=$(dsn e6)" "${DAEMON_KV[@]}" timeout -k 5 30 "$WORK/$SERVICE_BIN" >"$WORK/daemon.out" 2>&1 || rc=$?
    if [ "$rc" = 124 ] || [ "$rc" = 137 ]; then
      rc=1
    fi
  else
    env "$DSN_VAR=$(dsn e6)" "${DAEMON_KV[@]}" timeout 60 "$WORK/$SERVICE_BIN" >"$WORK/daemon.out" 2>&1 || rc=$?
  fi
  if [ "$rc" = 0 ] || [ "$rc" = 124 ]; then
    fail "$IMAGE V1 daemon did not exit by itself with an error (rc=$rc)"
  elif [ "$(version_of e6)" = "$N" ] && [ "$(sql -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='e6' AND TABLE_NAME='$FIRST_TABLE'")" = "1" ]; then
    ok "$IMAGE V1 daemon start migrated an empty database before connecting the cache"
  else
    tail -5 "$WORK/daemon.out"; fail "$IMAGE V1 daemon start did not migrate"
  fi

  # V13 database created with the engine default collation: only the character set is named, so
  # MariaDB 12.x resolves it to utf8mb3_uca1400_ai_ci. A migration that compares a function result
  # with a literal fails here (Error 1267) while the explicit utf8_general_ci databases above pass.
  sql -e "CREATE DATABASE e8 CHARACTER SET utf8"
  if ! ctl e8 up >"$WORK/e8.out" 2>&1 || [ "$(version_of e8)" != "$N" ]; then
    tail -5 "$WORK/e8.out"; fail "$IMAGE V13 default-collation database up"
  elif [ "$DESTRUCTIVE" = 1 ]; then
    ok "$IMAGE V13 default-collation database: up ($N)"
    skip "$IMAGE V13 down/up (DESTRUCTIVE migration)"
  else
    d=0; while [ "$d" -lt "$N" ] && ctl e8 down >>"$WORK/e8.out" 2>&1; do d=$((d+1)); done
    if [ "$d" = "$N" ] && [ "$(version_of e8)" = "0" ] && ctl e8 up >>"$WORK/e8.out" 2>&1 && [ "$(version_of e8)" = "$N" ]; then
      ok "$IMAGE V13 default-collation database: up/down/up ($N -> 0 -> $N)"
    else
      tail -5 "$WORK/e8.out"; fail "$IMAGE V13 default-collation database down/up"
    fi
  fi

  # V7 account without CREATE
  sql -e "CREATE USER 'ro'@'%' IDENTIFIED BY 'ropw'; GRANT SELECT ON e5.* TO 'ro'@'%'"
  if env "$DSN_VAR=ro:ropw@tcp(127.0.0.1:$PORT)/e5" "$WORK/$CONTROL" "$MIGRATE_CMD" up >"$WORK/ro.out" 2>&1; then
    fail "$IMAGE V7 restricted account succeeded"
  elif grep -qi "denied" "$WORK/ro.out"; then
    ok "$IMAGE V7 restricted account: $(grep -io 'CREATE command denied[^(]*' "$WORK/ro.out" | head -1)"
  else
    cat "$WORK/ro.out"; fail "$IMAGE V7 message"
  fi

  # migrationhandler integration tests (V12 lock wait timeout included), only for the helper change
  if [ "${HELPER_TESTS:-0}" = 1 ]; then
    sql -e "CREATE DATABASE it"
    if (cd "$ROOT/bin-common-handler" && MIGRATOR_TEST_DSN="root:$PASS@tcp(127.0.0.1:$PORT)/it" go test ./pkg/migrationhandler/ -count=1 >"$WORK/it.out" 2>&1); then
      ok "$IMAGE migrationhandler integration tests (V5, V8, V9, V12): $(tail -1 "$WORK/it.out")"
    else
      cat "$WORK/it.out"; fail "$IMAGE migrationhandler integration tests"
    fi
  fi
done

echo "== engines: ${ENGINES[*]}$([ -n "${ENGINES_OVERRIDE:-}" ] && echo ' (ENGINES_OVERRIDE is set: not evidence)')"
[ "${HELPER_TESTS:-0}" = 1 ] || echo "== migrationhandler integration tests not run (HELPER_TESTS=1 is required for the helper change evidence)"
echo "== failures: $FAILS, skipped: $SKIPS"
[ "$FAILS" = 0 ]
