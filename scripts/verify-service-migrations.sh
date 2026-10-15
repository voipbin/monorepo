#!/usr/bin/env bash
# Dynamic verification of a service-owned migration set on scratch databases.
#
# usage:
#   ALEMBIC_VENV=<venv with the pinned alembic> scripts/verify-service-migrations.sh <service-dir> [<migrations-subdir>]
#   scripts/verify-service-migrations.sh --dump <service-dir> <database> [<migrations-subdir>]
#   scripts/verify-service-migrations.sh --self-test
#
# For each engine (mariadb:12.3, mysql:8.0) it starts a throwaway container and checks:
#   E1  database built by the Alembic head          vs  E2  empty database + service migrations
#   - the extended normalized dump of the owned tables (columns incl. comment, generation
#     expression and default, indexes, table options, constraints, foreign key rules, check
#     expressions) of E1 + the service migrations is identical to E2 (the baseline contract)
#   - applying the service migrations to E1 changes no row count (and, for a baseline only
#     service, not the dump)
#   - up -> down -> up, status, concurrent start, dirty refusal, force recovery,
#     a database version ahead of the image, a restricted account, special-character passwords,
#     the daemon start order
#   - the migrations package structure check of the service
#   - with HELPER_TESTS=1 the migrationhandler integration tests (including the lock wait case)
# It never touches a real database: only containers created here are used and removed on exit.
#
# Optional reference dump: REFERENCE_DUMP=<file> compares E2 with an externally produced
# extended normalized dump (made with the --dump mode) in addition to E1. The two engines print
# some values differently (for example column defaults), so a reference dump belongs to one engine:
# REFERENCE_ENGINE=<image> names it (required) and the comparison runs on that engine only. ALLOWED_DIFF=<file>
# lists the accepted differences as diff lines ("< reference" / "> E2"); every accepted line that
# was used is printed as ALLOWED-DIFF. Any other difference fails.
#
# E1_ALLOWED_DIFF=<file> is the list for the other comparison: E1 (Alembic head plus the service
# migrations) against E2 (empty database plus the service migrations), which is otherwise a strict
# diff 0. Use it only for a baseline that differs from the Alembic head on purpose. It is not the
# list of REFERENCE_DUMP (ALLOWED_DIFF): the reference comparison is not changed by it. A line of
# the file is a "#" comment, a pair line or a directive; anything else is refused before a container
# starts:
#   "< E1" / "> E2" pair lines with the tab separated fields of the dump, limited to the TBL and COL
#       tags; the two sides may differ only in the collation field (3rd field of TBL, 8th of COL).
#       The dump lines carry no table name, so a listed line is accepted for every table, and a line
#       that appears in the diff N times (N tables) is listed N times. For a table under
#       column-order the COL pair line has an empty ordinal (2nd) field.
#   fk-name <table>      the foreign keys of the table may differ in the name only (kind, columns,
#       referenced table and columns, ON UPDATE and ON DELETE rules are compared per key).
#   column-order <table> the column ordinals of the table may differ; the column set and every column
#       definition are compared.
# Used entries are printed as ALLOWED-DIFF, entries no engine needed as UNUSED-ALLOWED-DIFF (not a
# failure; an engine without the difference leaves them unused).
#
# --dump prints the extended normalized dump of the owned tables of <database>. The connection
# comes from SQL_CMD (default "mysql -N -B"; put the credentials in a client option file). This
# script does not open connections by itself in that mode.
#
# Environment (defaults in brackets):
#   ALEMBIC_VENV      venv with alembic 1.11.3, SQLAlchemy 1.4.52, PyMySQL 1.1.0 (required)
#   ENGINES_OVERRIDE  engines to run [mariadb:12.3 mysql:8.0]; any other value is not evidence
#   HELPER_TESTS      1 runs the migrationhandler integration tests [off]
#   CONTROL_BIN       control binary name [<name>-control]   SERVICE_BIN [<name>-manager]
#   VERSION_TABLE     version table [<name>_schema_migrations]
#   MIGRATE_CMD       control subcommand [migrate]           DSN_VAR [DATABASE_DSN]
#   DAEMON_ENV        space separated KEY=VALUE for the daemon start; @DSN@ is replaced by the
#                     database DSN [REDIS_ADDRESS=127.0.0.1:1 RABBITMQ_ADDRESS=... PROMETHEUS_LISTEN_ADDRESS=127.0.0.1:0]
#   EXISTING_ROWS_SQL file with INSERT statements run against the Alembic database before the
#                     service migrations [built in rows for tag_tags]
#   REFERENCE_ENGINE engine image the REFERENCE_DUMP was made with (required with REFERENCE_DUMP)
#   E1_ALLOWED_DIFF  file of accepted E1 versus E2 differences: collation, fk-name, column-order (see above) [none: diff 0]
#   SEED_COMPARE_SQL  SELECT (@DB@ is the database name) whose output must be equal in E1 and E2
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

# check_allowed_file <file>: validates the list of accepted E1 versus E2 differences. Prints the
# reason and returns 1 when it is not usable. A non empty line is a "#" comment, a "< " or "> "
# pair line (TBL or COL tag, collation field only), or one directive "fk-name <table>" or
# "column-order <table>" with a table of $TABLES.
check_allowed_file() {
  local f="$1" bad dup t
  [ -f "$f" ] || { echo "$f is not a regular file"; return 1; }
  if grep -q $'\r' "$f"; then echo "$f contains a carriage return"; return 1; fi
  bad=$(grep -vE '^$|^#|^[<>]|^(fk-name|column-order) [a-z0-9_]+$' "$f" | head -1 || true)
  if [ -n "$bad" ]; then echo "$f: only comments (#), pair lines ('< ' or '> ') and the directives 'fk-name <table>' and 'column-order <table>' are accepted: $bad"; return 1; fi
  bad=$(grep -E '^[<>]' "$f" | grep -vE $'^[<>] (TBL|COL)\t' | head -1 || true)
  if [ -n "$bad" ]; then echo "$f: only '< ' or '> ' lines of the TBL and COL tags with tab separated fields are accepted: $bad"; return 1; fi
  grep -qE '^([<>] |fk-name |column-order )' "$f" || { echo "$f has no '< ' or '> ' line and no directive"; return 1; }
  dup=$(grep -E '^(fk-name|column-order) ' "$f" | sort | uniq -d | head -1 || true)
  if [ -n "$dup" ]; then echo "$f: duplicate directive: $dup"; return 1; fi
  for t in $(grep -E '^(fk-name|column-order) ' "$f" | cut -d' ' -f2); do
    case " $(echo $TABLES) " in *" $t "*) ;; *) echo "$f: directive for $t, which is not a table of this service"; return 1;; esac
  done
  awk 'BEGIN{FS=OFS="\t"}
    /^[<>] /{
      side=substr($1,1,1); tag=substr($1,3)
      if (tag=="COL") { if (NF<8) { print "COL line has fewer than 8 fields: " $0; bad=1; next } $8="" }
      else           { if (NF<3) { print "TBL line has fewer than 3 fields: " $0; bad=1; next } $3="" }
      n[substr($0,3)] += (side=="<") ? 1 : -1
    }
    END{ for (k in n) if (n[k]!=0) { print "the two sides differ in more than the collation field (or in the number of lines): " k; bad=1 } exit bad?1:0 }' "$f" || return 1
}

# directive_tables <file> <directive>: the tables named by the directive lines, space separated.
directive_tables() { { grep -E "^$2 [a-z0-9_]+\$" "$1" || true; } | cut -d' ' -f2 | tr '\n' ' '; }

# canon_dump <dump> <fk-name tables> <column-order tables>: the dump with the two intended
# differences removed for the listed tables (space separated lists; both empty: identical output).
#  - column-order: the COL lines are sorted by column name with an empty ordinal field, the DEF and
#    GEN lines are sorted by column name. Column set and every definition stay compared.
#  - fk-name: each foreign key gets one signature joining its KCU lines (ordinal, column,
#    referenced table and column) and its FK line (rules, referenced table); the keys are renamed
#    fk#1, fk#2, ... in signature order in the CON (type FOREIGN KEY), KCU and FK lines. The renamed
#    CON lines follow the unchanged CON lines; the KCU and FK groups are sorted by new name.
canon_dump() {
  LC_ALL=C awk -v fk=" $2 " -v ord=" $3 " '
    function srt(K, V, n,   i, j, k, v) {
      for (i = 2; i <= n; i++) { k = K[i]; v = V[i]; j = i - 1
        while (j >= 1 && K[j] > k) { K[j+1] = K[j]; V[j+1] = V[j]; j-- }
        K[j+1] = k; V[j+1] = v }
    }
    function tabs(line, f) { return split(line, f, "\t") }
    function join(f, n,   i, r) { r = f[1]; for (i = 2; i <= n; i++) r = r "\t" f[i]; return r }
    function run(tag, a, b,   i, n, f, K, V, m, nk, nv) {
      # lines a..b of the section buffer, all with the same tag
      if (tag == "COL" && doord) {
        for (i = a; i <= b; i++) { n = tabs(L[i], f); K[i-a+1] = f[3] ""; f[2] = ""; V[i-a+1] = join(f, n) }
        srt(K, V, b - a + 1); for (i = 1; i <= b - a + 1; i++) print V[i]; return
      }
      if ((tag == "DEF" || tag == "GEN") && doord) {
        for (i = a; i <= b; i++) { tabs(L[i], f); K[i-a+1] = f[2] ""; V[i-a+1] = L[i] }
        srt(K, V, b - a + 1); for (i = 1; i <= b - a + 1; i++) print V[i]; return
      }
      if (tag == "CON" && dofk) {
        m = 0
        for (i = a; i <= b; i++) { n = tabs(L[i], f)
          if (f[3] == "FOREIGN KEY" && (f[2] in nn)) { m++; K[m] = sprintf("%06d", nidx[f[2]]); f[2] = nn[f[2]]; V[m] = join(f, n) }
          else print L[i] }
        srt(K, V, m); for (i = 1; i <= m; i++) print V[i]; return
      }
      if ((tag == "KCU" || tag == "FK") && dofk) {
        for (i = a; i <= b; i++) { n = tabs(L[i], f)
          if (f[2] in nn) { K[i-a+1] = sprintf("%06d", nidx[f[2]]); f[2] = nn[f[2]]; V[i-a+1] = join(f, n) }
          else { K[i-a+1] = "999999"; V[i-a+1] = L[i] } }
        srt(K, V, b - a + 1); for (i = 1; i <= b - a + 1; i++) print V[i]; return
      }
      for (i = a; i <= b; i++) print L[i]
    }
    function section(   i, n, f, name, c, k, sg, SK, SV, a, tag, t2) {
      if (cnt == 0) return
      split(L[1], f, " "); t = f[2]
      dofk = index(fk, " " t " ") > 0; doord = index(ord, " " t " ") > 0
      if (dofk) {
        delete ksig; delete fsig; delete nn; delete nidx; c = 0
        for (i = 2; i <= cnt; i++) { n = tabs(L[i], f)
          if (f[1] == "KCU") ksig[f[2]] = ksig[f[2]] "\001" f[3] ":" f[4] ":" f[5] ":" f[6]
          else if (f[1] == "FK") fsig[f[2]] = f[3] "\001" f[4] "\001" f[5] }
        for (i = 2; i <= cnt; i++) { n = tabs(L[i], f)
          if (f[1] == "CON" && f[3] == "FOREIGN KEY") { c++; SK[c] = ksig[f[2]] "\002" fsig[f[2]] "\003" f[2]; SV[c] = f[2] } }
        srt(SK, SV, c)
        for (i = 1; i <= c; i++) { nn[SV[i]] = "fk#" i; nidx[SV[i]] = i }
      }
      print L[1]
      a = 2
      while (a <= cnt) {
        split(L[a], f, "\t"); tag = f[1]; k = a
        while (k + 1 <= cnt) { split(L[k+1], f, "\t"); if (f[1] != tag) break; k++ }
        run(tag, a, k); a = k + 1
      }
    }
    /^## / { section(); cnt = 0 }
    { L[++cnt] = $0 }
    END { section() }
  ' "$1"
}

# report_directives <e1 dump> <e2 dump> <fk tables> <order tables>: ALLOWED-DIFF for a directive
# whose difference is present in the two raw dumps, UNUSED-ALLOWED-DIFF when it is not.
report_directives() {
  local t
  proj() { awk -F'\t' -v t="## $1" -v m="$2" '$0==t{f=1;next} /^## /{f=0} f{ if (m=="fk") { if (($1=="CON"&&$3=="FOREIGN KEY")||$1=="KCU"||$1=="FK") print } else if ($1=="COL") print $2 ":" $3 }' "$3"; }
  for t in $3; do
    if [ "$(proj "$t" fk "$1")" != "$(proj "$t" fk "$2")" ]; then echo "ALLOWED-DIFF fk-name $t"; else echo "UNUSED-ALLOWED-DIFF fk-name $t"; fi
  done
  for t in $4; do
    if [ "$(proj "$t" col "$1")" != "$(proj "$t" col "$2")" ]; then echo "ALLOWED-DIFF column-order $t"; else echo "UNUSED-ALLOWED-DIFF column-order $t"; fi
  done
  return 0
}

# compare_e1 <e1 dump> <e2 dump> <list file>: the E1 versus E2 comparison with the accepted
# differences of the list (pair lines and directives). Canonical dumps go to separate files.
compare_e1() {
  local e1="$1" e2="$2" list="$3" fkt ordt rc=0
  fkt=$(directive_tables "$list" fk-name); ordt=$(directive_tables "$list" column-order)
  canon_dump "$e1" "$fkt" "$ordt" > "$WORK/e1.canon" || return 1
  canon_dump "$e2" "$fkt" "$ordt" > "$WORK/e2.canon" || return 1
  report_directives "$e1" "$e2" "$fkt" "$ordt"
  compare_dumps "$WORK/e1.canon" "$WORK/e2.canon" "$list" || rc=$?
  return $rc
}

# compare_dumps <reference> <actual> [<allowed diff file>]: 0 when the dumps are equal apart from
# the listed accepted differences.
compare_dumps() {
  local ref="$1" act="$2" allowed="${3:-}" line rc=0
  [ -s "$ref" ] && [ -f "$ref" ] || { echo "reference dump $ref is not a non-empty regular file"; return 1; }
  [ -s "$act" ] || { echo "dump $act is empty"; return 1; }
  if [ -n "$allowed" ]; then [ -f "$allowed" ] || { echo "allowed difference file $allowed is not a regular file"; return 1; }; fi
  diff "$ref" "$act" > "$WORK/diff.raw" 2>&1 || rc=$?
  [ "$rc" -le 1 ] || { echo "diff failed (rc=$rc)"; return 1; }
  grep -E '^[<>] ' "$WORK/diff.raw" | LC_ALL=C sort > "$WORK/diff.lines" || true
  : > "$WORK/allowed.lines"
  if [ -n "$allowed" ]; then grep -E '^[<>] ' "$allowed" | LC_ALL=C sort > "$WORK/allowed.lines" || true; fi
  LC_ALL=C comm -12 "$WORK/diff.lines" "$WORK/allowed.lines" > "$WORK/used.lines"
  LC_ALL=C comm -13 "$WORK/diff.lines" "$WORK/allowed.lines" > "$WORK/unused.lines"
  LC_ALL=C comm -23 "$WORK/diff.lines" "$WORK/allowed.lines" > "$WORK/rest.lines"
  while IFS= read -r line; do echo "ALLOWED-DIFF $line"; done < "$WORK/used.lines"
  while IFS= read -r line; do echo "UNUSED-ALLOWED-DIFF $line"; done < "$WORK/unused.lines"
  if [ -s "$WORK/rest.lines" ]; then cat "$WORK/rest.lines"; return 1; fi
  return 0
}

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
  local image tag cn d out base tagname
  TABLES="st_parent st_child"

  # normalization: only the character set prefix of the expression lines is removed
  [ "$(printf 'DEF\tc\t_utf8mb4\x27x\x27\n' | norm)" = "$(printf 'DEF\tc\t\x27x\x27\n')" ] && ok "norm removes a character set prefix in a default" || fail "norm prefix"
  local raw_a raw_b
  raw_a='CHK'$'\t''k'$'\t''(`c` = _latin1\\'"'"'x\\'"'"')'
  raw_b='CHK'$'\t''k'$'\t''(`c` = _utf8mb4\\'"'"'x\\'"'"')'
  [ "$(echo "$raw_a" | norm)" = "$(echo "$raw_b" | norm)" ] && [ "$(echo "$raw_a" | norm)" != "$raw_a" ] && ok "norm makes escaped prefixes of different sessions equal" || fail "norm escaped prefix"
  [ "$(printf 'DEF\tc\t\x27a_b\x27\n' | norm)" != "$(printf 'DEF\tc\t\x27a_c\x27\n' | norm)" ] && ok "norm keeps 'a_b' and 'a_c' different" || fail "norm over-normalizes"
  [ "$(printf 'COL\t1\tc\t_utf8mb4\x27x\x27\n' | norm)" = "$(printf 'COL\t1\tc\t_utf8mb4\x27x\x27\n')" ] && ok "norm leaves other lines unchanged" || fail "norm other lines"

  # reference dump comparison
  printf '## t\nCOL\t1\tid\nCOL\t2\tname\n' > "$WORK/ref.txt"
  cp "$WORK/ref.txt" "$WORK/same.txt"
  printf '## t\nCOL\t1\tid\nCOL\t2\tname2\n' > "$WORK/changed.txt"
  printf '< COL\t2\tname\n> COL\t2\tname2\n' > "$WORK/allowed.txt"
  compare_dumps "$WORK/ref.txt" "$WORK/same.txt" >/dev/null && ok "reference compare: identical dumps pass" || fail "reference compare identical"
  if compare_dumps "$WORK/ref.txt" "$WORK/changed.txt" >/dev/null; then fail "reference compare: a changed line passed"; else ok "reference compare: a changed line fails"; fi
  if out=$(compare_dumps "$WORK/ref.txt" "$WORK/changed.txt" "$WORK/allowed.txt") && echo "$out" | grep -q '^ALLOWED-DIFF < COL' && echo "$out" | grep -q '^ALLOWED-DIFF > COL'; then
    ok "reference compare: allowed differences pass and are printed"
  else
    fail "reference compare allowed"
  fi
  if compare_dumps "$WORK/does-not-exist.txt" "$WORK/same.txt" >/dev/null; then fail "reference compare: a missing reference file passed"; else ok "reference compare: a missing reference file fails"; fi
  if compare_dumps /tmp "$WORK/same.txt" >/dev/null; then fail "reference compare: a directory as reference passed"; else ok "reference compare: a directory as reference fails"; fi
  : > "$WORK/empty.txt"
  if compare_dumps "$WORK/empty.txt" "$WORK/same.txt" >/dev/null; then fail "reference compare: an empty reference passed"; else ok "reference compare: an empty reference fails"; fi
  printf '< COL\t9\tother\n' > "$WORK/allowed2.txt"
  if out=$(compare_dumps "$WORK/ref.txt" "$WORK/changed.txt" "$WORK/allowed2.txt"); then fail "reference compare: an unrelated allowance hid a difference"; else ok "reference compare: a difference outside the allowed list fails"; fi

  # E1_ALLOWED_DIFF: file check and the E1 style comparison
  local colu colz tblu tblz
  colu=$'COL\t2\tname\tvarchar(10)\tYES\t\tutf8mb4\tutf8mb4_uca1400_ai_ci\t'
  colz=$'COL\t2\tname\tvarchar(10)\tYES\t\tutf8mb4\tutf8mb4_0900_ai_ci\t'
  tblu=$'TBL\tInnoDB\tutf8mb4_uca1400_ai_ci\tDynamic\t\t'
  tblz=$'TBL\tInnoDB\tutf8mb4_0900_ai_ci\tDynamic\t\t'
  printf '# a comment\n< %s\n> %s\n< %s\n> %s\n' "$colu" "$colz" "$tblu" "$tblz" > "$WORK/e1ok.txt"
  check_allowed_file "$WORK/e1ok.txt" >/dev/null && ok "allowed file check: a collation only pair is accepted" || fail "allowed file check positive"
  check_allowed_file "$WORK/does-not-exist.txt" >/dev/null && fail "allowed file check: a missing file passed" || ok "allowed file check: a missing file fails"
  printf '# only a comment\n' > "$WORK/e1c.txt"
  check_allowed_file "$WORK/e1c.txt" >/dev/null && fail "allowed file check: a comment only file passed" || ok "allowed file check: a file without a diff line fails"
  printf '< FK\tfk_1\tCASCADE\tCASCADE\tt\n> FK\tfk_2\tCASCADE\tCASCADE\tt\n' > "$WORK/e1fk.txt"
  check_allowed_file "$WORK/e1fk.txt" >/dev/null && fail "allowed file check: an FK line passed" || ok "allowed file check: a tag other than TBL and COL fails"
  printf '< %s\r\n> %s\r\n' "$colu" "$colz" > "$WORK/e1cr.txt"
  check_allowed_file "$WORK/e1cr.txt" >/dev/null && fail "allowed file check: a CRLF file passed" || ok "allowed file check: a carriage return fails"
  printf '< %s\n> %s\n' "$colu" "${colz//varchar(10)/varchar(20)}" > "$WORK/e1type.txt"
  check_allowed_file "$WORK/e1type.txt" >/dev/null && fail "allowed file check: a type change passed" || ok "allowed file check: a pair that differs in the type fails"
  printf '< %s\n' "$colu" > "$WORK/e1one.txt"
  check_allowed_file "$WORK/e1one.txt" >/dev/null && fail "allowed file check: one sided line passed" || ok "allowed file check: a line without its pair fails"
  printf '<x\n> %s\n' "$colz" > "$WORK/e1bad.txt"
  check_allowed_file "$WORK/e1bad.txt" >/dev/null && fail "allowed file check: a malformed marker passed" || ok "allowed file check: a malformed marker fails"
  # two tables with the same lines: the accepted lines are needed twice
  printf '## a\n%s\n## b\n%s\n' "$colu" "$colu" > "$WORK/e1ref.txt"
  printf '## a\n%s\n## b\n%s\n' "$colz" "$colz" > "$WORK/e1act.txt"
  printf '< %s\n> %s\n' "$colu" "$colz" > "$WORK/e1once.txt"
  printf '< %s\n> %s\n< %s\n> %s\n' "$colu" "$colz" "$colu" "$colz" > "$WORK/e1twice.txt"
  if compare_dumps "$WORK/e1ref.txt" "$WORK/e1act.txt" "$WORK/e1once.txt" >/dev/null; then fail "E1 compare: a line listed once covered two tables"; else ok "E1 compare: a line that appears for two tables must be listed twice"; fi
  if out=$(compare_dumps "$WORK/e1ref.txt" "$WORK/e1act.txt" "$WORK/e1twice.txt") && [ "$(echo "$out" | grep -c '^ALLOWED-DIFF ')" = 4 ]; then ok "E1 compare: the lines listed twice are accepted and printed"; else fail "E1 compare twice"; fi

  # E1_ALLOWED_DIFF directives (fk-name, column-order): file check, canon_dump and the E1 comparison
  printf '# directives only\nfk-name st_child\ncolumn-order st_parent\n\n' > "$WORK/dok.txt"
  check_allowed_file "$WORK/dok.txt" >/dev/null && ok "allowed file check: directives only are accepted" || fail "allowed file check directives"
  printf 'fk-name st_other\n' > "$WORK/dtab.txt"
  check_allowed_file "$WORK/dtab.txt" >/dev/null && fail "allowed file check: an unknown table passed" || ok "allowed file check: a directive for an unknown table fails"
  printf 'fk-names st_child\n' > "$WORK/dtypo.txt"
  check_allowed_file "$WORK/dtypo.txt" >/dev/null && fail "allowed file check: a misspelled directive passed" || ok "allowed file check: a misspelled directive fails"
  printf 'fk-name st_child\nfk-name st_child\n' > "$WORK/ddup.txt"
  check_allowed_file "$WORK/ddup.txt" >/dev/null && fail "allowed file check: a duplicate directive passed" || ok "allowed file check: a duplicate directive fails"
  printf 'some free text\nfk-name st_child\n' > "$WORK/dfree.txt"
  check_allowed_file "$WORK/dfree.txt" >/dev/null && fail "allowed file check: free text passed" || ok "allowed file check: free text instead of a comment fails"
  printf 'fk-name st_child extra\n' > "$WORK/dextra.txt"
  check_allowed_file "$WORK/dextra.txt" >/dev/null && fail "allowed file check: a directive with a second table passed" || ok "allowed file check: a directive with extra words fails"
  printf 'fk-name st_child\r\n' > "$WORK/dcr.txt"
  check_allowed_file "$WORK/dcr.txt" >/dev/null && fail "allowed file check: a CRLF directive passed" || ok "allowed file check: a CRLF directive fails"

  # mkd <fk1> <fk2> <rule1> <rule2> <order a|b|c|d> <g type> <g expression>: synthetic dump of a parent and a
  # child table (two foreign keys, two generated columns g and h). order a: id a_id b_id g h; b: id h g a_id b_id;
  # c: a plus column x; d: a without b_id.
  # An eighth argument "same" puts the second key on a_id as well.
  mkd() {
    local cols n=0 c ty ex
    case "$5" in a) cols="id a_id b_id g h";; b) cols="id h g a_id b_id";; c) cols="id a_id b_id g h x";; d) cols="id a_id g h";; esac
    printf '## st_parent\nCOL\t1\tid\tbinary(16)\tNO\t\t\t\t\nDEF\tid\t<none>\nIDX\tPRIMARY\t1\tid\t0\t\tBTREE\tA\t\nTBL\tInnoDB\tutf8mb4_uca1400_ai_ci\tDynamic\t\t\nCON\tPRIMARY\tPRIMARY KEY\n## st_child\n'
    for c in $cols; do n=$((n+1)); ty='binary(16)'; ex=''; [ "$c" = g ] && { ty="$6"; ex='STORED GENERATED'; }; [ "$c" = h ] && ex='STORED GENERATED';
      printf 'COL\t%s\t%s\t%s\tYES\t%s\t\t\t\n' "$n" "$c" "$ty" "$ex"; done
    for c in $cols; do printf 'DEF\t%s\tNULL\n' "$c"; done
    for c in $cols; do [ "$c" = g ] && printf 'GEN\tg\t%s\n' "$7"; [ "$c" = h ] && printf 'GEN\th\t(`id` + 3)\n'; done
    printf 'IDX\tPRIMARY\t1\tid\t0\t\tBTREE\tA\t\nTBL\tInnoDB\tutf8mb4_uca1400_ai_ci\tDynamic\t\t\n'
    printf 'CON\tPRIMARY\tPRIMARY KEY\nCON\t%s\tFOREIGN KEY\nCON\t%s\tFOREIGN KEY\n' "$1" "$2" | LC_ALL=C sort -s -t$'\t' -k2,2
    printf 'KCU\t%s\t1\ta_id\tst_parent\tid\nKCU\t%s\t1\t%s\tst_parent\tid\n' "$1" "$2" "$([ "${8:-}" = same ] && echo a_id || echo b_id)" | LC_ALL=C sort -s -t$'\t' -k2,2
    printf 'FK\t%s\tRESTRICT\t%s\tst_parent\nFK\t%s\tRESTRICT\t%s\tst_parent\n' "$1" "$3" "$2" "$4" | LC_ALL=C sort -s -t$'\t' -k2,2
  }
  local ge='(`id` + 1)'
  printf 'fk-name st_child\n' > "$WORK/lfk.txt"; printf 'column-order st_child\n' > "$WORK/lord.txt"
  printf 'fk-name st_child\nfk-name st_parent\n' > "$WORK/ltwo.txt"
  printf 'fk-name st_child\ncolumn-order st_child\n' > "$WORK/lboth.txt"; printf 'fk-name st_parent\n' > "$WORK/lparent.txt"
  mkd 1 2 CASCADE 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_unnamed.txt"
  mkd st_child_ibfk_1 st_child_ibfk_2 CASCADE 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_named.txt"
  canon_dump "$WORK/d_unnamed.txt" "" "" | cmp -s - "$WORK/d_unnamed.txt" && ok "canon_dump with no listed table is the identity" || fail "canon_dump identity"
  if compare_dumps "$WORK/d_unnamed.txt" "$WORK/d_named.txt" >/dev/null; then fail "E1 compare: renamed foreign keys passed without a directive"; else ok "E1 compare: renamed foreign keys fail without a directive"; fi
  if out=$(compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named.txt" "$WORK/lfk.txt") && [ "$out" = "ALLOWED-DIFF fk-name st_child" ]; then ok "E1 compare: renamed foreign keys pass with fk-name and are reported"; else fail "E1 compare fk-name: $out"; fi
  if out=$(compare_e1 "$WORK/d_named.txt" "$WORK/d_named.txt" "$WORK/lfk.txt") && [ "$out" = "UNUSED-ALLOWED-DIFF fk-name st_child" ]; then ok "E1 compare: equal names leave fk-name unused"; else fail "E1 compare fk-name unused: $out"; fi
  if out=$(compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named.txt" "$WORK/ltwo.txt") && [ "$out" = "$(printf 'ALLOWED-DIFF fk-name st_child\nUNUSED-ALLOWED-DIFF fk-name st_parent')" ]; then ok "E1 compare: fk-name for two tables is reported per table"; else fail "E1 compare two tables: $out"; fi
  if compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named.txt" "$WORK/lparent.txt" >/dev/null; then fail "E1 compare: fk-name for another table hid a rename"; else ok "E1 compare: fk-name for another table does not accept the rename"; fi
  mkd k1 k2 CASCADE 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_k12.txt"; mkd k2 k1 CASCADE 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_k21.txt"
  if out=$(compare_e1 "$WORK/d_k12.txt" "$WORK/d_k21.txt" "$WORK/lfk.txt") && [ "$out" = "ALLOWED-DIFF fk-name st_child" ]; then ok "E1 compare: a swap of two key names passes and is reported as used"; else fail "E1 compare swap: $out"; fi
  mkd st_child_ibfk_1 st_child_ibfk_2 RESTRICT 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_rule.txt"
  if compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_rule.txt" "$WORK/lfk.txt" >/dev/null; then fail "E1 compare: a renamed key with another ON DELETE rule passed"; else ok "E1 compare: a rename plus a changed ON DELETE rule fails"; fi
  mkd k1 k2 'SET NULL' CASCADE a binary\(16\) "$ge" > "$WORK/d_ruleswap.txt"
  if compare_e1 "$WORK/d_k12.txt" "$WORK/d_ruleswap.txt" "$WORK/lfk.txt" >/dev/null; then fail "E1 compare: a swap of the rules of two keys passed"; else ok "E1 compare: a swap of the rules between two keys fails"; fi
  mkd k1 k2 CASCADE 'SET NULL' a binary\(16\) "$ge" same > "$WORK/d_same1.txt"; mkd k1 k2 'SET NULL' CASCADE a binary\(16\) "$ge" same > "$WORK/d_same2.txt"
  if out=$(compare_e1 "$WORK/d_same1.txt" "$WORK/d_same2.txt" "$WORK/lfk.txt") && [ "$out" = "ALLOWED-DIFF fk-name st_child" ]; then ok "E1 compare: two keys on the same columns with swapped names pass (rules stay coupled to the key)"; else fail "E1 compare same columns: $out"; fi
  if compare_e1 "$WORK/d_k12.txt" "$WORK/d_same1.txt" "$WORK/lfk.txt" >/dev/null; then fail "E1 compare: a foreign key on another column passed"; else ok "E1 compare: a foreign key on another column fails"; fi
  mkd 1 2 CASCADE 'SET NULL' b binary\(16\) "$ge" > "$WORK/d_order_b.txt"; mkd 1 2 CASCADE 'SET NULL' a binary\(16\) "$ge" > "$WORK/d_order_a.txt"
  if compare_dumps "$WORK/d_order_a.txt" "$WORK/d_order_b.txt" >/dev/null; then fail "E1 compare: another column order passed without a directive"; else ok "E1 compare: another column order fails without a directive"; fi
  if out=$(compare_e1 "$WORK/d_order_a.txt" "$WORK/d_order_b.txt" "$WORK/lord.txt") && [ "$out" = "ALLOWED-DIFF column-order st_child" ]; then ok "E1 compare: another column order passes with column-order and is reported"; else fail "E1 compare column-order: $out"; fi
  if out=$(compare_e1 "$WORK/d_order_a.txt" "$WORK/d_order_a.txt" "$WORK/lord.txt") && [ "$out" = "UNUSED-ALLOWED-DIFF column-order st_child" ]; then ok "E1 compare: an equal column order leaves column-order unused"; else fail "E1 compare column-order unused: $out"; fi
  if compare_e1 "$WORK/d_order_a.txt" "$WORK/d_order_b.txt" "$WORK/lfk.txt" >/dev/null; then fail "E1 compare: fk-name accepted a column order difference"; else ok "E1 compare: fk-name does not accept a column order difference"; fi
  if compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: column-order accepted a foreign key rename"; else ok "E1 compare: column-order does not accept a foreign key rename"; fi
  mkd 1 2 CASCADE 'SET NULL' b 'binary(32)' "$ge" > "$WORK/d_type.txt"
  if compare_e1 "$WORK/d_order_a.txt" "$WORK/d_type.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: a reorder with a changed column type passed"; else ok "E1 compare: a reorder with a changed column type fails"; fi
  sed 's/^DEF\ta_id\tNULL$/DEF\ta_id\t0/' "$WORK/d_order_b.txt" > "$WORK/d_default.txt"
  if cmp -s "$WORK/d_default.txt" "$WORK/d_order_b.txt" || compare_e1 "$WORK/d_order_a.txt" "$WORK/d_default.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: a reorder with a changed column default passed"; else ok "E1 compare: a reorder with a changed column default fails"; fi
  mkd 1 2 CASCADE 'SET NULL' b binary\(16\) '(`id` + 2)' > "$WORK/d_expr.txt"
  if compare_e1 "$WORK/d_order_a.txt" "$WORK/d_expr.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: a reorder with a changed generation expression passed"; else ok "E1 compare: a reorder with a changed generation expression fails"; fi
  mkd 1 2 CASCADE 'SET NULL' c binary\(16\) "$ge" > "$WORK/d_added.txt"; mkd 1 2 CASCADE 'SET NULL' d binary\(16\) "$ge" > "$WORK/d_dropped.txt"
  if compare_e1 "$WORK/d_order_a.txt" "$WORK/d_added.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: an added column passed"; else ok "E1 compare: an added column fails"; fi
  if compare_e1 "$WORK/d_order_a.txt" "$WORK/d_dropped.txt" "$WORK/lord.txt" >/dev/null; then fail "E1 compare: a dropped column passed"; else ok "E1 compare: a dropped column fails"; fi
  mkd st_child_ibfk_1 st_child_ibfk_2 CASCADE 'SET NULL' b binary\(16\) "$ge" > "$WORK/d_named_b.txt"
  if out=$(compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named_b.txt" "$WORK/lboth.txt") && [ "$(echo "$out" | grep -c '^ALLOWED-DIFF ')" = 2 ]; then ok "E1 compare: a rename and a reorder together pass with both directives"; else fail "E1 compare both: $out"; fi
  if compare_e1 "$WORK/d_unnamed.txt" "$WORK/d_named_b.txt" "$WORK/lfk.txt" >/dev/null; then fail "E1 compare: a rename and a reorder passed with fk-name only"; else ok "E1 compare: a rename and a reorder fail with fk-name only"; fi

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

DUMP_ONLY=0
if [ "${1:-}" = "--dump" ]; then DUMP_ONLY=1; shift; fi

SERVICE="${1:?usage: $0 <service-dir, e.g. bin-tag-manager> [<migrations-subdir>] | --dump <service-dir> <database> [<migrations-subdir>] | --self-test}"
if [ "$DUMP_ONLY" = 1 ]; then DUMP_DB="${2:?usage: $0 --dump <service-dir> <database> [<migrations-subdir>]}"; MIG_REL="${3:-migrations}"; else MIG_REL="${2:-migrations}"; fi
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

if [ "$DUMP_ONLY" = 1 ]; then
  sql() { ${SQL_CMD:-mysql -N -B} "$@"; }
  dump_db "$DUMP_DB" > "$WORK/dump.txt"
  dump_complete "$WORK/dump.txt" || exit 1
  cat "$WORK/dump.txt"
  exit
fi

if [ -n "${ALLOWED_DIFF:-}" ] && [ -z "${REFERENCE_DUMP:-}" ]; then echo "ALLOWED_DIFF is set without REFERENCE_DUMP (E1_ALLOWED_DIFF is the list of the Alembic head comparison)"; exit 1; fi
if [ -n "${E1_ALLOWED_DIFF:-}" ]; then
  E1_MSG=$(check_allowed_file "$E1_ALLOWED_DIFF") || { echo "E1_ALLOWED_DIFF: $E1_MSG"; exit 1; }
fi
if [ -n "${REFERENCE_DUMP:-}" ]; then
  [ -n "${REFERENCE_ENGINE:-}" ] || { echo "REFERENCE_DUMP needs REFERENCE_ENGINE (the engine image the dump was made with)"; exit 1; }
  [ -s "$REFERENCE_DUMP" ] && [ -f "$REFERENCE_DUMP" ] || { echo "REFERENCE_DUMP $REFERENCE_DUMP is not a non-empty regular file"; exit 1; }
  if [ -n "${ALLOWED_DIFF:-}" ] && [ ! -f "$ALLOWED_DIFF" ]; then echo "ALLOWED_DIFF $ALLOWED_DIFF is not a regular file"; exit 1; fi
  [[ " ${ENGINES[*]} " == *" $REFERENCE_ENGINE "* ]] || { echo "REFERENCE_ENGINE $REFERENCE_ENGINE is not one of the engines being run: ${ENGINES[*]}"; exit 1; }
fi
: "${ALEMBIC_VENV:?set ALEMBIC_VENV to a venv with alembic 1.11.3, SQLAlchemy 1.4.52, PyMySQL 1.1.0}"

# expected version: the highest up file number
N=$(ls "$MIG_DIR"/*.up.sql | sed -E 's#.*/([0-9]{6})_.*#\1#' | sort -n | tail -1 | sed -E 's/^0+//')
[ -n "$N" ] || { echo "no migration files in $MIG_DIR"; exit 1; }
if [ -f "$MIG_DIR/embed.go" ]; then
  grep -q "VersionTable = \"$VERSION_TABLE\"" "$MIG_DIR/embed.go" || { echo "migrations.VersionTable is not $VERSION_TABLE"; exit 1; }
else
  skip "no embed.go in $MIG_REL: the version table name is not checked"
fi
# a destructive migration (marker line) is exempt from the row count check
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
  for db in e1 e2 e3 e4 e5 e6; do sql -e "CREATE DATABASE $db CHARACTER SET utf8 COLLATE utf8_general_ci"; done
  dsn() { echo "root:$PASS@tcp(127.0.0.1:$PORT)/$1?parseTime=true&loc=Local"; }
  ctl() { local db="$1"; shift; env "$DSN_VAR=$(dsn "$db")" "$WORK/$CONTROL" "$MIGRATE_CMD" "$@"; }
  version_of() { ctl "$1" status | awk -F': ' '/Current version/{print $2}'; }
  rowcounts() { local t; for t in $TABLES; do echo "$t $(sql -e "SELECT COUNT(*) FROM $1.$t")"; done; }
  seed_of() { sql -e "${SEED_COMPARE_SQL//@DB@/$1}"; }

  # E1: Alembic head
  ALE="$WORK/alembic-$TAG"; cp -r "$ROOT/bin-dbscheme-manager/bin-manager" "$ALE"
  sed "s#^sqlalchemy.url = .*#sqlalchemy.url = mysql+pymysql://root:$PASS@127.0.0.1:$PORT/e1#" "$ALE/alembic.ini.sample" > "$ALE/alembic.ini"
  (cd "$ALE" && "$ALEMBIC_VENV/bin/alembic" -c alembic.ini upgrade head >/dev/null 2>"$WORK/alembic.err") || { cat "$WORK/alembic.err"; fail "$IMAGE alembic upgrade head"; continue; }
  ok "$IMAGE E1 built by the Alembic head: $(sql -e "SELECT version_num FROM e1.alembic_version")"
  dump_db e1 > "$WORK/e1-before.txt"

  # V2 existing database with data
  if [ -n "${EXISTING_ROWS_SQL:-}" ]; then
    sql -D e1 < "$EXISTING_ROWS_SQL"
  elif [ "$FIRST_TABLE" = "tag_tags" ]; then
    ID='0x00112233445566778899aabbccddeeff'
    sql -e "INSERT INTO e1.tag_tags (id, customer_id, name) VALUES ($ID, $ID, 'existing')"
  else
    skip "$IMAGE V2 no rows inserted: set EXISTING_ROWS_SQL for $SERVICE"
  fi
  rowcounts e1 > "$WORK/rows-before.txt"
  ctl e1 up >/dev/null
  dump_db e1 > "$WORK/e1-after.txt"
  rowcounts e1 > "$WORK/rows-after.txt"
  [ "$(version_of e1)" = "$N" ] && ok "$IMAGE V2 existing database migrated to version $N" || fail "$IMAGE V2 version"
  if [ "$DESTRUCTIVE" = 1 ]; then
    skip "$IMAGE V2 row count (DESTRUCTIVE migration)"
  elif diff -q "$WORK/rows-before.txt" "$WORK/rows-after.txt" >/dev/null; then
    ok "$IMAGE V2 row counts unchanged ($(tr '\n' ';' < "$WORK/rows-after.txt"))"
  else
    fail "$IMAGE V2 row counts changed"
  fi
  if [ "$N" = 1 ]; then
    if diff -q "$WORK/e1-before.txt" "$WORK/e1-after.txt" >/dev/null; then ok "$IMAGE V2 dump of the existing database identical"; else fail "$IMAGE V2 dump changed"; fi
  else
    skip "$IMAGE V2 baseline-only dump check (N=$N>1)"
  fi

  # V1/V3 empty database, dump equals E1 with the service migrations applied
  ctl e2 up >/dev/null
  dump_db e2 > "$WORK/e2.txt"
  if [ -n "${E1_ALLOWED_DIFF:-}" ]; then
    cmp_rc=0
    compare_e1 "$WORK/e1-after.txt" "$WORK/e2.txt" "$E1_ALLOWED_DIFF" >"$WORK/cmp.out" || cmp_rc=$?
    cat "$WORK/cmp.out"
    if [ "$cmp_rc" = 0 ]; then ok "$IMAGE V1/V3 normalized dump E1 vs E2: diff 0 apart from E1_ALLOWED_DIFF ($(grep -c '^ALLOWED-DIFF ' "$WORK/cmp.out" || true) accepted lines used, $(grep -c '^UNUSED-ALLOWED-DIFF ' "$WORK/cmp.out" || true) unused)"; else fail "$IMAGE V3 dump differs outside E1_ALLOWED_DIFF"; fi
  elif diff "$WORK/e1-after.txt" "$WORK/e2.txt" >"$WORK/diff.txt"; then ok "$IMAGE V1/V3 normalized dump E1 vs E2: diff 0 ($(wc -l < "$WORK/e2.txt") lines)"; else cat "$WORK/diff.txt"; fail "$IMAGE V3 dump differs"; fi
  [ "$(version_of e2)" = "$N" ] && ok "$IMAGE V1 version $N" || fail "$IMAGE V1 version"
  if dump_complete "$WORK/e1-after.txt" && dump_complete "$WORK/e2.txt"; then :; else fail "$IMAGE V3 a dump holds no columns"; fi
  if [ -n "${REFERENCE_DUMP:-}" ]; then
    if [ "$IMAGE" = "$REFERENCE_ENGINE" ]; then
      REFERENCE_USED=1
      if compare_dumps "$REFERENCE_DUMP" "$WORK/e2.txt" "${ALLOWED_DIFF:-}"; then ok "$IMAGE V3 reference dump vs E2: no difference outside the allowed list"; else fail "$IMAGE V3 reference dump differs"; fi
    else
      skip "$IMAGE V3 reference dump (it was made with $REFERENCE_ENGINE)"
    fi
  fi
  if [ -n "${SEED_COMPARE_SQL:-}" ]; then
    if [ "$(seed_of e1)" = "$(seed_of e2)" ] && [ -n "$(seed_of e2)" ]; then ok "$IMAGE V3 seed rows of E1 and E2 are equal"; else fail "$IMAGE V3 seed rows differ"; fi
  fi

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
  env "$DSN_VAR=$(dsn e6)" "${DAEMON_KV[@]}" timeout 60 "$WORK/$SERVICE_BIN" >"$WORK/daemon.out" 2>&1 || rc=$?
  if [ "$rc" = 0 ] || [ "$rc" = 124 ]; then
    fail "$IMAGE V1 daemon did not exit by itself with an error (rc=$rc)"
  elif [ "$(version_of e6)" = "$N" ] && [ "$(sql -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='e6' AND TABLE_NAME='$FIRST_TABLE'")" = "1" ]; then
    ok "$IMAGE V1 daemon start migrated an empty database before connecting the cache"
  else
    tail -5 "$WORK/daemon.out"; fail "$IMAGE V1 daemon start did not migrate"
  fi

  # Order on a new database: Alembic first, then the service. The reverse order is unsupported:
  # the Alembic history creates the tables and fails when the service already created them.
  sql -e "CREATE DATABASE e7 CHARACTER SET utf8 COLLATE utf8_general_ci"
  ctl e7 up >/dev/null
  sed "s#/e1\$#/e7#" "$ALE/alembic.ini" > "$ALE/alembic-e7.ini"
  if (cd "$ALE" && "$ALEMBIC_VENV/bin/alembic" -c alembic-e7.ini upgrade head >/dev/null 2>"$WORK/alembic-e7.err"); then
    fail "$IMAGE Alembic after the service unexpectedly succeeded"
  elif grep -qi "already exists" "$WORK/alembic-e7.err" && grep -qE "$(echo "$TABLES" | paste -sd'|')" "$WORK/alembic-e7.err"; then
    ok "$IMAGE service-first then Alembic fails with 'already exists' (unsupported order, documented)"
  else
    tail -3 "$WORK/alembic-e7.err"; fail "$IMAGE service-first then Alembic failed for another reason"
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
[ "${REFERENCE_USED:-0}" = 1 ] || echo "== reference dump not given: the result is not compared with an existing database (REFERENCE_DUMP)"
[ "${HELPER_TESTS:-0}" = 1 ] || echo "== migrationhandler integration tests not run (HELPER_TESTS=1 is required for the helper change evidence)"
echo "== failures: $FAILS, skipped: $SKIPS"
[ "$FAILS" = 0 ]
