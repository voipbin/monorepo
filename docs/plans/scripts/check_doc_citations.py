#!/usr/bin/env python3
"""Resolve every `<file>:<line>` citation in a plan document against the worktree.

Citation drift recurred in five consecutive review rounds of
docs/plans/2026-09-27-mcp-lifecycle-and-docs-analysis.md: each commit that fixed
the prose also expanded comments, which shifted the lines the prose cited. Hand
re-derivation failed every time. Run this before committing any revision.

    python3 docs/plans/scripts/check_doc_citations.py docs/plans/<doc>.md

Exit status is 1 when any citation is out of range, so this can gate a commit.

What it checks:
  * every `path:N` and `path:N-M` citation resolves to a file in the repo
  * the cited line number exists in that file
  * (--verbose) the source line it lands on, so a human can spot anchors that
    drifted onto a blank line, a closing brace, or an unrelated statement
  * every 7-40 hex commit-like token is a real commit (`git cat-file -t`)

Ambiguous basenames (handler.go, start.go, main.go exist in several services)
are resolved by longest matching path suffix. A citation giving only a basename
that matches more than one file is reported as AMBIGUOUS rather than guessed:
add enough path prefix to the citation to disambiguate it.
"""

import argparse
import os
import re
import subprocess
import sys

CITATION = re.compile(r"`?([A-Za-z0-9_./-]+\.(?:go|py|yaml|yml|js|ts|rst|md|sql)):(\d+)(?:-(\d+))?`?")
HEXTOKEN = re.compile(r"\b([0-9a-f]{7,40})\b")

# Cited files that legitimately live outside this repository.
# square-admin paths appear under several shapes in this document
# (src/views/..., teamgraph/..., __tests__/...), all in monorepo-javascript.
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
)


def repo_root(start):
    out = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        cwd=start, capture_output=True, text=True, check=True,
    )
    return out.stdout.strip()


def index_files(root):
    """Map every tracked-looking source path to its absolute path."""
    index = {}
    skip = {".git", "node_modules", "vendor", ".worktrees"}
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in skip]
        for name in filenames:
            full = os.path.join(dirpath, name)
            index.setdefault(os.path.relpath(full, root), full)
    return index


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
    "bin-common-handler/",
)


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
    bad = []
    seen = set()
    for token in HEXTOKEN.findall(doc_text):
        if token in seen:
            continue
        seen.add(token)
        # Skip tokens that are plainly not hashes (uuid fragments are 8/12 chars
        # of hex too, so only report tokens git itself rejects as non-commits).
        out = subprocess.run(
            ["git", "cat-file", "-t", token],
            cwd=root, capture_output=True, text=True,
        )
        if out.returncode == 0 and out.stdout.strip() != "commit":
            bad.append((token, out.stdout.strip()))
    return bad


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("doc")
    ap.add_argument("--verbose", action="store_true",
                    help="print the source line every citation lands on")
    args = ap.parse_args()

    doc_path = os.path.abspath(args.doc)
    root = repo_root(os.path.dirname(doc_path))
    index = index_files(root)
    text = open(doc_path, encoding="utf-8").read()

    total = external = resolved = 0
    problems = []
    landings = []

    for lineno, line in enumerate(text.split("\n"), start=1):
        for match in CITATION.finditer(line):
            cited, start_s, end_s = match.group(1), match.group(2), match.group(3)
            total += 1
            if any(hint in cited for hint in EXTERNAL_HINTS):
                external += 1
                continue
            path, note = resolve(cited, index)
            if path is None:
                if note == "NOT-FOUND" and "/" not in cited:
                    external += 1  # bare name we cannot place; not a repo claim
                    continue
                problems.append("doc:%d  %s:%s  %s" % (lineno, cited, start_s, note))
                continue
            resolved += 1
            src = open(path, encoding="utf-8", errors="replace").read().split("\n")
            for num_s in (start_s, end_s):
                if num_s is None:
                    continue
                num = int(num_s)
                if num < 1 or num > len(src):
                    problems.append(
                        "doc:%d  %s:%d OUT OF RANGE (file has %d lines)"
                        % (lineno, cited, num, len(src))
                    )
                elif args.verbose:
                    landings.append(
                        "doc:%-5d %s:%-5d | %s"
                        % (lineno, cited, num, src[num - 1].strip()[:100])
                    )

    bad_commits = check_commits(text, root)

    print("citations extracted : %d" % total)
    print("resolved in-repo    : %d" % resolved)
    print("out-of-repo/skipped : %d" % external)
    if args.verbose:
        print("\n--- landings ---")
        for entry in landings:
            print(entry)
    if bad_commits:
        print("\nBAD COMMIT-LIKE TOKENS:")
        for token, kind in bad_commits:
            print("  %s resolves to a %s, not a commit" % (token, kind))
    if problems:
        print("\nPROBLEMS (%d):" % len(problems))
        for entry in problems:
            print("  " + entry)
        return 1
    print("\nOK: every in-repo citation resolves and lands inside its file.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
