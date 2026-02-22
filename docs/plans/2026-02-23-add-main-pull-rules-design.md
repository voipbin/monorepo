# Design: Add Main Branch Pull Rules to Git Workflow

**Date:** 2026-02-23

## Problem

The current git workflow rules specify conflict-checking before PR/merge but don't clarify that this should happen from the worktree directory. Additionally, there's no rule for pulling the latest main into the local main repository after a PR is merged, which can cause the local main branch to drift from remote.

## Rules to Add

### Rule 1: Before PR/Merge — Fetch from Worktree

Update the existing "Pull Main and Check Conflicts Before PR/Merge" section to explicitly state that the conflict check must be done **from the worktree directory** (where the feature branch lives).

### Rule 2: After Merge — Pull Main from Main Directory

Add a new rule: after a PR is merged on GitHub, go to the **main repository directory** (`~/gitvoipbin/monorepo`) and run `git pull origin main` to keep the local main branch in sync with remote.

## Files to Update

1. `~/CLAUDE.md` — "Pull Main and Check Conflicts Before PR/Merge" section
2. `monorepo/CLAUDE.md` — Branch Management section
3. `docs/git-workflow-guide.md` — Merging to Main section

## Changes

Each file gets:
- Worktree context added to the pre-PR conflict check steps
- A new "After Merge" subsection with `cd ~/gitvoipbin/monorepo && git pull origin main`
