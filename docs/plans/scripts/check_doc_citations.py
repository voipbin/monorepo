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

# Cited files that legitimately live outside this repository.
# square-admin paths appear under several shapes in this document
# (src/views/..., teamgraph/..., __tests__/...), all in monorepo-javascript;
# the api-validator files live in monorepo-monitoring.
EXTERNAL_HINTS = (
    "square-admin/",
    "api-validator/",
    "monorepo-monitoring/",
    "monorepo-javascript/",
    "teamgraph/",
    "__tests__/",
    "src/views/",
    "skill.md",
    "llms.txt",
    "cleanup_report.py",
    "test_mcpservers_lifecycle.py",
    "test_ai_lifecycle.py",
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


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("doc")
    ap.add_argument("--verbose", action="store_true",
                    help="print the source line every citation lands on")
    ap.add_argument("--strict-anchors", action="store_true",
                    help="fail when any citation lacks a symbol anchor")
    ap.add_argument("--extra-root", action="append", default=[],
                    metavar="DIR",
                    help="additional repository root to resolve citations in "
                         "(repeatable; e.g. ../monorepo-javascript). Defaults "
                         "to the sibling repos this document cites.")
    args = ap.parse_args()

    doc_path = os.path.abspath(args.doc)
    root = repo_root(os.path.dirname(doc_path))
    extra = list(args.extra_root)
    if not extra:
        parent = os.path.dirname(os.path.dirname(root))
        for sibling in ("monorepo-javascript", "monorepo-monitoring"):
            candidate = os.path.join(parent, sibling)
            if os.path.isdir(candidate):
                extra.append(candidate)
    index = index_files(root, extra)
    text = open(doc_path, encoding="utf-8").read()

    total = external = anchored = unanchored = 0
    problems = []
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
            if symbol not in landed:
                problems.append(
                    "doc:%d  %s:%d ANCHOR MISMATCH: expected %r, line reads %r"
                    % (doc_line, cited, num, symbol, landed.strip()[:80])
                )
                return
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

    for doc_line, line in enumerate(text.split("\n"), start=1):
        last_path = None
        for match in CITATION.finditer(line):
            cited = match.group("path")
            total += 1
            if is_external(cited):
                external += 1
                last_path = None
                continue
            path, note = resolve(cited, index)
            if path is None:
                if note == "NOT-FOUND" and "/" not in cited:
                    problems.append(
                        "doc:%d  %s UNRESOLVED bare basename (add a path prefix, "
                        "or add it to EXTERNAL_HINTS if it lives elsewhere)"
                        % (doc_line, cited)
                    )
                else:
                    problems.append("doc:%d  %s  %s" % (doc_line, cited, note))
                last_path = None
                continue
            last_path = (cited, path)
            # A symbol anchor counts only when it sits INSIDE the code span:
            # `start.go:375 refreshMcpToolMap`. Prose after a closing backtick
            # is not an anchor.
            symbol = match.group("symbol")
            if symbol and not (match.group("tick") and match.group("endtick")):
                symbol = None
            check_one(doc_line, cited, path, int(match.group("start")), symbol)
            if match.group("end"):
                check_one(doc_line, cited, path, int(match.group("end")), None)

        if last_path is not None:
            cited, path = last_path
            for cont in CONTINUATION.finditer(line):
                total += 1
                check_one(doc_line, cited, path, int(cont.group("start")), None)
                if cont.group("end"):
                    check_one(doc_line, cited, path, int(cont.group("end")), None)

    bad_commits = check_commits(text, root)

    print("citations extracted  : %d" % total)
    print("out-of-repo/skipped  : %d" % external)
    print("symbol-anchored      : %d (verified against the landed line)" % anchored)
    print("unanchored           : %d (range-checked only -- drive this down)"
          % unanchored)
    if args.verbose:
        print("\n--- landings ---")
        for entry in landings:
            print(entry)
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
