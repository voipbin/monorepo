# Fix Dependabot Alerts Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Resolve all fixable dependabot security alerts by bumping `filippo.io/edwards25519` in 30 Go services and upgrading `pipecat-ai` to 0.0.103 in bin-pipecat-manager.

**Architecture:** Two independent fix categories — Go transitive dependency bump (mechanical) and Python pipecat-ai upgrade (with one code change to preserve VAD behavior).

**Tech Stack:** Go modules, uv (Python package manager), pipecat-ai framework

---

## Worktree

All work happens in: `~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts/`

---

## Part A: Go — Bump filippo.io/edwards25519 to v1.1.1

### Task 1: Bump edwards25519 in bin-common-handler

bin-common-handler is the shared library — update it first since other services depend on it.

**Files:**
- Modify: `bin-common-handler/go.mod`
- Modify: `bin-common-handler/go.sum`
- Modify: `bin-common-handler/vendor/`

**Step 1: Bump the dependency**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts/bin-common-handler
go get filippo.io/edwards25519@v1.1.1
```

**Step 2: Run full verification**

```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass. edwards25519 v1.1.1 appears in go.mod/go.sum.

**Step 3: Verify the bump**

```bash
grep 'filippo.io/edwards25519' go.mod
```

Expected: `filippo.io/edwards25519 v1.1.1 // indirect`

---

### Task 2: Bump edwards25519 in all 29 remaining Go services

Each service needs the same treatment. Process them in batches. The services are:

```
bin-agent-manager
bin-ai-manager
bin-api-manager
bin-billing-manager
bin-call-manager
bin-campaign-manager
bin-conference-manager
bin-contact-manager
bin-conversation-manager
bin-customer-manager
bin-email-manager
bin-flow-manager
bin-hook-manager
bin-message-manager
bin-number-manager
bin-outdial-manager
bin-pipecat-manager
bin-queue-manager
bin-registrar-manager
bin-route-manager
bin-sentinel-manager
bin-storage-manager
bin-tag-manager
bin-talk-manager
bin-transcribe-manager
bin-transfer-manager
bin-tts-manager
bin-webhook-manager
voip-asterisk-proxy
```

**Step 1: For EACH service, run:**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts/<service-name>
go get filippo.io/edwards25519@v1.1.1
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Use parallel agents — each agent handles one service. All 29 are independent.

**Step 2: Verify all services bumped**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts
grep -r 'filippo.io/edwards25519 v1.1.0' --include='go.mod'
```

Expected: No results (all should be v1.1.1 now).

---

### Task 3: Commit Part A

**Step 1: Stage and commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts
git add -A
git commit -m "NOJIRA-fix-dependabot-alerts

Bump filippo.io/edwards25519 from v1.1.0 to v1.1.1 across all Go services
to resolve dependabot security alerts (low severity).

- bin-common-handler: Bump filippo.io/edwards25519 to v1.1.1
- bin-agent-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-ai-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-api-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-billing-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-call-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-campaign-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-conference-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-contact-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-conversation-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-customer-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-email-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-flow-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-hook-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-message-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-number-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-outdial-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-pipecat-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-queue-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-registrar-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-route-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-sentinel-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-storage-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-tag-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-talk-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-transcribe-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-transfer-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-tts-manager: Bump filippo.io/edwards25519 to v1.1.1
- bin-webhook-manager: Bump filippo.io/edwards25519 to v1.1.1
- voip-asterisk-proxy: Bump filippo.io/edwards25519 to v1.1.1"
```

---

## Part B: Python — Upgrade pipecat-ai to 0.0.103

### Task 4: Update pyproject.toml and regenerate lock file

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/pyproject.toml` (line 9)
- Modify: `bin-pipecat-manager/scripts/pipecat/uv.lock` (regenerated)

**Step 1: Update pyproject.toml**

Change line 9 from:
```
    "pipecat-ai[silero,deepgram,openai,cartesia,websocket,google]>=0.0.82",
```
To:
```
    "pipecat-ai[silero,deepgram,openai,cartesia,websocket,google]>=0.0.103",
```

**Step 2: Regenerate lock file**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts/bin-pipecat-manager/scripts/pipecat
uv lock
```

Expected: Lock file updated with pipecat-ai 0.0.103, protobuf >= 5.29.6, pillow >= 12.1.1.

**Step 3: Verify patched versions in lock file**

```bash
grep -A2 '^name = "protobuf"' uv.lock
grep -A2 '^name = "pillow"' uv.lock
grep -A2 '^name = "pipecat-ai"' uv.lock
```

Expected: protobuf >= 5.29.6, pillow >= 12.1.1, pipecat-ai 0.0.103.

---

### Task 5: Fix SileroVADAnalyzer to preserve VAD stop_secs behavior

Pipecat 0.0.102 changed the default `stop_secs` from 0.8 to 0.2 seconds. Without this fix, conversations would end prematurely on brief pauses.

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py` (lines 30, 81)

**Step 1: Add VADParams import**

At line 30, change:
```python
from pipecat.audio.vad.silero import SileroVADAnalyzer
```
To:
```python
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.audio.vad.vad_analyzer import VADParams
```

**Step 2: Set explicit stop_secs**

At line 81, change:
```python
            vad_analyzer = SileroVADAnalyzer()
```
To:
```python
            vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=0.8))
```

This preserves the pre-0.0.102 behavior where silence must last 0.8 seconds before a turn ends.

---

### Task 6: Commit Part B

**Step 1: Stage and commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts
git add bin-pipecat-manager/scripts/pipecat/pyproject.toml bin-pipecat-manager/scripts/pipecat/uv.lock bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-fix-dependabot-alerts

Upgrade pipecat-ai from 0.0.100 to 0.0.103 to resolve dependabot alerts
for protobuf and pillow. Explicitly set VAD stop_secs=0.8 to preserve
current voice activity detection behavior after pipecat 0.0.102 changed
the default from 0.8 to 0.2 seconds.

- bin-pipecat-manager: Upgrade pipecat-ai to >=0.0.103
- bin-pipecat-manager: Set explicit VAD stop_secs=0.8 in SileroVADAnalyzer
- bin-pipecat-manager: Regenerate uv.lock with patched protobuf and pillow"
```

---

## Part C: Push and Create PR

### Task 7: Push and create PR

**Step 1: Fetch main and check for conflicts**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-dependabot-alerts
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

Expected: No conflicts.

**Step 2: Push branch**

```bash
git push -u origin NOJIRA-fix-dependabot-alerts
```

**Step 3: Create PR**

```bash
gh pr create --title "NOJIRA-fix-dependabot-alerts" --body "Fix all fixable dependabot security alerts by bumping Go and Python dependencies.

- bin-common-handler: Bump filippo.io/edwards25519 v1.1.0 -> v1.1.1
- bin-*-manager (29 services): Bump filippo.io/edwards25519 v1.1.0 -> v1.1.1
- voip-asterisk-proxy: Bump filippo.io/edwards25519 v1.1.0 -> v1.1.1
- bin-pipecat-manager: Upgrade pipecat-ai 0.0.100 -> 0.0.103
- bin-pipecat-manager: Set explicit VAD stop_secs=0.8 to preserve behavior
- bin-pipecat-manager: Regenerate uv.lock (fixes protobuf and pillow CVEs)

Note: nltk critical alert (CVE for Zip Slip) remains open — no upstream fix available (3.9.2 is latest)."
```
