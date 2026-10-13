#!/usr/bin/env bash
#
# Enforces three structure conventions that no stock linter can express:
#   R2a  transport DTO packages (pkg/listenhandler/models/{request,response})
#        may only be imported from pkg/listenhandler and the RPC client
#        (root CLAUDE.md, "Layering - transport DTO ownership")
#   R2b  the squirrel query builder may only be imported from the dbhandler
#        directories (docs/conventions/database.md 7.1 and 7.7)
#   R3   a persisted model that carries both an ID and a CustomerID embeds
#        commonidentity.Identity instead of redefining them
#        (docs/conventions/models.md 6.1); the existing hand-written structs
#        are pinned in scripts/structure-conventions-allow.txt
#
# Whole-tree check (no git diff, no merge base): the current violation count of
# R2 is 0, and R3 is decided per struct against the allow list.
#
# Fail-closed: an enumeration failure, an empty enumeration, an unreadable file
# or allow list, an unparseable construct, a dead scanner (no SCANNED line) and
# a stale allow-list entry all exit non-zero.
#
# Usage: scripts/check-structure-conventions.sh [--self-test]
# Note: files not yet `git add`ed are invisible to `git ls-files` locally; CI
# checks out committed files so it is unaffected.
# Note: --self-test runs every case with both `awk` and `mawk` (they are the
# same binary on Ubuntu images; gawk compatibility is only exercised where awk
# is gawk). A missing variant is a failure, not a skip. Do not use gawk-only
# features in AWK_PROG: gensub, asort, \b \w \s, {n} intervals, switch,
# BEGINFILE/ENDFILE, PROCINFO, length(array). Every opened file is close()d.
# Note: R2 recognizes `import` only at the start of a line (gofmt layout);
# golangci-lint's gofmt formatter in .golangci.yml rejects other layouts.
# Note: [*\[\]]* from the design doc is written [][*]* here (POSIX bracket).
#
set -uo pipefail
export LC_ALL=C

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ALLOW_DEFAULT="${SCRIPT_DIR}/structure-conventions-allow.txt"
# R2b allowed directories as written in the database.md exception table.
# The regexes in AWK_PROG must stay in step with these strings.
R2B_DOC_PATHS=('bin-*/pkg/dbhandler/' 'bin-common-handler/pkg/databasehandler/' 'bin-timeline-manager/pkg/analysisdbhandler/')

read -r -d '' AWK_PROG <<'AWKEOF'
function gate(m) { print "GATE " m; ngate++ }
function viol(r, loc, m) { print "VIOLATION " r " " loc ": " m; nviol++ }

# lex(): strip comments from L[1..n] into C (literals kept) and B (literals
# blanked, for brace counting). S[i]=1 when line i starts in code state.
# Returns 0 and reports a gate failure on any unterminated token.
function lex(f, n,   i, k, line, len, ch, nx, st, out, outb, j, c) {
  st = 0
  for (i = 1; i <= n; i++) {
    line = L[i]; len = length(line); S[i] = (st == 0); SR[i] = (st != 2); out = ""; outb = ""; k = 1
    while (k <= len) {
      ch = substr(line, k, 1); nx = substr(line, k + 1, 1)
      if (st == 1) {
        if (ch == "*" && nx == "/") { st = 0; k += 2; out = out " "; outb = outb " " } else k++
        continue
      }
      if (st == 2) {
        out = out ch
        if (ch == "`") { st = 0; outb = outb ch }
        k++; continue
      }
      if (ch == "/" && nx == "/") break
      if (ch == "/" && nx == "*") { st = 1; k += 2; out = out " "; outb = outb " "; continue }
      if (ch == "`") { st = 2; out = out ch; outb = outb ch; k++; continue }
      if (ch == "\"") {
        j = k + 1
        while (j <= len) { c = substr(line, j, 1); if (c == "\\") j += 2; else if (c == "\"") break; else j++ }
        if (j > len) { gate(f ":" i ": unterminated string literal"); return 0 }
        out = out substr(line, k, j - k + 1); outb = outb "\"\""; k = j + 1; continue
      }
      if (ch == "'") {
        j = k + 1
        if (substr(line, j, 1) == "\\") j += 2; else j++
        while (j <= len && substr(line, j, 1) != "'") j++
        if (j > len) { gate(f ":" i ": unterminated rune literal"); return 0 }
        out = out substr(line, k, j - k + 1); outb = outb "''"; k = j + 1; continue
      }
      out = out ch; outb = outb ch; k++
    }
    C[i] = out; B[i] = outb
  }
  if (st == 1) { gate(f ": unterminated /* comment"); return 0 }
  if (st == 2) { gate(f ": unterminated raw string"); return 0 }
  return 1
}

function r2check(f, i, p) {
  if (p ~ /^monorepo\/[^\/]+\/pkg\/listenhandler\/models\/(request|response)(\/.*)?$/) {
    if (f ~ /^[^\/]+\/pkg\/listenhandler\// || f ~ /^bin-common-handler\/pkg\/requesthandler\//) nallowed++
    else viol("R2a", f ":" i, "imports " p "; transport DTO packages may only be imported from pkg/listenhandler (or the requesthandler RPC client). See CLAUDE.md, Layering: transport DTO ownership")
  } else if (p == "github.com/Masterminds/squirrel" || p ~ /^github\.com\/Masterminds\/squirrel\//) {
    if (f ~ /^bin-[^\/]+\/pkg\/dbhandler\// || f ~ /^bin-common-handler\/pkg\/databasehandler\// || f ~ /^bin-timeline-manager\/pkg\/analysisdbhandler\//) nallowed++
    else viol("R2b", f ":" i, "imports squirrel outside the dbhandler directories. See docs/conventions/database.md 7.1 and 7.7")
  }
}

function r2scan(f, n,   i, line, inblk, rest, m) {
  inblk = 0
  for (i = 1; i <= n; i++) {
    if (!SR[i]) continue
    line = C[i]
    if (!inblk) {
      if (line ~ /^[[:space:]]*import[[:space:]]*\(/) { inblk = 1; sub(/^[[:space:]]*import[[:space:]]*\(/, "", line) }
      else if (line ~ /^[[:space:]]*import[[:space:]]/) { sub(/^[[:space:]]*import/, "", line) }
      else continue
    }
    if (index(line, "`")) { gate(f ":" i ": backtick import path is not supported"); return }
    rest = line
    while (match(rest, /"[^"]*"/)) { r2check(f, i, substr(rest, RSTART + 1, RLENGTH - 2)); rest = substr(rest, RSTART + RLENGTH) }
    if (inblk && index(line, ")")) inblk = 0
  }
  if (inblk) gate(f ": unterminated import block")
}

function isid(s)  { return (s ~ /(^|[^[:alnum:]_])ID[[:space:]]+[][*]*[[:alnum:]_]+\.UUID/ && s ~ /db:"id[,"]/) }
function iscid(s) { return (s ~ /(^|[^[:alnum:]_])CustomerID[[:space:]]+[][*]*[[:alnum:]_]+\.UUID/ && s ~ /db:"customer_id[,"]/) }

function r3scan(f, n,   i, d0, nb, nc, tmp, depth, inblk, name, hid, hcid, raw, att, un, dir, key, op) {
  depth = 0; inblk = 0; raw = 0; att = 0; un = 0
  dir = f; sub(/\/[^\/]*$/, "", dir)
  for (i = 1; i <= n; i++) {
    if (L[i] !~ /^[[:space:]]*\/\// && (isid(L[i]) || iscid(L[i]))) raw++
    d0 = depth; tmp = B[i]; op = gsub(/\{/, "{", tmp); tmp = B[i]; nc = gsub(/\}/, "}", tmp)
    if (isid(C[i]) || iscid(C[i])) {
      if (inblk && d0 == 1 && op == 0 && nc == 0) { att++; if (isid(C[i])) hid = 1; if (iscid(C[i])) hcid = 1 }
      else { un++; gate(f ":" i ": ID/CustomerID definition outside a recognized top-level struct field (one-line, nested, anonymous, grouped or generic struct)") }
    }
    if (inblk && d0 >= 1 && S[i] && B[i] ~ /^[[:space:]]*type[[:space:]]/) { gate(f ":" i ": type declaration inside an open struct block (unbalanced braces?)"); return }
    if (!inblk && d0 == 0 && S[i] && match(B[i], /^[[:space:]]*type[[:space:]]+[[:alnum:]_]+[[:space:]]+struct[[:space:]]*\{/)) {
      name = B[i]; sub(/^[[:space:]]*type[[:space:]]+/, "", name); sub(/[^[:alnum:]_].*$/, "", name)
      inblk = 1; hid = 0; hcid = 0; bline = i
    }
    depth += op - nc
    if (depth < 0) { gate(f ":" i ": unbalanced closing brace"); return }
    if (inblk && depth == 0) {
      inblk = 0; nstruct++
      if (hid && hcid) { key = dir ":" name; found[key] = f ":" bline }
    }
  }
  if (depth != 0) { gate(f ": unbalanced braces at end of file"); return }
  if (raw != att + un) gate(f ": ID/CustomerID definition lines found by a plain grep (" raw ") differ from the lines the scanner attributed (" att + un ")")
}

function loadallow(   line, r, k, nk) {
  nk = 0
  while ((r = (getline line < ALLOW)) > 0) {
    gsub(/[\r\t]/, " ", line)
    if (line ~ /^[[:space:]]*(#|$)/) continue
    k = line; sub(/^[[:space:]]+/, "", k); sub(/[[:space:]].*$/, "", k)
    if (k in allow) gate("allow list: duplicate key " k)
    allow[k] = 1; nk++
  }
  if (r < 0) { gate("allow list: cannot read " ALLOW); close(ALLOW); return }
  close(ALLOW)
  if (nk == 0) gate("allow list: no entries in " ALLOW)
}

BEGIN {
  loadallow()
  while ((r = (getline f < LIST)) > 0) {
    nfiles++
    n = 0; pre = 0; split("", L); split("", C); split("", B); split("", S); split("", SR)
    while ((r2 = (getline line < f)) > 0) {
      L[++n] = line
      if (index(line, "listenhandler/models/") || index(line, "Masterminds/squirrel")) pre = 1
    }
    close(f)
    if (r2 < 0) { gate(f ": cannot read file"); continue }
    ism = (f ~ /^bin-[^\/]+\/models\// && f !~ /_test\.go$/)
    if (!pre && !ism) continue
    if (!lex(f, n)) continue
    if (pre) { nr2++; r2scan(f, n) }
    if (ism) { nr3++; r3scan(f, n) }
  }
  if (r < 0) gate("cannot read file list " LIST)
  close(LIST)
  for (k in found) if (!(k in allow)) viol("R3", found[k], "struct " k " defines ID and CustomerID by hand; embed commonidentity.Identity instead. See docs/conventions/models.md 6.1")
  for (k in allow) if (!(k in found)) gate("allow list: stale entry " k " is no longer a hand-written struct; delete it from scripts/structure-conventions-allow.txt")
  print "SCANNED files=" nfiles " r2files=" nr2 " r3files=" nr3 " structs=" nstruct " allowed_imports=" nallowed " violations=" nviol+0 " gate_failures=" ngate+0
}
AWKEOF

# run_check <repo> <awk> <allow-list> <database.md>; exit 0 pass, 1 violation or gate failure
run_check() {
  local repo=$1 awkbin=$2 allow=$3 dbdoc=$4 listf out rc p
  listf="$(mktemp)"
  if ! (cd "$repo" && git -c core.quotePath=false ls-files '*.go') > "$listf" 2>/dev/null; then
    echo "GATE cannot enumerate Go files (git ls-files failed in $repo)"; rm -f "$listf"; return 1
  fi
  grep -vE '(^|/)vendor/' "$listf" > "$listf.f" || true; mv "$listf.f" "$listf"
  if [ ! -s "$listf" ]; then echo "GATE no Go files enumerated; refusing to pass an empty scan"; rm -f "$listf"; return 1; fi
  rc=0
  for p in "${R2B_DOC_PATHS[@]}"; do
    grep -qF -- "$p" "$dbdoc" 2>/dev/null || { echo "GATE database.md R2b exception table lacks '$p' (script and doc drifted)"; rc=1; }
  done
  out="$(cd "$repo" && "$awkbin" -v LIST="$listf" -v ALLOW="$allow" "$AWK_PROG" </dev/null 2>&1)"; local arc=$?
  rm -f "$listf"
  [ -n "$out" ] && echo "$out"
  [ "$arc" -eq 0 ] || { echo "GATE scanner exited $arc"; rc=1; }
  grep -q '^SCANNED ' <<<"$out" || { echo "GATE scanner produced no SCANNED line (dead scanner?)"; rc=1; }
  # here-string, not `echo | grep -q`: with pipefail a large output makes grep -q
  # exit early, echo dies of SIGPIPE and the pipeline reads as "no match" (fail-open)
  grep -qE '^(VIOLATION|GATE) ' <<<"$out" && rc=1
  return $rc
}

# ---------------------------------------------------------------- self-test
# Cases are functions S<nn>. Each starts from an empty git repo (reset), adds
# the baseline (base) and fixtures (f), then asserts the exit code and output
# (want). A mutated awk (broken scanner) must make the gate fail, not pass.
f() { mkdir -p "$(dirname "$1")"; printf '%s\n' "$2" > "$1"; }
manual() { printf 'package m\n\nimport "github.com/gofrs/uuid"\n\ntype %s struct {\n\t%s uuid.UUID `db:"id,uuid"`\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n}\n' "$1" "${2:-ID}"; }
DTO='"monorepo/bin-b-manager/pkg/listenhandler/models/request"'
SQ='"github.com/Masterminds/squirrel"'
imp() { printf 'package p\n\nimport %s\n' "$1"; }
reset() { cd "$T" && rm -rf repo && mkdir repo && cd repo && git init -q; }
base() {
  f bin-allowed/models/old/old.go "$(manual Old)"
  printf '# allow list\nbin-allowed/models/old:Old legacy\n' > "$ALLOW"
  printf 'bin-*/pkg/dbhandler/\nbin-common-handler/pkg/databasehandler/\nbin-timeline-manager/pkg/analysisdbhandler/\n' > "$DBDOC"
}
want() { # <rc> [output-pattern...]
  local rc=$1 out got p; shift
  [ -n "${NOADD:-}" ] || git add -A >/dev/null 2>&1
  out="$(run_check "$PWD" "$AWKV" "$ALLOW" "$DBDOC" 2>&1)"; got=$?
  [ "$got" -eq "$rc" ] || { echo "FAIL $CASE ($AWKV): rc=$got want $rc"; echo "$out" | head -5; NFAIL=$((NFAIL+1)); return; }
  for p in "$@"; do grep -q -- "$p" <<<"$out" || { echo "FAIL $CASE ($AWKV): output lacks '$p'"; NFAIL=$((NFAIL+1)); return; }; done
}
S01() { f bin-a-manager/pkg/foohandler/x.go "$(imp "$DTO")"; want 1 R2a; }
S02() { f bin-a-manager/pkg/listenhandler/x.go "$(imp '"monorepo/bin-a-manager/pkg/listenhandler/models/request"')"; want 0; }
S03() { f bin-a-manager/pkg/listenhandler/x.go "$(imp "$DTO")"; want 0; }
S04() { f bin-common-handler/pkg/requesthandler/x.go "$(imp "$DTO")"; want 0; }
S05() { f bin-a-manager/pkg/listenhandlerx/x.go "$(imp "$DTO")"; want 1 R2a; }
S06() { f bin-common-handler/pkg/requesthandler2/x.go "$(imp "$DTO")"; want 1 R2a; }
S07() { f bin-a-manager/pkg/foo/x.go "$(imp '"monorepo/bin-b-manager/pkg/listenhandler/models/request/sub"')"; want 1 R2a; }
S08() { f bin-new-manager/pkg/foo/x.go "$(imp "$DTO")"; want 1 R2a; }
S09() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nimport (\n\t"fmt"\n\tamreq %s\n)' "$DTO")"; want 1 R2a; }
S10() { f bin-a-manager/pkg/foo/x.go "$(imp "_ $DTO")"; want 1 R2a; }
S11() { f bin-a-manager/pkg/foo/x.go "$(imp ". $DTO")"; want 1 R2a; }
S12() { f bin-a-manager/pkg/foo/x.go "$(imp "(${DTO/request/response})")"; want 1 R2a; }
S13() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n// import %s\n/* import %s */\n' "$DTO" "$DTO")"; want 0; }
S14() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nvar s = %s\n' "$DTO")"; want 0; }
S15() { f bin-a-manager/pkg/foo/x.go "$(imp '`monorepo/bin-b-manager/pkg/listenhandler/models/request`')"; want 1 GATE; }
S16() { f bin-a-manager/pkg/listenhandler/x.go "$(printf 'package p\n\nimport %s\n\ntype T struct {\n\tA string `json:"a"`\n}\n' "$DTO")"; want 0; }
S17() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nimport (\n\t/* c */ %s\n)' "$DTO")"; want 1 R2a; }
S18() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nimport (\n\t%s\n' "$DTO")"; want 1 GATE; }
S19() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nvar x = `\nimport %s\n`\n' "$DTO")"; want 0; }
S20() { f bin-a-manager/pkg/foo/x_test.go "$(imp "$DTO")"; want 1 R2a; }
S22() { f bin-a-manager/pkg/foohandler/x.go "$(imp "$SQ")"; want 1 R2b; }
S23() { f bin-a-manager/pkg/dbhandler/x.go "$(imp "$SQ")"; want 0; }
S24() { f bin-common-handler/pkg/databasehandler/x.go "$(imp "$SQ")"; want 0; }
S25() { f bin-timeline-manager/pkg/analysisdbhandler/x.go "$(imp "$SQ")"; want 0; }
S26() { f bin-a-manager/pkg/dbhandlerx/x.go "$(imp "$SQ")"; want 1 R2b; }
S27() { f bin-common-handler/pkg/databasehandler/x_test.go "$(imp "$SQ")"; want 0; }
S28() { f bin-a-manager/pkg/foohandler/x_test.go "$(imp "$SQ")"; want 1 R2b; }
S29() { printf 'bin-*/pkg/dbhandler/\nbin-common-handler/pkg/databasehandler/\n' > "$DBDOC"; want 1 GATE; }
S31() { f bin-a-manager/pkg/foohandler/x.go "$(imp "sq $SQ")"; want 1 R2b; }
S32() { f bin-n/models/new/new.go "$(manual New)"; want 1 R3; }
S33() { f bin-allowed/models/old/two.go "$(manual Two)"; want 1 R3; }
S34() { f bin-n/models/n/n.go 'package n
import commonidentity "monorepo/bin-common-handler/models/identity"
type N struct {
	commonidentity.Identity
}'; want 0; }
S35() { f bin-n/models/n/n.go "$(printf 'package n\ntype N struct {\n\tID uuid.UUID `db:"id,uuid"`\n}\ntype M struct {\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n}')"; want 0; }
S37() { f bin-n/models/n/filters.go "$(printf 'package n\ntype F struct {\n\tID uuid.UUID `filter:"id"`\n\tCustomerID uuid.UUID `filter:"customer_id"`\n}')"; want 0; }
S38() { f bin-n/models/n/webhook.go "$(printf 'package n\ntype W struct {\n\tID uuid.UUID `json:"id"`\n\tCustomerID uuid.UUID `json:"customer_id"`\n}')"; want 0; }
S39() { f bin-allowed/models/old/old.go "$(printf 'package m\ntype Old struct {\n\tID uuid.UUID `json:"id" db:"id,uuid"`\n\tCustomerID uuid.UUID `db:"customer_id,uuid" json:"c"`\n}')"; want 0; }
S40() { f bin-allowed/models/old/old.go "$(printf 'package m\n\n\ntype Old struct {\n\tName string\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n\tID uuid.UUID `db:"id,uuid"`\n}')"; want 0; }
S41() { rm bin-allowed/models/old/old.go; f bin-allowed/models/old/renamed.go "$(manual Old)"; want 0; }
S42() { f bin-allowed/models/old/old.go "$(printf 'package m\ntype Old struct {\n\tcommonidentity.Identity\n}\n%s' "$(manual New | sed 1,3d)")"; want 1 R3 stale; }
S43() { printf 'bin-allowed/models/old:Old x\nbin-gone/models/g:G x\n' > "$ALLOW"; want 1 GATE stale; }
S44() { rm "$ALLOW"; want 1 GATE; }
S45() { printf 'bin-allowed/models/old:Old x\nbin-allowed/models/old:Old y\n' > "$ALLOW"; want 1 GATE; }
S46() { printf '# only comments\n\n' > "$ALLOW"; want 1 GATE; }
S47() { printf 'bin-allowed/models/old:Old x\nbin-allowed/models/old:OldX y\n' > "$ALLOW"; f bin-allowed/models/old/x.go "$(manual OldXX)"; want 1 R3; }
S48() { f bin-n/models/n/n.go 'package n
type X struct{ ID uuid.UUID `db:"id"` }'; want 1 GATE; }
S49() { f bin-n/models/n/n.go "$(printf 'package n\ntype (\n\tX struct {\n\t\tID uuid.UUID `db:"id"`\n\t}\n)')"; want 1 GATE; }
S50() { f bin-allowed/models/old/old.go "$(printf 'package m\ntype Old struct {\n\tID uuid.UUID `db:"id,uuid"`\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n\tIn struct {\n\t\tID uuid.UUID `db:"id"`\n\t}\n}')"; want 1 GATE; }
S51() { f bin-n/models/n/n.go "$(printf 'package n\nvar v = struct {\n\tID uuid.UUID `db:"id"`\n}{}')"; want 1 GATE; }
S52() { f bin-n/models/n/n.go "$(printf 'package n\n// type Fake struct {\n%s' "$(manual Real | sed 1,2d)")"; want 1 R3 'n:Real'; }
S53() { f bin-n/models/n/n.go "$(printf 'package n\n/*\ntype Fake struct {\n*/\n%s' "$(manual Real | sed 1,2d)")"; want 1 R3 'n:Real'; }
S54() { f bin-n/models/n/n.go "$(printf 'package n\nvar s = "a/*b"\n%s' "$(manual Real | sed 1,2d)")"; want 1 R3 'n:Real'; }
S55() { f bin-n/models/n/n.go "$(printf 'package n\ntype Real struct {\n\tID uuid.UUID `example:"http://a" db:"id,uuid"`\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n}')"; want 1 R3 'n:Real'; }
S56() { f bin-n/models/n/n.go 'package n
/* never closed'; want 1 GATE; }
S57() { f bin-n/models/n/n.go 'package n
var s = "abc'; want 1 GATE; }
S58() { f bin-n/models/n/n.go 'package n
var s = `abc'; want 1 GATE; }
S59() { f bin-n/models/n/n.go "package n
var r = 'a"; want 1 GATE; }
S60() { f bin-n/models/n/n.go "$(printf 'package n\n/*\n\tID uuid.UUID `db:"id"`\n*/')"; want 1 GATE; }
S61() { f bin-n/models/n/n.go "$(printf 'package n\ntype R struct {\n\tID u.UUID `db:"id,omitempty"`\n\tCustomerID u.UUID `db:"customer_id,uuid"`\n}')"; want 1 R3 'n:R'; }
S62() { f bin-n/models/n/n.go "$(printf 'package n\ntype R struct {\n\tID *uuid.UUID `db:"id"`\n\tCustomerID []uuid.UUID `db:"customer_id"`\n}')"; want 1 R3 'n:R'; }
S64() { f bin-n/models/n/n.go "$(printf 'package n\ntype R struct {\n\tX, ID uuid.UUID `db:"id"`\n\tY, CustomerID uuid.UUID `db:"customer_id"`\n}')"; want 1 R3 'n:R'; }
S65() { f bin-n/models/n/n.go "$(printf 'package n\ntype R struct {\n\tID, X uuid.UUID `db:"id"`\n\tCustomerID, Y uuid.UUID `db:"customer_id"`\n}')"; want 0; }
S66() { f bin-allowed/models/old/old.go "$(printf 'package m\ntype Old struct {\n\tA string `json:"a{b"`\n\tID uuid.UUID `db:"id,uuid"`\n\tCustomerID uuid.UUID `db:"customer_id,uuid"`\n}\nconst o = '"'{'"'\ntype Z struct {\n\tB string\n}')"; want 0; }
S67() { f bin-n/models/n/n.go "$(printf 'package n\ntype G[T any] struct {\n\tID uuid.UUID `db:"id"`\n\tCustomerID uuid.UUID `db:"customer_id"`\n}')"; want 1 GATE; }
S68() { f bin-n/models/n/n.go "$(printf 'package n\ntype A struct {\n\tX string\ntype B struct {\n\tID uuid.UUID `db:"id"`\n}')"; want 1 GATE; }
S69() { f bin-n/models/n/n.go "$(printf 'package n\ntype E struct{}\n%s' "$(manual Q | sed 1,2d)")"; want 1 R3 'n:Q'; }
S70() { f bin-n/models/n/n_test.go "$(manual T)"; want 0; }
S71() { f bin-a-manager/pkg/dbhandler/addr.go "$(manual A)"; want 0; }
S72() { f bin-openapi-manager/gens/models/x.go "$(manual X)"; want 0; }
S73() { printf 'bin-allowed/models/old:Old\twhy\r\n' > "$ALLOW"; want 0; }
S74() { printf 'bin-allowed/models/old:Old\n' > "$ALLOW"; want 0; }
S75() { f bin-a-manager/pkg/foohandler/x.go "$(imp "$DTO")"; printf 'bin-allowed/models/old:Old x\nbin-gone/models/g:G x\n' > "$ALLOW"; want 1 VIOLATION GATE; }
S80() { reset; want 1 GATE; }
S81() { cd "$T" && rm -rf nogit && mkdir nogit && cd nogit && want 1 GATE; }
S82() { local A=$AWKV; AWKV="$T/bad-awk"; printf '#!/bin/sh\nexit 1\n' > "$AWKV"; chmod +x "$AWKV"; want 1 GATE; AWKV=$A; }
S83() { local A=$AWKV; AWKV="$T/dead-awk"; printf '#!/bin/sh\nexit 0\n' > "$AWKV"; chmod +x "$AWKV"; want 1 GATE; AWKV=$A; }
S84() { f bin-a-manager/pkg/x/x.go 'package x'; git add -A; rm bin-a-manager/pkg/x/x.go; NOADD=1 want 1 GATE; }
S85() { want 0; }
S92() { f bin-a-manager/vendor/x/x.go "$(imp "$DTO")"; want 0; }
S93() { f bin-a-manager/internal/pkg/listenhandler/x.go "$(imp "$DTO")"; want 1 R2a; }
S94() { rm -f "$ALLOW"; printf 'bin-allowed/models/old:Old x\n' > "$ALLOW"; rm -rf ./*; want 1 'no Go files'; }
S95() { f bin-n/models/n/n.go 'package n
}'; want 1 GATE; }
S96() { local i; for i in $(seq 400); do f bin-a-manager/pkg/foo/x$i.go "$(imp "$DTO")"; done; want 1 'violations=400'; }
S86() { f bin-a-manager/pkg/foo/x.go "$(printf 'package x\n/* c\n*/ import %s\n' "$DTO")"; want 1 R2a; }
S87() { f bin-a-manager/pkg/foo/x.go "$(printf 'package x\nvar s = "q\\"/*"\nimport %s\n' "$DTO")"; want 1 R2a; }
S88() { f bin-a-manager/pkg/foohandler/x.go "$(imp '"github.com/Masterminds/squirrel/v3"')"; want 1 R2b; }
S89() { f bin-n/models/n/n.go "$(printf 'package n\ntype R struct {\n\tParentID uuid.UUID `db:"id"`\n\tCustomerID uuid.UUID `db:"customer_id"`\n}')"; want 0; }
S90() { f bin-a-manager/pkg/foo/x.go "$(printf 'package p\n\nimport (\n\t"fmt"\n)\n\nvar l = []string{\n\t%s,\n}\n' "$DTO")"; want 0; }
S91() { f bin-n/models/n/n.go 'package n
type A struct {
	X string'; want 1 GATE; }

self_test() {
  local v; NFAIL=0; NCASE=0
  T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
  ALLOW="$T/allow.txt"; DBDOC="$T/db.md"
  for v in awk mawk; do
    command -v "$v" >/dev/null 2>&1 || { echo "self-test: '$v' not found; install it (mawk is required, a skipped variant is a fail-open gate)" >&2; return 1; }
    AWKV="$(command -v "$v")"; echo "self-test: variant $v -> $AWKV"
    for CASE in $(compgen -A function | grep '^S[0-9]' | sort); do
      reset; base
      # S80/S81/S84 build their own state after base
      NCASE=$((NCASE+1)); "$CASE"
    done
  done
  [ "$NFAIL" -eq 0 ] && { echo "self-test: $NCASE case runs passed"; return 0; }
  echo "self-test: $NFAIL of $NCASE case runs FAILED" >&2; return 1
}

if [ "${1:-}" = "--self-test" ]; then self_test; exit $?; fi
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "GATE not inside a git repository"; exit 1; }
AWKBIN="$(command -v mawk || command -v awk)" || { echo "GATE awk not found"; exit 1; }
run_check "$ROOT" "$AWKBIN" "$ALLOW_DEFAULT" "$ROOT/docs/conventions/database.md"
