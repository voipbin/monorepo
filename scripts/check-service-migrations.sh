#!/usr/bin/env bash
# Git relative check of the migration files of a service that owns its schema.
#
# usage: scripts/check-service-migrations.sh <service-dir> [<migrations-subdir>=migrations]
#        scripts/check-service-migrations.sh --self-test
#
# Compares HEAD with origin/main (three dot diff) inside <service-dir>/<migrations-subdir> and fails when
#   - a .sql file there was modified, deleted, renamed or changed in type or mode (merged files are immutable),
#   - an added file is not a regular file named NNNNNN_<description>.(up|down).sql, or sits in a subdirectory,
#   - the added numbers do not continue the numbers of origin/main without gaps (from 000001 when
#     origin/main has no migration there) or an up file has no down file or the reverse.
# Run it right after `git fetch origin main`. It checks commits, not the working tree.
# Limits: a changed VersionTable in embed.go is not detected; a number that another branch adds
# later is caught by the structure check of the service tests after the merge.
set -euo pipefail

die() { echo "check-service-migrations: $*" >&2; exit 1; }

# check_dir <repo> <service-dir> <migrations-subdir>: prints the violations, returns 0 when none
check_dir() {
  local repo="$1" svc="$2" sub="$3" dir problems=0 meta path rel status newmode num max=0 name
  dir="$svc/$sub"
  local -A ups=() downs=()

  git -C "$repo" rev-parse --verify -q 'origin/main^{commit}' >/dev/null || { echo "origin/main does not exist (run git fetch origin main)"; return 1; }
  git -C "$repo" merge-base origin/main HEAD >/dev/null 2>&1 || { echo "no merge base with origin/main"; return 1; }
  [ -n "$(git -C "$repo" ls-tree --name-only HEAD -- "$dir/")" ] || { echo "$dir does not exist in HEAD"; return 1; }

  # the highest up number already on origin/main
  local listing
  listing=$(git -C "$repo" ls-tree --name-only origin/main -- "$dir/") || { echo "could not list $dir on origin/main"; return 1; }
  while IFS= read -r name; do
    name="${name##*/}"
    if [[ "$name" =~ ^([0-9]{6})_[a-z0-9_]+\.up\.sql$ ]]; then
      num=$((10#${BASH_REMATCH[1]})); [ "$num" -gt "$max" ] && max=$num
    fi
  done <<< "$listing"

  # read the whole diff first so that a failing git command cannot pass the check
  local raw
  raw=$(mktemp)
  if ! git -C "$repo" -c core.quotePath=false diff --raw -z --no-renames --no-abbrev 'origin/main...HEAD' -- "$dir" > "$raw"; then
    rm -f "$raw"; echo "git diff failed"; return 1
  fi

  while IFS= read -r -d '' meta; do
    IFS= read -r -d '' path
    status="${meta##* }"; newmode="${meta#:* }"; newmode="${newmode%% *}"
    rel="${path#"$dir/"}"
    [[ "${rel,,}" == *.sql ]] || continue
    if [[ "$rel" == */* ]]; then echo "$path: a .sql file in a subdirectory"; problems=1; continue; fi
    if [ "$status" != "A" ]; then echo "$path: status $status, merged migration files are immutable"; problems=1; continue; fi
    if [ "$newmode" != "100644" ]; then echo "$path: mode $newmode, a regular file (100644) is required"; problems=1; continue; fi
    if [[ "$rel" =~ ^([0-9]{6})_[a-z0-9_]+\.(up|down)\.sql$ ]]; then
      num=$((10#${BASH_REMATCH[1]}))
      if [ "${BASH_REMATCH[2]}" = up ]; then
        [ -n "${ups[$num]:-}" ] && { echo "$path: number $num is also used by ${ups[$num]}"; problems=1; }
        ups[$num]="$rel"
      else
        [ -n "${downs[$num]:-}" ] && { echo "$path: number $num is also used by ${downs[$num]}"; problems=1; }
        downs[$num]="$rel"
      fi
    else
      echo "$path: the file name must be NNNNNN_<description>.(up|down).sql"; problems=1
    fi
  done < "$raw"
  rm -f "$raw"

  for num in "${!ups[@]}"; do [ -n "${downs[$num]:-}" ] || { echo "$dir: ${ups[$num]} has no down file"; problems=1; }; done
  for num in "${!downs[@]}"; do [ -n "${ups[$num]:-}" ] || { echo "$dir: ${downs[$num]} has no up file"; problems=1; }; done

  # the added numbers continue the numbers of origin/main without gaps
  local expect=$((max+1))
  for num in $(printf '%s\n' "${!ups[@]}" | sort -n); do
    [ -z "$num" ] && continue
    if [ "$num" -ne "$expect" ]; then
      echo "$dir: added number $num, expected $expect (origin/main has up to $max)"; problems=1
      expect=$((num+1))
    else
      expect=$((expect+1))
    fi
  done

  [ "$problems" = 0 ]
}

self_test() {
  local tmp fails=0
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  local repo="$tmp/repo"
  git init -q "$repo"
  git -C "$repo" config user.email t@example.invalid; git -C "$repo" config user.name t
  mkdir -p "$repo/svc/migrations" "$repo/svc2/migrations"
  echo 'SELECT 1;' > "$repo/svc/migrations/000001_baseline.up.sql"; echo 'SELECT 1;' > "$repo/svc/migrations/000001_baseline.down.sql"
  echo 'package migrations' > "$repo/svc/migrations/embed.go"; echo 'package migrations' > "$repo/svc2/migrations/embed.go"
  git -C "$repo" add -A; git -C "$repo" commit -q -m base
  git -C "$repo" update-ref refs/remotes/origin/main HEAD
  git -C "$repo" branch -q -M main

  run() { # run <name> <expect 0|1> <service> <commands...>
    local name="$1" want="$2" svc="$3" cmd="$4" out rc=0
    git -C "$repo" checkout -q -f -B work main
    git -C "$repo" update-ref refs/remotes/origin/main main
    ( cd "$repo" && eval "$cmd" ) >/dev/null 2>&1
    git -C "$repo" add -A; git -C "$repo" commit -q --allow-empty -m change
    out=$(check_dir "$repo" "$svc" migrations 2>&1) || rc=$?
    if { [ "$want" = 0 ] && [ "$rc" = 0 ]; } || { [ "$want" = 1 ] && [ "$rc" != 0 ]; }; then echo "PASS  $name"; else echo "FAIL  $name (rc=$rc) $out"; fails=$((fails+1)); fi
  }

  run "no change" 0 svc ':'
  run "add 000002 up and down" 0 svc 'echo x > svc/migrations/000002_a_add_b.up.sql; echo x > svc/migrations/000002_a_add_b.down.sql'
  run "add 000002 and 000003" 0 svc 'for n in 2 3; do echo x > svc/migrations/00000${n}_a.up.sql; echo x > svc/migrations/00000${n}_a.down.sql; done'
  run "first PR of a service with 000001 and 000002" 0 svc2 'for n in 1 2; do echo x > svc2/migrations/00000${n}_a.up.sql; echo x > svc2/migrations/00000${n}_a.down.sql; done'
  run "a non sql file is ignored" 0 svc 'echo x > svc/migrations/README.md'
  run "modify the baseline" 1 svc 'echo y >> svc/migrations/000001_baseline.up.sql'
  run "delete a down file" 1 svc 'git rm -q svc/migrations/000001_baseline.down.sql'
  run "rename the baseline" 1 svc 'git mv svc/migrations/000001_baseline.up.sql svc/migrations/000001_base.up.sql'
  run "mode change of the baseline" 1 svc 'chmod +x svc/migrations/000001_baseline.up.sql'
  run "reuse number 000001" 1 svc 'echo x > svc/migrations/000001_other.up.sql; echo x > svc/migrations/000001_other.down.sql'
  run "skip a number" 1 svc 'echo x > svc/migrations/000003_a.up.sql; echo x > svc/migrations/000003_a.down.sql'
  run "up without down" 1 svc 'echo x > svc/migrations/000002_a.up.sql'
  run "down without up" 1 svc 'echo x > svc/migrations/000002_a.down.sql'
  run "two up files with the same number" 1 svc 'for s in a b; do echo x > svc/migrations/000002_$s.up.sql; done; echo x > svc/migrations/000002_a.down.sql'
  run "two down files with the same number" 1 svc 'echo x > svc/migrations/000002_a.up.sql; for s in a b; do echo x > svc/migrations/000002_$s.down.sql; done'
  run "upper case extension pair" 1 svc 'echo x > svc/migrations/000002_a.UP.SQL; echo x > svc/migrations/000002_a.DOWN.SQL'
  run "upper case extension" 1 svc 'echo x > svc/migrations/000002_a.UP.SQL; echo x > svc/migrations/000002_a.down.sql'
  run "upper case description" 1 svc 'echo x > svc/migrations/000002_A.up.sql; echo x > svc/migrations/000002_A.down.sql'
  run "file without a number" 1 svc 'echo x > svc/migrations/add_column.up.sql'
  run "sql file in a subdirectory" 1 svc 'mkdir svc/migrations/sub; echo x > svc/migrations/sub/000002_a.up.sql'
  run "symbolic link" 1 svc 'ln -s 000001_baseline.up.sql svc/migrations/000002_a.up.sql; echo x > svc/migrations/000002_a.down.sql'
  run "first PR of a service starting at 000002" 1 svc2 'echo x > svc2/migrations/000002_a.up.sql; echo x > svc2/migrations/000002_a.down.sql'
  run "service directory missing" 1 missing ':'

  # origin/main moved on after the branch point: the same number is now taken
  git -C "$repo" checkout -q -f -B work main
  echo x > "$repo/svc/migrations/000002_branch.up.sql"; echo x > "$repo/svc/migrations/000002_branch.down.sql"
  git -C "$repo" add -A; git -C "$repo" commit -q -m branch
  git -C "$repo" checkout -q -f main
  echo x > "$repo/svc/migrations/000002_main.up.sql"; echo x > "$repo/svc/migrations/000002_main.down.sql"
  git -C "$repo" add -A; git -C "$repo" commit -q -m main2
  git -C "$repo" update-ref refs/remotes/origin/main HEAD
  git -C "$repo" checkout -q -f work
  if check_dir "$repo" svc migrations >/dev/null 2>&1; then echo "FAIL  branch behind origin/main with a taken number"; fails=$((fails+1)); else echo "PASS  branch behind origin/main with a taken number"; fi

  # the merged branch only adds the next number
  git -C "$repo" merge -q -m merge main >/dev/null 2>&1 || true
  git -C "$repo" checkout -q -f -B work2 main
  echo x > "$repo/svc/migrations/000003_branch.up.sql"; echo x > "$repo/svc/migrations/000003_branch.down.sql"
  git -C "$repo" add -A; git -C "$repo" commit -q -m b3
  if check_dir "$repo" svc migrations >/dev/null 2>&1; then echo "PASS  branch on top of origin/main adds the next number"; else echo "FAIL  branch on top of origin/main adds the next number"; fails=$((fails+1)); fi

  # no origin/main
  git -C "$repo" update-ref -d refs/remotes/origin/main
  if check_dir "$repo" svc migrations >/dev/null 2>&1; then echo "FAIL  missing origin/main must fail"; fails=$((fails+1)); else echo "PASS  missing origin/main fails"; fi

  echo "== failures: $fails"
  [ "$fails" = 0 ]
}

if [ "${1:-}" = "--self-test" ]; then self_test; exit; fi

SERVICE="${1:?usage: $0 <service-dir> [<migrations-subdir>] | --self-test}"
SUB="${2:-migrations}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if check_dir "$ROOT" "${SERVICE%/}" "${SUB%/}"; then echo "check-service-migrations: OK $SERVICE/$SUB"; else die "FAILED $SERVICE/$SUB"; fi
