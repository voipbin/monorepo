#!/usr/bin/env bash
# Dynamic verification of a service-owned migration set on scratch databases.
#
# usage: ALEMBIC_VENV=<venv with the pinned alembic> scripts/verify-service-migrations.sh bin-tag-manager
#
# Written for a service whose only migration is the baseline: the expected versions (1), the
# Alembic-vs-service dump equality and the single down step assume it. Extend the version and
# dump expectations when the first migration after the baseline is added.
#
# For each engine (mariadb:12.3, mysql:8.0) it starts a throwaway container and checks:
#   E1  database built by the Alembic head          vs  E2  empty database + service migrations
#   - normalized dump of the owned tables is identical (the baseline contract)
#   - applying the service migrations to E1 changes neither the dump nor the row count
#   - up -> down -> up, status, concurrent start, dirty refusal, force recovery,
#     a database version ahead of the image, a restricted account, special-character passwords
#   - the migrator integration tests (including the lock wait timeout case)
# It never touches a real database: only containers created here are used and removed on exit.
set -euo pipefail

SERVICE="${1:?usage: $0 <service-dir, e.g. bin-tag-manager>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVC_DIR="$ROOT/$SERVICE"
: "${ALEMBIC_VENV:?set ALEMBIC_VENV to a venv with alembic 1.11.3, SQLAlchemy 1.4.52, PyMySQL 1.1.0}"
read -ra ENGINES <<< "${ENGINES_OVERRIDE:-mariadb:12.3 mysql:8.0}"

NAME="${SERVICE#bin-}"; NAME="${NAME%-manager}"
CONTROL="${NAME}-control"
SERVICE_BIN="${NAME}-manager"
# every table created by any up file, with or without IF NOT EXISTS
TABLES=$(grep -ihoE '^[[:space:]]*CREATE TABLE( IF NOT EXISTS)? +`?[a-z0-9_]+' "$SVC_DIR"/migrations/*.up.sql | awk '{print $NF}' | tr -d '`' | sort -u)
[ -n "$TABLES" ] || { echo "no tables found in $SVC_DIR/migrations"; exit 1; }
VERSION_TABLE="${NAME}_schema_migrations"
grep -q "VersionTable = \"$VERSION_TABLE\"" "$SVC_DIR/migrations/embed.go" || { echo "migrations.VersionTable is not $VERSION_TABLE"; exit 1; }
WORK="$(mktemp -d)"
PASS=rootpw
FAILS=0
CONTAINERS=()

cleanup() { for c in "${CONTAINERS[@]:-}"; do [ -n "$c" ] && docker rm -f "$c" >/dev/null 2>&1 || true; done; rm -rf "$WORK"; }
trap cleanup EXIT

ok()   { echo "PASS  $*"; }
fail() { echo "FAIL  $*"; FAILS=$((FAILS+1)); }

(cd "$SVC_DIR" && go build -o "$WORK/" ./cmd/...)

for IMAGE in "${ENGINES[@]}"; do
  TAG="${IMAGE//[:.]/_}"
  CN="verify-${NAME}-${TAG}-$$"
  echo "== engine $IMAGE (container $CN)"
  if [[ "$IMAGE" == mariadb* ]]; then
    docker run -d --name "$CN" -e MARIADB_ROOT_PASSWORD=$PASS -p 127.0.0.1::3306 "$IMAGE" >/dev/null
    CLI=mariadb
  else
    docker run -d --name "$CN" -e MYSQL_ROOT_PASSWORD=$PASS -p 127.0.0.1::3306 "$IMAGE" >/dev/null
    CLI=mysql
  fi
  CONTAINERS+=("$CN")
  PORT=$(docker port "$CN" 3306/tcp | head -1 | sed 's/.*://')
  sql() { docker exec -i "$CN" $CLI --protocol=tcp -h127.0.0.1 -uroot -p$PASS -N -B "$@" 2> >(grep -v "Using a password" >&2); }
  for _ in $(seq 1 90); do sql -e 'select 1' >/dev/null 2>&1 && break; sleep 1; done
  sql -e "select 1" >/dev/null

  # charset utf8 like the self-hosted init script
  for db in e1 e2 e3 e4 e5 e6; do sql -e "CREATE DATABASE $db CHARACTER SET utf8 COLLATE utf8_general_ci"; done
  dsn() { echo "root:$PASS@tcp(127.0.0.1:$PORT)/$1?parseTime=true&loc=Local"; }
  ctl() { local db="$1"; shift; DATABASE_DSN="$(dsn "$db")" "$WORK/$CONTROL" "$@"; }
  dump() { # normalized dump of the owned tables of database $1
    local db="$1" t
    for t in $TABLES; do
      echo "## $t"
      sql -e "SELECT 'COL',ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,IFNULL(COLUMN_DEFAULT,'<none>'),EXTRA,IFNULL(CHARACTER_SET_NAME,''),IFNULL(COLLATION_NAME,'') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY ORDINAL_POSITION" | sed 's/\tNULL\t/\t<null>\t/'
      sql -e "SELECT 'IDX',INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE,IFNULL(SUB_PART,''),INDEX_TYPE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY INDEX_NAME,SEQ_IN_INDEX"
      sql -e "SELECT 'TBL',ENGINE,TABLE_COLLATION FROM information_schema.TABLES WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t'"
      sql -e "SELECT 'CON',CONSTRAINT_NAME,CONSTRAINT_TYPE FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA='$db' AND TABLE_NAME='$t' ORDER BY CONSTRAINT_NAME"
    done
  }
  version_of() { ctl "$1" migrate status | awk -F': ' '/Current version/{print $2}'; }

  # E1: Alembic head
  ALE="$WORK/alembic-$TAG"; cp -r "$ROOT/bin-dbscheme-manager/bin-manager" "$ALE"
  sed "s#^sqlalchemy.url = .*#sqlalchemy.url = mysql+pymysql://root:$PASS@127.0.0.1:$PORT/e1#" "$ALE/alembic.ini.sample" > "$ALE/alembic.ini"
  (cd "$ALE" && "$ALEMBIC_VENV/bin/alembic" -c alembic.ini upgrade head >/dev/null 2>"$WORK/alembic.err") || { cat "$WORK/alembic.err"; fail "$IMAGE alembic upgrade head"; continue; }
  ok "$IMAGE E1 built by the Alembic head: $(sql -e "SELECT version_num FROM e1.alembic_version")"
  dump e1 > "$WORK/e1-before.txt"

  # V2 existing database with data
  ID='0x00112233445566778899aabbccddeeff'
  sql -e "INSERT INTO e1.$(echo $TABLES | awk '{print $1}') (id, customer_id, name) VALUES ($ID, $ID, 'existing')"
  ROWS_BEFORE=$(sql -e "SELECT COUNT(*) FROM e1.$(echo $TABLES | awk '{print $1}')")
  ctl e1 migrate up >/dev/null
  dump e1 > "$WORK/e1-after.txt"
  ROWS_AFTER=$(sql -e "SELECT COUNT(*) FROM e1.$(echo $TABLES | awk '{print $1}')")
  [ "$(version_of e1)" = "1" ] && [ "$ROWS_BEFORE" = "$ROWS_AFTER" ] && diff -q "$WORK/e1-before.txt" "$WORK/e1-after.txt" >/dev/null \
    && ok "$IMAGE V2 existing database unchanged (rows $ROWS_BEFORE, version 1, dump identical)" || fail "$IMAGE V2 existing database"

  # V1/V3 empty database, dump equals E1
  ctl e2 migrate up >/dev/null
  dump e2 > "$WORK/e2.txt"
  if diff "$WORK/e1-before.txt" "$WORK/e2.txt" >"$WORK/diff.txt"; then ok "$IMAGE V1/V3 normalized dump E1 vs E2: diff 0 ($(wc -l < "$WORK/e2.txt") lines)"; else cat "$WORK/diff.txt"; fail "$IMAGE V3 dump differs"; fi
  [ "$(version_of e2)" = "1" ] && ok "$IMAGE V1 version 1" || fail "$IMAGE V1 version"

  # V4 up -> down -> up
  ctl e2 migrate down >/dev/null && [ "$(version_of e2)" = "0" ] && ctl e2 migrate up >/dev/null && [ "$(version_of e2)" = "1" ] \
    && ok "$IMAGE V4 up/down/up (baseline down is a no-op, tables kept: $(sql -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='e2' AND TABLE_NAME='$(echo $TABLES | awk '{print $1}')'"))" || fail "$IMAGE V4"
  dump e2 | diff -q - "$WORK/e2.txt" >/dev/null && ok "$IMAGE V4 dump unchanged after up/down/up" || fail "$IMAGE V4 dump"

  # V5 concurrent start on an empty database
  ( ctl e3 migrate up >"$WORK/c1.out" 2>&1; echo $? >"$WORK/c1.rc" ) & ( ctl e3 migrate up >"$WORK/c2.out" 2>&1; echo $? >"$WORK/c2.rc" ) & wait
  [ "$(cat "$WORK/c1.rc")" = 0 ] && [ "$(cat "$WORK/c2.rc")" = 0 ] && [ "$(version_of e3)" = "1" ] && ok "$IMAGE V5 concurrent up: both exit 0, version 1" || fail "$IMAGE V5 concurrent up ($(cat "$WORK/c1.rc") $(cat "$WORK/c2.rc"))"

  # V8 dirty refused, force recovers
  sql -e "UPDATE e3.$VERSION_TABLE SET dirty=1"
  if ctl e3 migrate up >"$WORK/dirty.out" 2>&1; then fail "$IMAGE V8 dirty accepted"; else grep -q "Dirty database version 1" "$WORK/dirty.out" && ok "$IMAGE V8 dirty refused" || fail "$IMAGE V8 message"; fi
  ctl e3 migrate force 1 >/dev/null && ctl e3 migrate up >/dev/null && [ "$(version_of e3)" = "1" ] && ok "$IMAGE V8 force 1 then up recovers" || fail "$IMAGE V8 recover"
  ctl e3 migrate force -- -1 >/dev/null && [ "$(version_of e3)" = "0" ] && ctl e3 migrate up >/dev/null && [ "$(version_of e3)" = "1" ] && ok "$IMAGE V8 force -1 resets and the baseline reapplies" || fail "$IMAGE V8 force -1"

  # V9 database ahead of the image
  ctl e3 migrate force 2 >/dev/null
  if ctl e3 migrate up >"$WORK/ahead.out" 2>&1 && [ "$(version_of e3)" = "2" ] && grep -q "ahead of this image" "$WORK/ahead.out"; then ok "$IMAGE V9 ahead of image: warning logged and migrations skipped"; else cat "$WORK/ahead.out"; fail "$IMAGE V9"; fi
  sql -e "UPDATE e3.$VERSION_TABLE SET dirty=1"
  if ctl e3 migrate up >"$WORK/dirtyahead.out" 2>&1; then fail "$IMAGE V9 dirty+ahead accepted"; else grep -q "Dirty database version 2" "$WORK/dirtyahead.out" && ok "$IMAGE V9 dirty and ahead is refused" || { cat "$WORK/dirtyahead.out"; fail "$IMAGE V9 dirty+ahead message"; }; fi

  # V6 special-character password and DSN parameters
  SPPW="a+b""%41""c""@/?#"":&"
  sql -e "CREATE USER 'sp'@'%' IDENTIFIED BY '$SPPW'; GRANT ALL ON e4.* TO 'sp'@'%'"
  DATABASE_DSN="sp:$SPPW@tcp(127.0.0.1:$PORT)/e4?parseTime=true&loc=Local" "$WORK/$CONTROL" migrate up >"$WORK/sp.out" 2>&1 && ok "$IMAGE V6 password with % + @ / ? # : & and DSN parameters" || { cat "$WORK/sp.out"; fail "$IMAGE V6"; }

  # V1 service start: the daemon applies the migrations right after the database connection,
  # before the cache and the message broker. Redis and RabbitMQ are unreachable here on purpose,
  # so the daemon must exit non-zero after migrating.
  rc=0
  DATABASE_DSN="$(dsn e6)" REDIS_ADDRESS="127.0.0.1:1" RABBITMQ_ADDRESS="amqp://guest:guest@127.0.0.1:1" PROMETHEUS_LISTEN_ADDRESS="127.0.0.1:0" timeout 60 "$WORK/$SERVICE_BIN" >"$WORK/daemon.out" 2>&1 || rc=$?
  if [ "$rc" = 0 ] || [ "$rc" = 124 ]; then
    fail "$IMAGE V1 daemon did not exit by itself with an error (rc=$rc)"
  elif [ "$(version_of e6)" = "1" ] && [ "$(sql -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='e6' AND TABLE_NAME='$(echo $TABLES | awk '{print $1}')'")" = "1" ]; then
    ok "$IMAGE V1 daemon start migrated an empty database before connecting the cache"
  else
    tail -5 "$WORK/daemon.out"; fail "$IMAGE V1 daemon start did not migrate"
  fi

  # Order on a new database: Alembic first, then the service. The reverse order is unsupported:
  # the Alembic history renames tags to tag_tags and fails when the service already created it.
  sql -e "CREATE DATABASE e7 CHARACTER SET utf8 COLLATE utf8_general_ci"
  ctl e7 migrate up >/dev/null
  sed "s#/e1\$#/e7#" "$ALE/alembic.ini" > "$ALE/alembic-e7.ini"
  if (cd "$ALE" && "$ALEMBIC_VENV/bin/alembic" -c alembic-e7.ini upgrade head >/dev/null 2>"$WORK/alembic-e7.err"); then
    fail "$IMAGE Alembic after the service unexpectedly succeeded"
  elif grep -qi "already exists" "$WORK/alembic-e7.err" && grep -q "$(echo $TABLES | awk '{print $1}')" "$WORK/alembic-e7.err"; then
    ok "$IMAGE service-first then Alembic fails with 'already exists' (unsupported order, documented)"
  else
    tail -3 "$WORK/alembic-e7.err"; fail "$IMAGE service-first then Alembic failed for another reason"
  fi

  # V7 account without CREATE
  sql -e "CREATE USER 'ro'@'%' IDENTIFIED BY 'ropw'; GRANT SELECT ON e5.* TO 'ro'@'%'"
  if DATABASE_DSN="ro:ropw@tcp(127.0.0.1:$PORT)/e5" "$WORK/$CONTROL" migrate up >"$WORK/ro.out" 2>&1; then fail "$IMAGE V7 restricted account succeeded"; else grep -qi "denied" "$WORK/ro.out" && ok "$IMAGE V7 restricted account: $(grep -io 'CREATE command denied[^(]*' "$WORK/ro.out" | head -1)" || { cat "$WORK/ro.out"; fail "$IMAGE V7 message"; }; fi

  # migrator integration tests (V12 lock wait timeout included)
  sql -e "CREATE DATABASE it"
  if (cd "$SVC_DIR" && MIGRATOR_TEST_DSN="root:$PASS@tcp(127.0.0.1:$PORT)/it" go test ./internal/migrator/ -count=1 >"$WORK/it.out" 2>&1); then ok "$IMAGE migrator integration tests (V5, V8, V9, V12): $(tail -1 "$WORK/it.out")"; else cat "$WORK/it.out"; fail "$IMAGE migrator integration tests"; fi
done

echo "== failures: $FAILS"
[ "$FAILS" = 0 ]
