#!/usr/bin/env bash
# Applies the service-owned migrations of the changed services to a database created with the
# engine default collation (VOIP-1687).
#
# Why: a migration that compares a function result with a literal fails with Error 1267 on a
# MariaDB 12.x database whose character set is utf8 and whose collation is not named
# (default utf8mb3_uca1400_ai_ci), while a database created with an explicit utf8_general_ci
# passes (VOIP-1686). This is step V13 of scripts/verify-service-migrations.sh, run in CI.
#
# Per target (a service migrations directory, plus bin-timeline-manager@mysql):
#   create database (CHARACTER SET utf8 only) -> assert the default collation is uca1400 ->
#   build <name>-control -> up (version N) -> N x down (version 0) -> up (version N).
#   A target with a "-- DESTRUCTIVE:" marker in an *.up.sql only does the first up.
#
# Targets: the changed targets of the branch (git diff against the merge base with origin/main,
# or HEAD~1 on main). When nothing maps to a target the smoke target bin-tag-manager runs, so a
# change of this script or of the job definition exercises the job. When the changed files cannot
# be determined every target runs (fail safe).
#
# Environment (defaults in brackets):
#   TARGETS         space separated target ids replacing the git scan [none]; no smoke fallback
#   LIST_ONLY       1 prints one line per target after the validation and exits [0]
#   DB_HOST DB_PORT DB_USER DB_PASSWORD  [127.0.0.1 3306 root empty]
#   ALLOW_REMOTE_DB 1 allows a DB_HOST that is not loopback [0]
#   CIRCLE_BRANCH   main selects the HEAD~1 range [unset]
set -uo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "not inside a git repository"; exit 1; }
cd "$ROOT" || exit 1

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-}"
SMOKE_TARGET="bin-tag-manager"
TIMELINE_MYSQL="bin-timeline-manager@mysql"
NL=$'\n'

WORK="$(mktemp -d)"
CREATED_DBS=()
MYSQL=()
trap 'cleanup' EXIT

cleanup() {
  local db
  if [ "${#MYSQL[@]}" -gt 0 ]; then
    for db in "${CREATED_DBS[@]:-}"; do
      [ -n "$db" ] && "${MYSQL[@]}" -e "DROP DATABASE IF EXISTS \`$db\`" >/dev/null 2>&1
    done
  fi
  rm -rf "$WORK"
}

# ---------------------------------------------------------------- target resolution

# target_spec <id>: sets SVC SUBDIR CMD NAME for a target id.
target_spec() {
  case "$1" in
    "$TIMELINE_MYSQL") SVC="bin-timeline-manager"; SUBDIR="migrations/mysql"; CMD="migrate-mysql" ;;
    *) SVC="$1"; SUBDIR="migrations"; CMD="migrate" ;;
  esac
  NAME="${SVC#bin-}"; NAME="${NAME%-manager}"
}

# excluded_reason <service-dir>: prints a reason when the service is not a MariaDB target.
excluded_reason() {
  case "$1" in
    bin-rag-manager) echo "bin-rag-manager is PostgreSQL" ;;
    bin-timeline-manager) echo "bin-timeline-manager migrations/ (outside mysql/) is ClickHouse" ;;
  esac
}

# path_to_target <path>: prints the target id, or "skip:<reason>", or nothing for a path outside migrations.
path_to_target() {
  local p="$1" svc rest
  [[ "$p" =~ ^(bin-[^/]+)/migrations/(.+)$ ]] || return 0
  svc="${BASH_REMATCH[1]}"; rest="${BASH_REMATCH[2]}"
  if [ "$svc" = "bin-timeline-manager" ] && [[ "$rest" == mysql/* ]]; then
    echo "$TIMELINE_MYSQL"
  elif [ -n "$(excluded_reason "$svc")" ]; then
    echo "skip:$(excluded_reason "$svc")"
  else
    echo "$svc"
  fi
}

# changed_files: writes the NUL separated paths changed by the branch to $WORK/changed.z; non-zero when it cannot be determined.
changed_files() {
  local out="$WORK/changed.z"
  if [ "${CIRCLE_BRANCH:-}" = "main" ]; then
    git rev-parse -q --verify 'HEAD~1' >/dev/null 2>&1 || git fetch --no-tags --deepen=1 origin main >/dev/null 2>&1
    git rev-parse -q --verify 'HEAD~1' >/dev/null 2>&1 || return 1
    git diff -z --no-renames --name-only 'HEAD~1' HEAD >"$out" 2>/dev/null || return 1
  else
    git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main' >/dev/null 2>&1
    if ! git merge-base origin/main HEAD >/dev/null 2>&1; then
      echo "Merge base unreachable; deepening the clone." >&2
      git fetch --no-tags --unshallow origin '+refs/heads/main:refs/remotes/origin/main' >/dev/null 2>&1 \
        || git fetch --no-tags --deepen=1000 origin '+refs/heads/main:refs/remotes/origin/main' >/dev/null 2>&1
    fi
    git merge-base origin/main HEAD >/dev/null 2>&1 || return 1
    git diff -z --no-renames --name-only origin/main...HEAD >"$out" 2>/dev/null || return 1
  fi
}

# all_targets: every migrations directory of the repository.
all_targets() {
  local d svc
  for d in bin-*/migrations; do
    [ -d "$d" ] || continue
    svc="${d%/migrations}"
    if [ "$svc" = "bin-timeline-manager" ]; then
      echo "skip:$(excluded_reason "$svc")"
      [ -d "$d/mysql" ] && echo "$TIMELINE_MYSQL"
    elif [ -n "$(excluded_reason "$svc")" ]; then
      echo "skip:$(excluded_reason "$svc")"
    else
      echo "$svc"
    fi
  done
}

# resolve_targets: fills TARGET_LIST (unique, sorted) and prints the skipped lines.
resolve_targets() {
  local raw="" t line
  TARGET_LIST=()
  if [ -n "${TARGETS:-}" ]; then
    for t in $TARGETS; do
      if [[ "$t" == "$TIMELINE_MYSQL" ]]; then raw+="$t$NL"
      elif [ -n "$(excluded_reason "$t")" ]; then raw+="skip:$(excluded_reason "$t")$NL"
      else raw+="$t$NL"; fi
    done
  else
    local p
    if changed_files; then
      while IFS= read -r -d '' p; do
        t="$(path_to_target "$p")"
        [ -n "$t" ] && raw+="$t$NL"
      done <"$WORK/changed.z"
    else
      echo "changed files unknown; checking every target"
      raw="$(all_targets)$NL"
    fi
  fi
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    if [[ "$line" == skip:* ]]; then
      echo "skipped: ${line#skip:}"
    else
      TARGET_LIST+=("$line")
    fi
  done < <(printf '%s' "$raw" | sort -u)

  local kept=() id
  for id in "${TARGET_LIST[@]:-}"; do
    [ -z "$id" ] && continue
    target_spec "$id"
    if [ -d "$SVC/$SUBDIR" ]; then kept+=("$id"); else echo "skipped: $id was removed"; fi
  done
  TARGET_LIST=("${kept[@]:-}")
  [ "${#TARGET_LIST[@]}" -eq 1 ] && [ -z "${TARGET_LIST[0]}" ] && TARGET_LIST=()
  if [ "${#TARGET_LIST[@]}" -eq 0 ] && [ -z "${TARGETS:-}" ]; then
    echo "no migration target changed; smoke target $SMOKE_TARGET"
    TARGET_LIST=("$SMOKE_TARGET")
  fi
}

# ---------------------------------------------------------------- validation

# validate_target <id>: sets SVC SUBDIR CMD NAME N DESTRUCTIVE; returns 1 with VALIDATE_MSG on a violation.
validate_target() {
  local dir files count seq
  target_spec "$1"
  dir="$SVC/$SUBDIR"
  N=0; DESTRUCTIVE=0; VALIDATE_MSG=""
  [ -f "$dir/embed.go" ] || { VALIDATE_MSG="$dir/embed.go is missing"; return 1; }
  [ -d "$SVC/cmd/$NAME-control" ] || { VALIDATE_MSG="$SVC/cmd/$NAME-control is missing"; return 1; }
  files="$(ls "$dir"/*.up.sql 2>/dev/null | sed -E 's#.*/([0-9]{6})_[^/]*$#\1#' | sort)"
  count="$(printf '%s' "$files" | grep -c . || true)"
  [ "$count" -ge 1 ] || { VALIDATE_MSG="no *.up.sql in $dir"; return 1; }
  seq="$(seq -f '%06g' 1 "$count")"
  [ "$files" = "$seq" ] || { VALIDATE_MSG="the numbers of $dir/*.up.sql are not 000001..$(printf '%06d' "$count") without gaps"; return 1; }
  N="$count"
  if grep -lq '^-- DESTRUCTIVE:' "$dir"/*.up.sql 2>/dev/null; then DESTRUCTIVE=1; fi
  return 0
}

# ---------------------------------------------------------------- execution

dsn_host() { if [ "$DB_HOST" = "::1" ]; then echo "[::1]"; else echo "$DB_HOST"; fi; }

preflight() {
  local c
  for c in mysql mariadb; do
    if command -v "$c" >/dev/null 2>&1; then MYSQL=("$c" --protocol=tcp -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER"); break; fi
  done
  [ "${#MYSQL[@]}" -gt 0 ] || { echo "no mysql or mariadb client found"; return 1; }
  command -v go >/dev/null 2>&1 || { echo "go not found"; return 1; }
  case "$DB_HOST" in
    127.0.0.1|localhost|::1) ;;
    *) [ "${ALLOW_REMOTE_DB:-0}" = 1 ] || { echo "DB_HOST $DB_HOST is not loopback; set ALLOW_REMOTE_DB=1 to allow it"; return 1; } ;;
  esac
  if [ -n "$DB_PASSWORD" ]; then export MYSQL_PWD="$DB_PASSWORD"; else unset MYSQL_PWD; fi
  local deadline=$((SECONDS + 120))
  until "${MYSQL[@]}" -e 'SELECT 1' >/dev/null 2>&1; do
    if [ "$SECONDS" -ge "$deadline" ]; then echo "the database did not accept connections in 120 seconds"; return 1; fi
    sleep 2
  done
}

# step_fail <id> <step> <logfile>
step_fail() {
  echo "FAIL  $1 $2"
  [ -f "$3" ] && tail -5 "$3" | sed 's/^/        /'
  FAILED=$((FAILED + 1))
}

run_target() {
  local id="$1" db log bin dsn i v
  validate_target "$id" || return 1   # sets SVC SUBDIR CMD NAME N DESTRUCTIVE for this target
  db="c_$(echo "${id#bin-}" | sed -E 's/-manager//; s/[^a-z0-9]+/_/g')"
  log="$WORK/$db.log"; : >"$log"
  bin="$WORK/$db/$NAME-control"; mkdir -p "$WORK/$db"
  dsn="$DB_USER:$DB_PASSWORD@tcp($(dsn_host):$DB_PORT)/$db?parseTime=true&loc=Local"

  ctl() {
    env DATABASE_DSN="$dsn" DATABASE_DSN_BIN="$dsn" \
      DOMAIN_NAME_EXTENSION=registrar.example.com DOMAIN_NAME_TRUNK=trunk.example.com \
      "$bin" "$CMD" "$@"
  }
  version_of() { ctl status 2>>"$log" | awk -F': ' '/Current version/{print $2}'; }

  if ! "${MYSQL[@]}" -e "CREATE DATABASE \`$db\` CHARACTER SET utf8" >"$log" 2>&1; then
    step_fail "$id" create "$log"
    echo "        (database $db may be left from a former run: DROP DATABASE \`$db\`)"
    return
  fi
  CREATED_DBS+=("$db")

  local coll
  coll="$("${MYSQL[@]}" -N -B -e "SELECT DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='$db'" 2>>"$log")"
  case "$coll" in
    *uca1400*) ;;
    *) echo "default collation is '$coll', not uca1400: the check would not see a default-collation failure" >>"$log"; step_fail "$id" precondition "$log"; return ;;
  esac

  if ! (cd "$SVC" && go build -o "$bin" "./cmd/$NAME-control") >>"$log" 2>&1; then step_fail "$id" build "$log"; return; fi

  if ! ctl up >>"$log" 2>&1; then step_fail "$id" up "$log"; return; fi
  v="$(version_of)"
  if [ "$v" != "$N" ]; then echo "version after up is '$v', expected $N" >>"$log"; step_fail "$id" version "$log"; return; fi

  if [ "$DESTRUCTIVE" = 1 ]; then
    echo "PASS  $id up ($N, default collation $coll)"
    echo "SKIP  $id down/up (DESTRUCTIVE migration)"
    return
  fi

  for ((i = 0; i < N; i++)); do
    if ! ctl down >>"$log" 2>&1; then step_fail "$id" down "$log"; return; fi
  done
  v="$(version_of)"
  if [ "$v" != "0" ]; then echo "version after down is '$v', expected 0" >>"$log"; step_fail "$id" version "$log"; return; fi
  if ! ctl up >>"$log" 2>&1; then step_fail "$id" up "$log"; return; fi
  v="$(version_of)"
  if [ "$v" != "$N" ]; then echo "version after the second up is '$v', expected $N" >>"$log"; step_fail "$id" version "$log"; return; fi
  echo "PASS  $id up/down/up ($N -> 0 -> $N, default collation $coll)"
}

# ---------------------------------------------------------------- main

resolve_targets

VALID=()
FAILED=0
CHECKED=0
for id in "${TARGET_LIST[@]:-}"; do
  [ -z "$id" ] && continue
  CHECKED=$((CHECKED + 1))
  if validate_target "$id"; then
    VALID+=("$id")
    [ "${LIST_ONLY:-0}" = 1 ] && echo "target=$id n=$N destructive=$DESTRUCTIVE validate=ok"
  else
    if [ "${LIST_ONLY:-0}" = 1 ]; then echo "target=$id n=0 destructive=0 validate=FAIL $VALIDATE_MSG"
    else echo "FAIL  $id validate: $VALIDATE_MSG"; fi
    FAILED=$((FAILED + 1))
  fi
done

if [ "${LIST_ONLY:-0}" = 1 ]; then
  [ "$FAILED" -eq 0 ]; exit $?
fi

if [ "$CHECKED" -eq 0 ]; then
  echo "checked 0, failed 0 (no target)"
  exit 0
fi

if [ "${#VALID[@]}" -gt 0 ]; then
  preflight || exit 1
  for id in "${VALID[@]}"; do run_target "$id"; done
fi

echo "checked $CHECKED, failed $FAILED"
[ "$FAILED" -eq 0 ]
