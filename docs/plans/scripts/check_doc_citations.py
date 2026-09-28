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
# most recent path cited on the same document line. An optional trailing symbol
# (`:327 resolveTools`) is the mirror image of SYMBOL_CONTINUATION below, and it
# was invisible for the same reason: the closing backtick used to be required
# immediately after the digits, so a trailing symbol broke this match, while the
# leading colon broke the other one. Nine citations in the MCP analysis document
# were written this way and a mutation to line 999999 kept every one of them
# green. Capture the symbol so it is verified rather than skipped.
CONTINUATION = re.compile(
    r"`:(?P<start>\d+)(?:[-,](?P<end>\d+))?(?: (?P<symbol>[A-Za-z_][A-Za-z0-9_.]*))?`"
)
# A symbol-qualified continuation, e.g. "`Update:205-207`" or "`Get:104-112`",
# also bound to the most recent path on the line. This spelling carried four
# stale citations through five review rounds untouched, because it looks like
# neither a path citation (no file extension) nor a bare continuation (the colon
# is not the first character), so BOTH patterns above skipped it and the line was
# never opened. The leading capital is what separates `Update:205` from a path;
# treat the symbol as an anchor so a shifted line is reported, not just counted.
SYMBOL_CONTINUATION = re.compile(
    r"`(?P<symbol>[A-Z][A-Za-z0-9_.]*):(?P<start>\d+)(?:[-,](?P<end>\d+))?`"
)
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
    (1, "plural ordinal list beyond the last section 5 item", "references item",
     "## 5. x\n1. **a** t\n2. **b** t\n\n## 6. y\nSee §5 items 1, 7 and 9.\n"),
    (1, "Q-label ordinal list beyond the last section 5 item", "references item",
     "## 5. x\n1. **a** t\n2. **b** t\n\n## 6. y\nSee §5 Q1, Q3, Q9.\n"),
    # Probe bodies are multi-line on purpose: a single-line body cannot reach
    # the cross-line path carry, which is the shape most of the real document
    # is made of.
    (1, "bare :N carried from an EARLIER line landing on a blank line",
     "CARRIED",
     "See `bin-ai-manager/pkg/aicallhandler/helpers_test.go:557 "
     "Test_aicallHandler_resolveActiveAIForMcp`\nand also `:127` here.\n"),
    (0, "bare :N carried from an EARLIER line landing on code must PASS", None,
     "See `bin-ai-manager/pkg/aicallhandler/mcp_tool.go:151 "
     "resolveActiveAIForMcp`\nand also `:127` here.\n"),
    # `helpers.go` exists in two packages (270 lines here, 25 there). Without
    # BASENAME_PREFERENCE the ambiguous basename resolves to the short one and
    # a CORRECT citation starts failing -- how the 62-false-hit episode began.
    (0, "ambiguous basename must resolve by preference, not first match", None,
     "`helpers.go:194 resolveActiveAIForMcp`"),
    # A CONTINUATION range has its own end check, separate from a citation's.
    (1, "continuation range END out of range",
     "OUT OF RANGE",
     "`bin-ai-manager/pkg/aicallhandler/helpers.go:194 resolveActiveAIForMcp` "
     "and `:20-999999`"),
    # The blank-line half of the carried check was probed from the start; the
    # LONE-BRACE half was not, and a single-token flip disabling it survived a
    # full self-test run. Both halves are probed now.
    (1, "bare :N carried from an EARLIER line landing on a lone brace",
     "CARRIED",
     "See `bin-ai-manager/pkg/aicallhandler/helpers.go:194 "
     "resolveActiveAIForMcp`\nand also `:270` there.\n"),
    # A carried RANGE is exempt from the brace rule (a block's last line is a
    # brace) but NOT from the blank-line rule -- a blank line ends no block in
    # any language, and one such landing escaped through the first exemption.
    #
    # This probe depends on the cited END line actually being blank in the real
    # tree, so it breaks silently whenever that file shifts: an earlier commit on
    # this branch deleted two struct fields and the old `:79-86` anchor landed on
    # a comment instead, leaving the probe green-by-accident while testing
    # nothing. Keep the END on a line that is blank for a structural reason (the
    # separator before a commented field group), and re-check it whenever
    # internal/config/main.go changes shape.
    (1, "carried range END landing on a blank line", "CARRIED",
     "See `bin-ai-manager/internal/config/main.go:151-154` here\n"
     "and the table at `:79-85` there.\n"),
    (0, "carried range END landing on a closing brace must PASS", None,
     "See `bin-ai-manager/pkg/aicallhandler/helpers.go:194 "
     "resolveActiveAIForMcp`\nand also `:196-225` there.\n"),
    # `start.go` lives in aicallhandler (1255 lines) and mcpoauthhandler (110).
    # BASENAME_PREFERENCE lists aicallhandler first ON PURPOSE; any reordering
    # (sorting it, emptying it, walk-order) sends this citation to the short
    # file. A preference probe that survives re-sorting proves nothing.
    (0, "ambiguous basename honours preference ORDER, not any order", None,
     "`start.go:1200`"),
    # One brace shape was probed and three were not, so each of the other three
    # could be deleted from the exempt list in silence. Every shape now has a
    # carried citation that must be REFUSED when it lands on that shape.
    (1, "carried bare :N landing on '})'", "CARRIED",
     "See `bin-ai-manager/pkg/aihandler/db_test.go:54 name`\n"
     "and also `:142` there.\n"),
    (1, "carried bare :N landing on '},'", "CARRIED",
     "See `bin-ai-manager/pkg/aihandler/db_test.go:142 name`\n"
     "and also `:54` there.\n"),
    (1, "carried bare :N landing on '};'", "CARRIED",
     "See `square-admin/src/views/ais/ais_detail.js:45 export`\n"
     "and also `:1422` there.\n"),
    # `doc:N` self-references. Adding this check without probing it would have
    # repeated, in the very commit that closes the class, the mistake the class
    # is made of.
    (1, "doc:N with no anchor is refused", "carries no anchor",
     "Line one.\nSee doc:1 for the statement.\n"),
    (1, "doc:N whose anchor no longer matches is refused", "ANCHOR MISMATCH",
     "Line one.\nSee doc:1 \"absent phrase\" for the statement.\n"),
    (1, "doc:N past the end of the document is refused", "outside this document",
     "Line one.\nSee doc:9999 \"anything\" here.\n"),
    (0, "doc:N whose anchor still matches must PASS", None,
     "Line one.\nSee doc:1 \"Line one\" for the statement.\n"),
    (0, "correct anchor must PASS", None,
     "`bin-ai-manager/pkg/aicallhandler/start.go:375 refreshMcpToolMap`"),
    # `Symbol:N` continuations. This spelling was invisible to both the path and
    # the bare-continuation patterns, so four stale citations written this way
    # survived five review rounds unopened. A probe per outcome, because a rule
    # nobody exercises is a rule that silently stops working (this file's own
    # `:79-86` probe did exactly that two commits ago).
    #
    # The PASS probe names a symbol that really is ON its cited line. Note what
    # that implies for authors: `Update:205-207` does NOT pass, because a Go
    # function's name lives on its signature, not on the lines inside it. That is
    # the intended pressure -- cite the file and line, or anchor on a symbol the
    # line actually contains.
    (1, "Symbol:N continuation whose line moved is refused", "ANCHOR MISMATCH",
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `IsValid:1`\n"),
    (1, "Symbol:N continuation past EOF is refused", "OUT OF RANGE",
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `IsValid:999999`\n"),
    (0, "Symbol:N continuation naming its real line must PASS", None,
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `IsValid:205`\n"),
    # The mirror spelling, `:N Symbol`. Nine citations in the MCP analysis
    # document were written this way and NONE was checked: the trailing symbol
    # broke CONTINUATION's closing backtick, and the leading colon broke
    # SYMBOL_CONTINUATION's [A-Z] start. The two rules had a gap exactly between
    # them. A range's END is deliberately not symbol-checked, since the symbol
    # describes the construct the range opens, so that case gets a PASS probe too.
    (1, ":N Symbol continuation whose line moved is refused", "ANCHOR MISMATCH",
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `:1 IsValid`\n"),
    (1, ":N Symbol continuation past EOF is refused", "OUT OF RANGE",
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `:999999 IsValid`\n"),
    (0, ":N Symbol continuation naming its real line must PASS", None,
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `:205 IsValid`\n"),
    (0, ":N-M Symbol must not check the symbol against the range end", None,
     "`bin-ai-manager/pkg/mcpserverhandler/handler.go:48` and `:205-207 IsValid`\n"),
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

    def judge(want, reason, body):
        """Run one probe and decide whether it behaved. Returns (ok, why, rc).

        Extracted so the harness's OWN rules can be tested below: while this
        logic was inline, disabling the reason check or hardcoding the total
        left the self-test printing a full green result.
        """
        try:
            with open(probe, "w", encoding="utf-8") as fh:
                fh.write(body if body.endswith("\n") else body + "\n")
            run = subprocess.run(
                [sys.executable, os.path.abspath(__file__), probe],
                capture_output=True, text=True,
            )
        finally:
            if os.path.exists(probe):
                os.remove(probe)
        ok = (run.returncode != 0) == (want != 0)
        why = ""
        if ok and reason and reason not in run.stdout:
            # Failed, but not for the reason the probe exists to prove.
            ok = False
            why = " (wrong reason: expected %r)" % reason
        return ok, why, run.returncode

    def judge_report(body, must_print, clean_body):
        """Decide whether a REPORT-only channel behaved. Returns (ok, why).

        Also extracted, for the same reason: inline, `ok = True` here printed a
        green line for four channels at once.
        """
        outs = []
        for b in (body, clean_body):
            try:
                with open(probe, "w", encoding="utf-8") as fh:
                    fh.write(b)
                outs.append(subprocess.run(
                    [sys.executable, os.path.abspath(__file__), probe],
                    capture_output=True, text=True,
                ).stdout)
            finally:
                if os.path.exists(probe):
                    os.remove(probe)
        if must_print not in outs[0]:
            return False, ""
        if must_print in outs[1]:
            # Fires on a document that has nothing to report: the assertion is
            # unconditional, so it proves nothing.
            return False, " (negative control ALSO reported it)"
        return True, ""

    failures = 0
    performed = 0
    for want, name, reason, body in SELF_TEST_PROBES:
        performed += 1
        ok, why, got = judge(want, reason, body)
        failures += 0 if ok else 1
        print("%-5s %-58s exit=%d want%s0%s"
              % ("ok" if ok else "FAIL", name, got,
                 "!=" if want else "==", why))
    # Some channels REPORT without failing (the unbindable-`:N` list, the
    # section 5 orphan warning). An exit-code probe cannot see those deleted,
    # so assert on stdout directly against a body that must produce them.
    # Each carries a NEGATIVE CONTROL: a body that must NOT produce the
    # message. Without one, replacing the whole assertion with `ok = True`
    # keeps printing a green line, which is how three of these went unprobed.
    for name, body, must_print, clean_body in (
        # A neighbour ABOVE the cited line and a neighbour BELOW must each be
        # reported on their own; one probe whose symbol repeats in BOTH
        # directions stays green when either half of the window is amputated.
        ("anchor repeating ABOVE the cited line is reported",
         "`bin-ai-manager/pkg/aicallhandler/helpers.go:36 CurrentMemberID`\n",
         "NOT unique",
         "`bin-ai-manager/pkg/aicallhandler/helpers.go:36 member_id`\n"),
        ("anchor repeating BELOW the cited line is reported",
         "`bin-ai-manager/pkg/aicallhandler/mcp_tool.go:122 toolHandleMcpCall`\n",
         "NOT unique",
         "`bin-ai-manager/pkg/aicallhandler/mcp_tool.go:127 messageContent`\n"),
        ("unbindable bare :N is reported",
         "A paragraph naming no path at all.\n\nThen `:4321` alone.\n",
         "has no resolvable file on its line",
         "`bin-ai-manager/pkg/aicallhandler/helpers.go:194 "
         "resolveActiveAIForMcp`\n"),
        ("section 5 ordinal nothing references is reported",
         "## 5. x\n1. **a** t\n2. **b** t\n\n## 6. y\nSee item 1.\n",
         "referenced by nothing",
         "## 5. x\n1. **a** t\n2. **b** t\n\n## 6. y\nSee item 1 and item 2.\n"),
    ):
        performed += 1
        ok, why = judge_report(body, must_print, clean_body)
        failures += 0 if ok else 1
        print("%-5s %-58s (report channel)%s"
              % ("ok" if ok else "FAIL", name, why))

    # THE HARNESS'S OWN RULES. With the reason check disabled, or matched
    # case-insensitively, or the total hardcoded, or the report assertions made
    # unconditional, this self-test still printed a full green result. Each rule
    # is now exercised through `judge`, so weakening it is a FAILURE here.
    out_of_range = "`bin-ai-manager/pkg/aicallhandler/helpers.go:999999`"
    for name, args, want_ok in (
        # Right answer, right reason: must be accepted.
        ("harness accepts a probe failing for its declared reason",
         (1, "OUT OF RANGE", out_of_range), True),
        # Right answer (it does fail), wrong reason: must be REJECTED, which is
        # the rule that makes every `reason` in the table load-bearing.
        ("harness rejects a probe failing for the WRONG reason",
         (1, "ANCHOR MISMATCH", out_of_range), False),
        # Same text in the wrong case: rejected, or `reason` stops being exact.
        ("harness reason matching is case-SENSITIVE",
         (1, "out of range", out_of_range), False),
        # A probe expected to pass that instead fails: rejected.
        ("harness rejects a pass-probe that fails",
         (0, None, out_of_range), False),
    ):
        performed += 1
        ok = judge(*args)[0]
        good = ok == want_ok
        failures += 0 if good else 1
        print("%-5s %-58s (harness)" % ("ok" if good else "FAIL", name))

    # The report-channel rules, likewise. An unconditional `ok` there muted four
    # channels at once while still printing four green lines.
    anchor_repeats = ("`bin-ai-manager/pkg/aicallhandler/"
                      "mcp_tool.go:122 toolHandleMcpCall`\n")
    anchor_unique = ("`bin-ai-manager/pkg/aicallhandler/"
                     "mcp_tool.go:127 messageContent`\n")
    for name, args, want_ok in (
        ("harness accepts a report channel that fires only when it should",
         (anchor_repeats, "NOT unique", anchor_unique), True),
        # Message absent from the positive body: must be rejected.
        ("harness rejects a report channel that never fires",
         (anchor_unique, "NOT unique", anchor_unique), False),
        # Message present in BOTH bodies: the assertion is unconditional.
        ("harness rejects a report channel with no negative control",
         (anchor_repeats, "citations extracted", anchor_unique), False),
    ):
        performed += 1
        good = judge_report(*args)[0] == want_ok
        failures += 0 if good else 1
        print("%-5s %-58s (harness)" % ("ok" if good else "FAIL", name))

    # A probe DELETED from the table lowers both the count and the total, so the
    # run stays green with a smaller number. An earlier version of this block
    # "cross-checked" `performed` against `len(SELF_TEST_PROBES) + ...`, which
    # is the same quantity and therefore proved nothing -- exactly the false
    # guarantee this harness exists to refuse, committed into the harness.
    #
    # The number below is the ONE figure here that must be maintained by hand:
    # it is written down, not derived, so removing a probe fails this run.
    # No self-test can do better -- a check cannot notice its own absence
    # unless something outside it remembers how many there should be.
    EXPECTED_CHECKS = 45
    if performed != EXPECTED_CHECKS:
        failures += 1
        print("FAIL  %-58s (harness)"
              % ("performed %d checks, expected %d -- a probe was added or "
                 "removed without updating EXPECTED_CHECKS"
                 % (performed, EXPECTED_CHECKS)))
    print("\n%s: %d/%d checks behaved correctly."
          % ("PASS" if not failures else "FAIL",
             performed - failures, performed))
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

    def check_one(doc_line, cited, path, num, symbol, carried=False,
                  carried_end=False):
        """Check ONE line reference. Every call counts as one line-check.

        `carried` marks a bare `:N` bound to a path cited on an EARLIER line.
        Nothing in the prose repeats the filename there, so a wrong binding is
        invisible to a reader; the only evidence that the pairing is right is
        that the landing looks like a construct. A carried reference landing on
        a blank line or a lone brace is therefore an error, not a warning.
        """
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
            # An anchor whose symbol also appears nearby does not prevent the
            # drift it exists to catch: the citation would still "match" after
            # the construct moved a few lines. Report it so anchors get chosen
            # from symbols that are unique in their window.
            lo, hi = max(0, num - 1 - 5), min(len(src), num + 5)
            near = [i + 1 for i in range(lo, hi)
                    if i + 1 != num
                    and re.search(r"\b%s\b" % re.escape(symbol), src[i])]
            if near:
                warnings.append(
                    "doc:%d  %s:%d anchor %r is NOT unique within +/-5 (also "
                    "%s) -- a small drift would still pass"
                    % (doc_line, cited, num, symbol,
                       ", ".join(str(n) for n in near)))
            if landed.lstrip().startswith(("//", "#", "*")):
                warnings.append(
                    "doc:%d  %s:%d anchor %r landed on a COMMENT, not the "
                    "construct" % (doc_line, cited, num, symbol)
                )
        else:
            unanchored += 1
        stripped = landed.strip()
        if stripped == "" or stripped in ("}", "})", "},", "};"):
            what = "a blank line" if not stripped else "a lone %r" % stripped
            # A range END is expected to be a block's closing brace, so a brace
            # landing proves nothing there -- but a BLANK line is not a block
            # end in any language, and one such landing (a carried range whose
            # path came from the previous line) escaped the first version of
            # this check precisely through the end exemption.
            if carried_end and stripped:
                pass
            elif carried or carried_end:
                problems.append(
                    "doc:%d  %s:%d CARRIED bare :N lands on %s -- the path came "
                    "from an earlier line, so this binding is unproven (cite "
                    "the path explicitly)" % (doc_line, cited, num, what))
                return
            warnings.append(
                "doc:%d  %s:%d lands on %s" % (doc_line, cited, num, what))
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
        # `Update:205-207` binds to the same carried path as `:205-207` does, but
        # carries a symbol we can verify, so route it through the same binding
        # logic and hand check_one the symbol as an anchor.
        sym_conts = list(SYMBOL_CONTINUATION.finditer(line))
        bound = []
        for cont in conts:
            # Nearest path cited BEFORE this continuation on the same line;
            # if none, the path carried over from the previous wrapped line.
            prior = [sp for sp in path_spans if sp[0] < cont.start()]
            src = (prior[-1][1], prior[-1][2]) if prior else carried_span
            if src is None:
                continue
            bound.append((cont, src, not prior))
        for cont, (cited, path), carried in bound:
            check_one(doc_line, cited, path, int(cont.group("start")),
                      cont.group("symbol"), carried=carried)
            if cont.group("end"):
                # The END of a range is expected to be a block's closing brace,
                # so landing quality proves nothing there. A trailing symbol
                # describes the construct the range OPENS, so it is not expected
                # on the closing line and must not be checked against it.
                check_one(doc_line, cited, path, int(cont.group("end")), None,
                          carried_end=carried)
        for cont in sym_conts:
            # A SYMBOL_CONTINUATION overlapping a real path citation is that
            # citation's own text (e.g. the `Main.go:12` inside a longer path),
            # already checked above. Skip it rather than double-reporting.
            if any(sp[0] <= cont.start() < sp[0] + 1 for sp in path_spans):
                continue
            if any(not (cont.end() <= c.start() or c.end() <= cont.start())
                   for c in conts):
                continue
            prior = [sp for sp in path_spans if sp[0] < cont.start()]
            src = (prior[-1][1], prior[-1][2]) if prior else carried_span
            if src is None:
                continue
            cited, path = src
            check_one(doc_line, cited, path, int(cont.group("start")),
                      cont.group("symbol"), carried=not prior)
        carried_span = last_path
        if len(bound) < len(conts):
            conts = [c for c in conts if all(c is not b[0] for b in bound)]
            # A bare `:N` with no resolvable path on its line is silently
            # unverifiable. Say so rather than dropping it.
            dropped.extend(
                "doc:%d  :%s has no resolvable file on its line (cite the path "
                "explicitly)" % (doc_line, c.group("start")) for c in conts
            )

    # `doc:N` points at THIS document's own line numbers -- the one citation
    # class no mechanism covered, and the only one still producing findings.
    # It is strictly worse than a file citation: any edit above the target
    # shifts it, and the commit that ADDS lines is usually the one repairing a
    # citation, so the repair invalidates itself. Require a quoted anchor and
    # verify the line still contains it.
    doc_lines = text.splitlines()
    for m in re.finditer(r"doc:(\d+)(?:\s+\u201c([^\u201d]+)\u201d|\s+\"([^\"]+)\")?",
                         text):
        doc_line = text[:m.start()].count("\n") + 1
        num = int(m.group(1))
        anchor = m.group(2) or m.group(3)
        if num < 1 or num > len(doc_lines):
            problems.append(
                "doc:%d  doc:%d is outside this document (%d lines)"
                % (doc_line, num, len(doc_lines))
            )
            continue
        landed = doc_lines[num - 1]
        if not anchor:
            problems.append(
                "doc:%d  doc:%d carries no anchor -- quote a phrase from the "
                "target line so an edit above it cannot silently re-point this"
                % (doc_line, num)
            )
        elif anchor not in landed:
            problems.append(
                "doc:%d  doc:%d ANCHOR MISMATCH: expected %r, line reads %r"
                % (doc_line, num, anchor, landed.strip()[:60])
            )
        elif not landed.strip():
            problems.append(
                "doc:%d  doc:%d lands on a blank line" % (doc_line, num)
            )

    # Section 5 is an ordered markdown list whose items are referenced by
    # ordinal ("§5 item 9"). Renumbering by hand has silently broken those
    # references, so derive the valid range from the rendered list and flag
    # any reference outside it, plus any ordinal the list defines that nothing
    # points at (the fingerprint of a half-finished remap).
    sec5 = re.search(r"^## 5[^\n]*\n(.*?)(?=^## 6)", text, re.S | re.M)
    if sec5:
        n_items = len(re.findall(r"^\d+\. \*\*", sec5.group(1), re.M))
        # Plural and Q-label forms hid a stale ordinal for two rounds running:
        # `item (\d+)` sees nothing in "§5 items 1, 3, 4, 7, 8" or "§5 Q1, Q3".
        # Match the whole enumeration, then every number inside it.
        refs = {int(m) for m in re.findall(r"item (\d+)", text)}
        for run in re.findall(r"§5 items? ((?:Q?\d+[,\s]+(?:and\s+)?)*Q?\d+)",
                              text):
            refs.update(int(n) for n in re.findall(r"\d+", run))
        for run in re.findall(r"§5 ((?:Q\d+[,\s]+(?:and\s+)?)+Q\d+)", text):
            refs.update(int(n) for n in re.findall(r"\d+", run))
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
