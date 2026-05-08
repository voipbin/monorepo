# README Update Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix three misleading sections in `README.md` for external developers — wrong scope framing, placeholder clone URL, and stale "coming soon" text.

**Architecture:** Pure markdown edit — no build pipeline, no tests, no Go toolchain. Four surgical string replacements in one file. Commit and push when done.

**Tech Stack:** Markdown, git

---

### Task 1: Reframe the intro paragraph

**Files:**
- Modify: `README.md` (lines 1–7)

**Step 1: Open README.md and locate the intro**

The current opening reads:

```
# VoIPbin Monorepo

Welcome to the VoIPBin monorepo — the unified backend codebase that powers all VoIPBin services.

VoIPBin is a cloud-native, scalable CPaaS platform designed for modern voice communication. This repository provides all the backend components for managing users, routing calls, handling media, running chatbots, and more — all in a single place.

This repository is a **monorepo** for all VoIPbin backend services. It provides a unified development environment for managing the services that power the VoIPbin platform.
```

**Step 2: Replace the intro with scoped text**

Replace lines 1–7 with:

```markdown
# VoIPbin Monorepo

VoIPbin is a cloud-native, open-source CPaaS platform for programmable voice communication. This repository is the primary backend services monorepo — it contains the 34 Go microservices that handle call routing, AI pipelines, conferencing, billing, messaging, and more.

This is one of VoIPbin's main repositories. It covers the backend service layer; other infrastructure (SIP proxy, Kamailio, Kubernetes configs) lives in separate repos.
```

**Step 3: Verify visually**

Open `README.md` and confirm:
- No mention of "unified backend codebase that powers all VoIPBin services"
- No mention of "all VoIPbin backend services"
- New text correctly scopes to "34 Go microservices" and "backend service layer"

**Step 4: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Update-readme
git add README.md
git commit -m "NOJIRA-Update-readme

- monorepo: Reframe intro to scope this as one of VoIPbin's main repos"
```

---

### Task 2: Fix the clone URL

**Files:**
- Modify: `README.md` (the "Clone the Repo" section, around line 133–137)

**Step 1: Locate the clone block**

Find:
```
   $ git clone https://github.com/your-org/voipbin.git
   $ cd voipbin
```

**Step 2: Replace with the real URL**

Replace with:
```
   $ git clone https://github.com/voipbin/monorepo.git
   $ cd monorepo
```

**Step 3: Verify**

Confirm no `your-org` or `voipbin.git` (old name) remains in the file:
```bash
grep -n "your-org\|cd voipbin" README.md
# Expected: no output
```

**Step 4: Commit**

```bash
git add README.md
git commit -m "NOJIRA-Update-readme

- monorepo: Fix placeholder clone URL to https://github.com/voipbin/monorepo.git"
```

---

### Task 3: Remove the stale "coming soon" sentence

**Files:**
- Modify: `README.md` (the "Understand the Architecture" subsection)

**Step 1: Locate the stale sentence**

Find:
```
We recommend reading the platform architecture guide (coming soon) before setup.
```

**Step 2: Delete that sentence entirely**

Remove the line. Do not replace it with anything.

**Step 3: Verify**

```bash
grep -n "coming soon" README.md
# Expected: no output
```

**Step 4: Commit**

```bash
git add README.md
git commit -m "NOJIRA-Update-readme

- monorepo: Remove stale 'platform architecture guide (coming soon)' sentence"
```

---

### Task 4: Check conflicts with main and push

**Step 1: Fetch latest main**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Update-readme
git fetch origin main
```

**Step 2: Check for conflicts**

```bash
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
# Expected: no output (no conflicts)
```

**Step 3: Review what changed on main since branch was created**

```bash
git log --oneline HEAD..origin/main
```

If anything touches `README.md`, resolve manually before proceeding.

**Step 4: Push branch**

```bash
git push -u origin NOJIRA-Update-readme
```

---

### Task 5: Create PR

**Step 1: Create PR via gh**

```bash
gh pr create \
  --title "NOJIRA-Update-readme" \
  --body "Fix three misleading sections in README.md for external developers.

- monorepo: Reframe intro to scope this as one of VoIPbin's main repos containing 34 Go microservices
- monorepo: Fix placeholder clone URL from github.com/your-org/voipbin.git to https://github.com/voipbin/monorepo.git
- monorepo: Remove stale 'platform architecture guide (coming soon)' sentence"
```

**Step 2: Confirm PR URL is returned**

Copy the URL and share with the user for review.
