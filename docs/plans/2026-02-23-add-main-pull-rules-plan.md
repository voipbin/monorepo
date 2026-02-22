# Add Main Branch Pull Rules - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add two git workflow rules: (1) clarify that pre-PR conflict checks run from the worktree, (2) add post-merge pull of main from the main directory.

**Architecture:** Documentation-only changes to three files that define git workflow rules.

**Tech Stack:** Markdown documentation

---

### Task 1: Update ~/CLAUDE.md — Add worktree context and post-merge rule

**Files:**
- Modify: `~/CLAUDE.md:21-31` (the "Pull Main and Check Conflicts" section)

**Step 1: Edit the conflict-check section**

Replace the existing section (lines 21-31) with this updated version that adds worktree context and a post-merge rule:

```markdown
### CRITICAL: Pull Main and Check Conflicts Before PR/Merge

**🚨 THIS IS A MANDATORY RULE - NO EXCEPTIONS 🚨**

**Before creating a PR or merging, ALWAYS pull the latest `main` and check for conflicts.**

Run these steps **from the worktree directory** (where your feature branch lives):

1. **Fetch latest main:** `git fetch origin main`
2. **Check for conflicts:** `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`
3. **Review what changed on main:** `git log --oneline HEAD..origin/main`
4. **If conflicts exist:** Rebase or merge main into your branch, resolve conflicts, and re-run the full verification workflow before proceeding.
5. **If no conflicts:** Proceed with PR creation or merge.

### CRITICAL: Pull Main After PR Merge

**After a PR is merged on GitHub, ALWAYS sync the local main branch:**

```bash
cd ~/gitvoipbin/monorepo && git pull origin main
```

This keeps the main repository directory in sync with remote so new worktrees start from the latest code.
```

**Step 2: Verify the edit**

Read `~/CLAUDE.md` and confirm the new sections are present and properly formatted.

**Step 3: Commit**

```bash
git add ~/CLAUDE.md
# Note: ~/CLAUDE.md is outside the repo, so this will be committed separately or skipped if not tracked
```

---

### Task 2: Update monorepo CLAUDE.md — Add worktree context and post-merge rule

**Files:**
- Modify: `CLAUDE.md:241-248` (the "Before creating a PR" conflict-check block)

**Step 1: Edit the conflict-check block**

Replace the existing block (lines 241-248) with this updated version:

```markdown
**CRITICAL: Before creating a PR or merging, ALWAYS pull the latest `main` and check for conflicts.**

This is mandatory — no exceptions. Run these steps **from the worktree directory** (where your feature branch lives):
1. **Fetch latest main:** `git fetch origin main`
2. **Check for conflicts:** `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`
3. **Review what changed on main:** `git log --oneline HEAD..origin/main`
4. **If conflicts exist:** Rebase or merge main into your branch, resolve conflicts, and re-run the full verification workflow before proceeding.
5. **If no conflicts:** Proceed with PR creation or merge.

**CRITICAL: After a PR is merged on GitHub, ALWAYS sync the local main branch:**

```bash
cd ~/gitvoipbin/monorepo && git pull origin main
```

This keeps the main repository directory in sync with remote so new worktrees start from the latest code.
```

**Step 2: Verify the edit**

Read `CLAUDE.md` around lines 241-260 and confirm the new content is correct.

---

### Task 3: Update docs/git-workflow-guide.md — Add post-merge section

**Files:**
- Modify: `docs/git-workflow-guide.md:242-278` (the "Merging to Main Branch" section)

**Step 1: Add post-merge rule at end of file**

After the "Only Merge When" subsection (line 278, end of file), append:

```markdown

### After PR is Merged

**After a PR is merged on GitHub, ALWAYS sync the local main branch:**

```bash
cd ~/gitvoipbin/monorepo && git pull origin main
```

This keeps the main repository directory in sync with remote so new worktrees start from the latest code.
```

**Step 2: Verify the edit**

Read the end of `docs/git-workflow-guide.md` and confirm the new section is present.

---

### Task 4: Commit all changes

**Step 1: Stage and commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-main-pull-rules
git add CLAUDE.md docs/git-workflow-guide.md docs/plans/
git commit -m "NOJIRA-add-main-pull-rules

Add git workflow rules for pulling main branch before and after PR merge.

- docs: Add worktree context to pre-PR conflict check in CLAUDE.md
- docs: Add post-merge 'pull main' rule to CLAUDE.md
- docs: Add post-merge section to git-workflow-guide.md
- docs: Add design and plan documents"
```

**Step 2: Also update ~/CLAUDE.md separately**

`~/CLAUDE.md` is outside the repo. Edit it directly (it's the user's private Claude instructions).

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-add-main-pull-rules
```

Then create PR with `gh pr create`.
