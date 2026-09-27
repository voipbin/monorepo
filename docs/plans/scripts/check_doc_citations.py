#!/usr/bin/env python3
"""Resolve every `<file>:<line>` citation in a plan document against the worktree.

Citation drift recurred in SIX consecutive review rounds of
docs/plans/2026-09-27-mcp-lifecycle-and-docs-analysis.md. Every instance had the
same shape: the commit that fixed the prose also expanded a comment, which
shifted the lines the prose cited. The anchors stayed inside the file, so a
range-only check reported OK while an implementer opening the cited line found
the wrong construct.

    python3 docs/plans/scripts/check_doc_citations.py docs/plans/<doc>.md

Exit status is 1 on any problem, so this can gate a commit.

WHAT ACTUALLY CLOSES THE DRIFT CLASS: a range check does not. A probe document
citing `start.go:1` for an arbitrary claim passes a range-only gate. So this
script supports SYMBOL-ANCHORED citations:

    `start.go:375 refreshMcpToolMap`

When a citation is followed by a bare identifier, the cited line MUST contain it,
or the run fails. Anchors written that way cannot silently drift. Anchors written
without a symbol are still range-checked, and the summary prints how many are
unverifiable, so that number is visible rather than invisible. Drive it down over
time and treat a rise in it as a regression; `--strict-anchors` fails on any.

Other checks:
  * `path:N`, `path:N-M`, `path:N,M` and bare `:N` continuations all resolve
  * commit-like tokens are validated, INCLUDING ones git cannot find at all
    (the round-17 blocker was a cited commit that had never existed)
  * landings on a blank line or a lone closing brace are reported as warnings
  * unresolvable bare basenames are reported, not silently called "external"

Ambiguous basenames (handler.go, start.go, main.go exist in several services)
are resolved by longest matching path suffix, then by BASENAME_PREFERENCE. A
citation that still matches more than one file is reported as AMBIGUOUS rather
than guessed: add enough path prefix to the citation to disambiguate it.
"""

import argparse
import os
import re
import subprocess
import sys

# `path:N`, optionally `-M` or `,M`. An optional symbol anchor may follow INSIDE
# the same backticks, after a space:  `start.go:375 refreshMcpToolMap`
# The symbol must be inside the code span, otherwise ordinary prose following a
# citation ("`handler.go:206` wraps ...") would be mistaken for an anchor.
CITATION = re.compile(
    r"(?P<tick>`?)(?P<path>[A-Za-z0-9_./-]+\.(?:go|py|yaml|yml|js|ts|rst|md|sql)):"
    r"(?P<start>\d+)(?:(?P<sep>[-,])(?P<end>\d+))?"
    r"(?:(?<=\S)(?P<anchorsp> )(?P<symbol>[A-Za-z_][A-Za-z0-9_.]*))?"
    r"(?P<endtick>`?)"
)
# A bare `:N` or `:N-M` continuation, e.g. "same file, `:53-59`". Bound to the
# most recent path cited on the same document line.
CONTINUATION = re.compile(r"`:(?P<start>\d+)(?:[-,](?P<end>\d+))?`")
HEXTOKEN = re.compile(r"\b([0-9a-f]{7,40})\b")

# Cited files that legitimately live outside every root we were given.
#
# This list is a LAST RESORT, consulted only after resolve() has failed against
# the main root AND every --extra-root. It used to be consulted FIRST, which
# meant every square-admin and api-validator citation was skipped unopened and
# a citation to line 999999 of a real file passed the gate. Do not restore that
# order: an entry here must mean "we have no copy of this file", never "do not
# check this file".
EXTERNAL_HINTS = (
    "skill.md",
    "llms.txt",
)

# Alembic revision identifiers are 12-hex strings that look exactly like short
# git hashes but are not git objects. Without this allowlist the commit check
# would reject every migration the document cites.
NON_COMMIT_HEX = {
    "9b0ad37e0360",
    "62c10f986f07",
    "071504ef41d0",
    "1ebd3fdcea8d",
    "d8e342656cf0",
}

# A bare basename cannot be placed by path alone. This document discusses one
# service and one package almost throughout, so prefer those when a basename is
# ambiguous, and say so in the note. Order matters: first match wins.
#
# The 2026-09-27 MCP document is about bin-ai-manager; within it, the D28 work
# lives in pkg/aicallhandler and the CRUD work in pkg/mcpserverhandler. A
# previous hand sweep resolved these basenames to the wrong package and produced
# 62 false "out of range" hits, so the preference list is explicit rather than
# alphabetical luck.
BASENAME_PREFERENCE = (
    "bin-ai-manager/pkg/aicallhandler/",
    "bin-ai-manager/pkg/mcpserverhandler/",
    "bin-ai-manager/pkg/aihandler/",
    "bin-ai-manager/pkg/listenhandler/",
    "bin-ai-manager/pkg/mcptoolhandler/",
    "bin-ai-manager/pkg/mcpoauthhandler/",
    "bin-ai-manager/pkg/dbhandler/",
    "bin-ai-manager/internal/config/",
    "bin-ai-manager/",
    "bin-openapi-manager/openapi/paths/mcpservers/",
    "bin-dbscheme-manager/",
    "bin-common-handler/",
)


def repo_root(start):
    out = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        cwd=start, capture_output=True, text=True, check=True,
    )
    return out.stdout.strip()


def index_files(root, extra_roots=()):
    """Map every tracked-looking source path to its absolute path.

    extra_roots let sibling repositories (monorepo-javascript for square-admin,
    monorepo-monitoring for api-validator) be checked for real instead of being
    suppressed as "external", which is how a broken cross-repo anchor used to
    pass unnoticed.
    """
    index = {}
    skip = {".git", "node_modules", "vendor", ".worktrees", "build", "dist",
            ".venv", "venv", "site-packages", "__pycache__"}
    for base in (root,) + tuple(extra_roots):
        if not os.path.isdir(base):
            continue
        for dirpath, dirnames, filenames in os.walk(base):
            dirnames[:] = [d for d in dirnames if d not in skip]
            for name in filenames:
                full = os.path.join(dirpath, name)
                index.setdefault(os.path.relpath(full, base), full)
    return index


def resolve(cited, index):
    """Return (abs_path, note) or (None, reason)."""
    if cited in index:
        return index[cited], "exact"
    matches = [rel for rel in index if rel == cited or rel.endswith("/" + cited)]
    if len(matches) == 1:
        return index[matches[0]], "suffix"
    if not matches:
        return None, "NOT-FOUND"
    for prefix in BASENAME_PREFERENCE:
        preferred = [rel for rel in matches if rel.startswith(prefix)]
        if len(preferred) == 1:
            return index[preferred[0]], "preferred:" + prefix
        if len(preferred) > 1:
            break
    return None, "AMBIGUOUS (%d candidates: %s)" % (
        len(matches), ", ".join(sorted(matches)[:4]),
    )


def check_commits(doc_text, root):
    """Report hex tokens that are not commits, INCLUDING ones git cannot find.

    The round-17 blocker was a commit hash cited three times that had never
    existed. An earlier version of this function only reported tokens git
    resolved to a non-commit, so a nonexistent hash (exit 128) was skipped and
    the check could not see the very defect it was written for.
    """
    bad = []
    for token in dict.fromkeys(HEXTOKEN.findall(doc_text)):
        if token in NON_COMMIT_HEX:
            continue
        out = subprocess.run(
            ["git", "cat-file", "-t", token],
            cwd=root, capture_output=True, text=True,
        )
        if out.returncode != 0:
            bad.append((token, "does not exist in this repository"))
        elif out.stdout.strip() != "commit":
            bad.append((token, "is a %s, not a commit" % out.stdout.strip()))
    return bad


def is_external(cited):
    return any(hint in cited for hint in EXTERNAL_HINTS)


# Each probe is a document this gate MUST reject (or, for the two marked 0,
# must accept). Three consecutive rounds of this document shipped a check that
# silently passed the very defect it was written to catch, so the gate now
# ships with the probes that refuted it.
SELF_TEST_PROBES = [
    (1, "cross-repo bogus line", "OUT OF RANGE",
     "`square-admin/src/views/ais/ais_detail.js:999999`"),
    # NOTE: this path must EXIST, or the probe passes on NOT-FOUND and proves
    # nothing about sibling-repo line-range resolution.
    (1, "api-validator bogus line", "OUT OF RANGE",
     "`api-validator/tests/scenarios/test_ai_lifecycle.py:999999`"),
    (1, "substring anchor pins nothing", "ANCHOR MISMATCH",
     "`bin-ai-manager/pkg/aicallhandler/helpers.go:21 resolveActiveAI`"),
    (1, "range END drifted", "OUT OF RANGE",
     "`bin-ai-manager/pkg/aicallhandler/helpers.go:194-999 resolveActiveAIForMcp`"),
    (1, "continuation binds to nearest PRECEDING path", "OUT OF RANGE",
     "`bin-common-handler/models/identity/identity.go:9` and `:73` then "
     "`bin-ai-manager/pkg/aicallhandler/mcp_tool.go:52`"),
    (1, "nonexistent commit", "does not exist",
     "commit `deadbeefcafe1` did it"),
    (1, "hint-listed path that EXISTS locally, bogus line "
        "(guards resolve-before-external ordering)", "OUT OF RANGE",
     "`square-main/public/skill.md:999999`"),
    (1, "reference beyond the last section 5 item", "references item",
     "## 5. x\n1. **a** t\n2. **b** t\n\n## 6. y\nSee item 7.\n"),
    (0, "correct anchor must PASS", None,
     "`bin-ai-manager/pkg/aicallhandler/start.go:375 refreshMcpToolMap`"),
    (0, "correct cross-repo citation must PASS", None,
     "`square-admin/src/views/ais/ais_detail.js:421`"),
]


def self_test(doc):
    """Run the gate against its own probes. Exit non-zero if any misbehaves.

    A probe that fails for an INCIDENTAL reason (an unresolvable path instead of
    the out-of-range line it was written to catch) is a false guarantee, so each
    failing probe also declares the message its failure must contain.
    """
    probe = os.path.join(os.path.dirname(os.path.abspath(doc)),
                         "_gate_self_test.md")
    failures = 0
    try:
        for want, name, reason, body in SELF_TEST_PROBES:
            with open(probe, "w", encoding="utf-8") as fh:
                fh.write(body if body.endswith("\n") else body + "\n")
            run = subprocess.run(
                [sys.executable, os.path.abspath(__file__), probe],
                capture_output=True, text=True,
            )
            got = run.returncode
            ok = (got != 0) == (want != 0)
            why = ""
            if ok and reason and reason not in run.stdout:
                # Failed, but not for the reason the probe exists to prove.
                ok = False
                why = " (wrong reason: expected %r)" % reason
            failures += 0 if ok else 1
            print("%-5s %-58s exit=%d want%s0%s"
                  % ("ok" if ok else "FAIL", name, got,
                     "!=" if want else "==", why))
    finally:
        if os.path.exists(probe):
            os.remove(probe)
    print("\n%s: %d/%d probes behaved correctly."
          % ("PASS" if not failures else "FAIL",
             len(SELF_TEST_PROBES) - failures, len(SELF_TEST_PROBES)))
    return 1 if failures else 0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("doc")
    ap.add_argument("--verbose", action="store_true",
                    help="print the source line every citation lands on")
    ap.add_argument("--self-test", action="store_true",
                    help="run the gate against deliberately broken probe "
                         "documents and verify it FAILS each one. Three "
                         "rounds shipped a check that silently passed its "
                         "own target defect; run this before trusting a "
                         "green result.")
    ap.add_argument("--strict-anchors", action="store_true",
                    help="fail when any citation lacks a symbol anchor")
    ap.add_argument("--extra-root", action="append", default=[],
                    metavar="DIR",
                    help="additional repository root to resolve citations in "
                         "(repeatable; e.g. ../monorepo-javascript). Defaults "
                         "to the sibling repos this document cites.")
    args = ap.parse_args()

    if args.self_test:
        return self_test(args.doc)

    doc_path = os.path.abspath(args.doc)
    root = repo_root(os.path.dirname(doc_path))
    extra = list(args.extra_root)
    if not extra:
        # root may be a worktree (…/monorepo/.worktrees/<branch>), so walk up
        # until we find a directory that actually holds the siblings. Without
        # this the documented bare invocation fails in every worktree, which
        # is where all review work happens.
        probe = root
        for _ in range(4):
            probe = os.path.dirname(probe)
            if not probe or probe == "/":
                break
            found = [os.path.join(probe, s)
                     for s in ("monorepo-javascript", "monorepo-monitoring")
                     if os.path.isdir(os.path.join(probe, s))]
            if found:
                extra = found
                break
    index = index_files(root, extra)
    text = open(doc_path, encoding="utf-8").read()

    total = external = anchored = unanchored = 0
    problems = []
    dropped = []
    warnings = []
    landings = []
    source_cache = {}

    def source_lines(path):
        if path not in source_cache:
            source_cache[path] = open(
                path, encoding="utf-8", errors="replace"
            ).read().split("\n")
        return source_cache[path]

    def check_one(doc_line, cited, path, num, symbol):
        """Check ONE line reference. Every call counts as one line-check."""
        nonlocal anchored, unanchored
        src = source_lines(path)
        if num < 1 or num > len(src):
            problems.append(
                "doc:%d  %s:%d OUT OF RANGE (file has %d lines)"
                % (doc_line, cited, num, len(src))
            )
            return
        landed = src[num - 1]
        if symbol:
            anchored += 1
            # Word-boundary match: a bare substring lets `resolveActiveAI`
            # pass against `resolveActiveAIIDFromAIcall`, which pins nothing.
            if not re.search(r"\b%s\b" % re.escape(symbol), landed):
                problems.append(
                    "doc:%d  %s:%d ANCHOR MISMATCH: expected %r, line reads %r"
                    % (doc_line, cited, num, symbol, landed.strip()[:80])
                )
                return
            if landed.lstrip().startswith(("//", "#", "*")):
                warnings.append(
                    "doc:%d  %s:%d anchor %r landed on a COMMENT, not the "
                    "construct" % (doc_line, cited, num, symbol)
                )
        else:
            unanchored += 1
        stripped = landed.strip()
        if stripped == "" or stripped in ("}", "})", "},", "};"):
            warnings.append(
                "doc:%d  %s:%d lands on %s"
                % (doc_line, cited, num,
                   "a blank line" if not stripped else "a lone %r" % stripped)
            )
        if args.verbose:
            landings.append("doc:%-5d %s:%-5d | %s"
                            % (doc_line, cited, num, stripped[:100]))

    last_path = None
    carried_span = None
    for doc_line, line in enumerate(text.split("\n"), start=1):
        # Paths cited on THIS line, with the offset where each appears.
        path_spans = []
        # Prose wraps, so a bullet may cite the path on one line and continue
        # with bare `:N` on the next. Carry the path across wrapped lines and
        # drop it at a paragraph or heading boundary, where a new subject
        # starts and binding would be a guess.
        if not line.strip() or line.lstrip().startswith(("#", "|", "```")):
            last_path = None
        for match in CITATION.finditer(line):
            cited = match.group("path")
            total += 1
            # Resolve FIRST. is_external is only a fallback for paths no root
            # has a copy of; consulting it first is what let a citation to
            # line 999999 of a real square-admin file pass.
            path, note = resolve(cited, index)
            if path is None:
                if is_external(cited):
                    external += 1
                    last_path = None
                    continue
                if note == "NOT-FOUND" and "/" not in cited:
                    problems.append(
                        "doc:%d  %s UNRESOLVED bare basename (add a path prefix, "
                        "or pass the repo that holds it with --extra-root)"
                        % (doc_line, cited)
                    )
                else:
                    problems.append("doc:%d  %s  %s" % (doc_line, cited, note))
                last_path = None
                continue
            last_path = (cited, path)
            # Record WHERE this path was cited, so a bare `:N` later on the
            # line binds to the path that precedes it rather than to whichever
            # path happens to come last. Prose means "nearest preceding".
            path_spans.append((match.start(), cited, path))
            # A symbol anchor counts only when it sits INSIDE the code span:
            # `start.go:375 refreshMcpToolMap`. Prose after a closing backtick
            # is not an anchor.
            symbol = match.group("symbol")
            if symbol and not (match.group("tick") and match.group("endtick")):
                symbol = None
            check_one(doc_line, cited, path, int(match.group("start")), symbol)
            if match.group("end"):
                # The range END gets the same anchor: a range whose start is
                # pinned and whose end drifts freely is half a check.
                check_one(doc_line, cited, path, int(match.group("end")), symbol)

        conts = list(CONTINUATION.finditer(line))
        bound = []
        for cont in conts:
            # Nearest path cited BEFORE this continuation on the same line;
            # if none, the path carried over from the previous wrapped line.
            prior = [sp for sp in path_spans if sp[0] < cont.start()]
            src = (prior[-1][1], prior[-1][2]) if prior else carried_span
            if src is None:
                continue
            bound.append((cont, src))
        for cont, (cited, path) in bound:
            check_one(doc_line, cited, path, int(cont.group("start")), None)
            if cont.group("end"):
                check_one(doc_line, cited, path, int(cont.group("end")), None)
        carried_span = last_path
        if len(bound) < len(conts):
            conts = [c for c in conts if all(c is not b[0] for b in bound)]
            # A bare `:N` with no resolvable path on its line is silently
            # unverifiable. Say so rather than dropping it.
            dropped.extend(
                "doc:%d  :%s has no resolvable file on its line (cite the path "
                "explicitly)" % (doc_line, c.group("start")) for c in conts
            )

    # Section 5 is an ordered markdown list whose items are referenced by
    # ordinal ("§5 item 9"). Renumbering by hand has silently broken those
    # references, so derive the valid range from the rendered list and flag
    # any reference outside it, plus any ordinal the list defines that nothing
    # points at (the fingerprint of a half-finished remap).
    sec5 = re.search(r"^## 5[^\n]*\n(.*?)(?=^## 6)", text, re.S | re.M)
    if sec5:
        n_items = len(re.findall(r"^\d+\. \*\*", sec5.group(1), re.M))
        refs = {int(m) for m in re.findall(r"item (\d+)", text)}
        for r in sorted(refs):
            if r > n_items:
                problems.append(
                    "§5 has %d items but the text references item %d"
                    % (n_items, r)
                )
        unused = [n for n in range(1, n_items + 1) if n not in refs]
        if unused:
            warnings.append(
                "§5 items %s are referenced by nothing -- check a renumbering "
                "did not leave references on the old ordinals"
                % ", ".join(str(n) for n in unused)
            )

    bad_commits = check_commits(text, root)

    # Every number below counts LINE REFERENCES (a `a.go:10-20` citation is
    # two), so the three add up to the total. They are not citation counts.
    print("citations extracted  : %d" % total)
    print("line refs checked    : %d" % (anchored + unanchored))
    print("  symbol-anchored    : %d (verified against the landed line)" % anchored)
    print("  unanchored         : %d (range-checked only -- drive this down)"
          % unanchored)
    print("skipped, no copy here: %d citations" % external)
    print("unverifiable bare :N : %d" % len(dropped))
    if args.verbose:
        print("\n--- landings ---")
        for entry in landings:
            print(entry)
    if dropped:
        print("\nUNVERIFIABLE CONTINUATIONS (%d):" % len(dropped))
        for entry in dropped:
            print("  " + entry)
    if warnings:
        print("\nWARNINGS (%d):" % len(warnings))
        for entry in warnings:
            print("  " + entry)
    if bad_commits:
        print("\nBAD COMMIT-LIKE TOKENS (%d):" % len(bad_commits))
        for token, why in bad_commits:
            print("  %s %s" % (token, why))
    if problems:
        print("\nPROBLEMS (%d):" % len(problems))
        for entry in problems:
            print("  " + entry)
    if problems or bad_commits:
        return 1
    if args.strict_anchors and unanchored:
        print("\nFAIL: --strict-anchors and %d citations carry no symbol anchor."
              % unanchored)
        return 1
    print("\nOK: every citation resolves; every anchored one names its line.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
