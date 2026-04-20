# RTC Architecture Documentation Update Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Update `architecture_rtc.rst` to accurately reflect the production SIP topology and document three missing concepts: bidirectional Kamailio↔Asterisk decoupling, RE-INVITE within-dialog routing, and the Kamailio Proxy sidecar.

**Architecture:** All changes are confined to a single RST file. No new files are created. After editing the RST, the HTML must be rebuilt and committed alongside the source changes.

**Tech Stack:** reStructuredText, Sphinx, Python (for HTML rebuild)

---

## Context

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs`

**Target file:** `bin-api-manager/docsdev/source/architecture_rtc.rst`

**Design doc:** `docs/plans/2026-04-21-rtc-architecture-docs-update-design.md`

**Key architectural facts to document:**

1. **Full SIP topology:**
   - Inbound: `Client → External LB → Kamailio Farm → Internal Asterisk LB → Asterisk Farm`
   - Return: `Asterisk Farm → Internal Kamailio LB → Kamailio Farm → External LB → Client`

2. **Kamailio uses dispatcher module** for load balancing between Kamailio instances. For routing to Asterisk, it uses **one single slot pointing to the Internal Asterisk LB** — not individual Asterisk addresses.

3. **Bidirectional decoupling:** Asterisk sends responses to the Internal Kamailio LB (not individual Kamailios). Result: neither side knows the other's instance list → both can scale independently with zero config changes.

4. **RE-INVITE routing:** In-dialog requests carry SIP Route headers set by Asterisk during dialog setup. Kamailio reads these headers and forwards directly to the specific Asterisk instance, bypassing the Internal Asterisk LB. No risk of dialog going to the wrong Asterisk.

5. **Kamailio Proxy:** A Go sidecar service (not in the SIP path) that monitors provider health via SIP OPTIONS probes and reports to `bin-route-manager`. Added in PR #787.

---

### Task 1: Replace VoIP Stack Overview Diagram

**File:**
- Modify: `bin-api-manager/docsdev/source/architecture_rtc.rst` (lines 17–59)

**Step 1: Read the current file to understand line ranges**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs
grep -n "" bin-api-manager/docsdev/source/architecture_rtc.rst | head -70
```

**Step 2: Replace the VoIP Stack Overview section**

Replace everything between `.. code::` (the first one, around line 17) and `.. image:: _static/images/architecture_rtc_voip.png` with the new diagram below.

New diagram content:

```rst
.. code::

    Full SIP Signaling Topology:

    +------------------+
    |  SIP/WebRTC      |
    |  Client          |
    +--------+---------+
             | INVITE / RE-INVITE / 200 OK
             v
    +------------------+
    |  External        |
    |  Load Balancer   |  <-- Internet-facing edge
    |  (Signal GW)     |
    +--------+---------+
             | Distributes to Kamailio Farm
             v
    +-------------------------------------+
    |         Kamailio Farm               |
    |  (Kamailio 1 / Kamailio 2 / ...)   |
    |                                     |
    |  o Dispatcher module for            |
    |    inter-Kamailio balancing         |
    |  o Single slot → Asterisk LB        |
    |    (no per-Asterisk entries)        |
    +----+--------------------------------+
         |                       ^
         | INVITE (new dialog)   | 200 OK / 200 OK(RE-INVITE)
         v                       |
    +------------------+   +-----+------------+
    |  Internal        |   |  Internal        |
    |  Asterisk LB     |   |  Kamailio LB     |
    +--------+---------+   +------------------+
             |                       ^
             v                       |
    +------------------+             |
    |  Asterisk Farm   +-------------+
    |  (Call PBX)      |  Responses sent to
    +------------------+  Internal Kamailio LB
```

**Step 3: Update the Key Characteristics bullets** to add the two new points:

After the existing `* **Zero-Downtime**` bullet, add:

```rst
* **Dispatcher-Based Kamailio Distribution**: Kamailio instances use the built-in dispatcher module to balance traffic across the farm
* **Single-Slot Asterisk Routing**: Kamailio routes to Asterisk via a single dispatcher slot pointing to the Internal Asterisk LB — not individual Asterisk addresses
* **Bidirectional Decoupling**: Asterisk sends responses to the Internal Kamailio LB, not to individual Kamailios. Neither side knows the other's instance list, enabling independent scaling of both farms at any time
```

**Step 4: Update the Traffic Flow numbered list** to reflect the correct topology:

Replace the existing 4-item list with:

```rst
1. **Inbound Signaling**: External Load Balancer distributes incoming SIP traffic to the Kamailio Farm
2. **New Call Routing**: Kamailio forwards new INVITEs to the Internal Asterisk LB (single dispatcher slot), which distributes to the Asterisk Farm
3. **Response Path**: Asterisk sends responses (200 OK) to the Internal Kamailio LB, which distributes to any Kamailio instance
4. **Media Setup**: RTPEngine handles RTP media streams and codec transcoding
5. **Call Control**: Asterisk manages call state and conference bridges
```

**Step 5: Verify the section looks correct**

```bash
grep -n "Bidirectional\|dispatcher\|single slot\|Single.Slot\|Internal Asterisk\|Internal Kamailio" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

Expected: lines found with all new keywords.

**Step 6: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs
git add bin-api-manager/docsdev/source/architecture_rtc.rst
git commit -m "NOJIRA-update-rtc-architecture-docs

- bin-api-manager: Replace VoIP Stack Overview diagram with full topology"
```

---

### Task 2: Update Kamailio Section — Dispatcher and Decoupling

**File:**
- Modify: `bin-api-manager/docsdev/source/architecture_rtc.rst` (Kamailio section, ~lines 65–120)

**Step 1: Locate the Kamailio section**

```bash
grep -n "Kamailio - SIP Edge Router\|Stateless Operation\|Key Features\|Stateless Benefits" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 2: Replace the "Stateless Operation" diagram**

The current diagram only shows Kamailio-1 and Kamailio-2 in a simplified way. Replace it with one that shows the dispatcher module and the single-slot-to-Asterisk-LB design:

```rst
.. code::

    Dispatcher Module — Kamailio Load Balancing:

    External LB
         |
         | Distributes SIP traffic
         v
    +-----------+   +-----------+   +-----------+
    | Kamailio 1|   | Kamailio 2|   | Kamailio 3|  ...
    +-----------+   +-----------+   +-----------+
         |               |               |
         +---------------+---------------+
                         |
         Dispatcher module: single slot
                         |
                         v
                +------------------+
                | Internal         |
                | Asterisk LB      |
                +------------------+
                         |
                         v
                +------------------+
                | Asterisk Farm    |
                +------------------+

    Note: Different Kamailio instances handle different messages
          in the same dialog (stateless operation).
          No Kamailio config change needed when Asterisk pods scale.
```

**Step 3: Replace the "Key Features" bullets** with updated content that includes dispatcher:

```rst
* **Dispatcher Module**: Uses Kamailio's built-in dispatcher for load balancing. Routes new calls to the Internal Asterisk LB via a single slot — no per-Asterisk entries needed
* **Load Balancing**: Distributes incoming SIP traffic across multiple instances
* **Stateless Operation**: No state maintained, enabling dynamic scaling and failover
* **High Availability**: Instances can be added or removed without affecting ongoing calls
* **Fast Performance**: C-based implementation with minimal overhead
```

**Step 4: Replace "Stateless Benefits" section** with a combined "Decoupling and Scaling" section:

```rst
Decoupling and Independent Scaling
++++++++++++++++++++++++++++++++++++

The combination of the dispatcher module and the single-slot Asterisk routing creates
a **bidirectionally decoupled architecture**:

.. code::

    Kamailio → Asterisk:
    +---------------------+         +------------------+
    | Kamailio Farm       |         | Internal         |
    | (dispatcher module) +-------->| Asterisk LB      |
    |                     |  one    +--------+---------+
    | No per-Asterisk     |  slot            |
    | entries needed      |                  v
    +---------------------+         +------------------+
                                    | Asterisk Farm    |
                                    | (scale freely)   |
                                    +------------------+

    Asterisk → Kamailio:
    +------------------+         +------------------+
    | Asterisk Farm    +-------->| Internal         |
    |                  |  sends  | Kamailio LB      |
    | No per-Kamailio  |  here   +--------+---------+
    | entries needed   |                  |
    +------------------+                  v
                                 +------------------+
                                 | Kamailio Farm    |
                                 | (scale freely)   |
                                 +------------------+

**Result:**

* Add or remove Asterisk pods → no Kamailio dispatcher config change required
* Add or remove Kamailio instances → no Asterisk config change required
* Both farms can be scaled independently at any time without coordination
```

**Step 5: Verify**

```bash
grep -n "Decoupling\|dispatcher\|single slot\|scale freely\|bidirectional" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 6: Commit**

```bash
git add bin-api-manager/docsdev/source/architecture_rtc.rst
git commit -m "NOJIRA-update-rtc-architecture-docs

- bin-api-manager: Update Kamailio section with dispatcher module and decoupling explanation"
```

---

### Task 3: Add RE-INVITE / Within-Dialog Routing Section

**File:**
- Modify: `bin-api-manager/docsdev/source/architecture_rtc.rst`
- Insert a new subsection after the Kamailio section, before the Asterisk section

**Step 1: Find insertion point**

```bash
grep -n "Asterisk - Media and Call Processing" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 2: Insert new section before the Asterisk section**

Insert this complete block immediately before `Asterisk - Media and Call Processing`:

```rst
RE-INVITE and Within-Dialog Routing
-------------------------------------

For requests within an established SIP dialog (such as RE-INVITE for media renegotiation
or hold/resume), VoIPBIN uses a **direct routing** strategy that bypasses the Internal
Asterisk LB entirely.

**Why direct routing matters:**

When an INVITE creates a new dialog, Asterisk records its own address in the SIP
``Record-Route`` header returned to the client. All subsequent in-dialog requests
(RE-INVITE, BYE, etc.) carry a ``Route`` header derived from this value, pointing
directly to the specific Asterisk instance that owns the dialog.

Kamailio reads this ``Route`` header and forwards the request straight to that
Asterisk instance — skipping the Internal Asterisk LB completely.

.. code::

    New INVITE (dialog setup):

    Client → External LB → Kamailio(any)
           → Internal Asterisk LB → Asterisk(X)
                                        |
                              Records own address
                              in Record-Route header

    RE-INVITE (within dialog):

    Client → External LB → Kamailio(any)
           → Asterisk(X) directly   <-- Route header bypasses LB
             (correct instance, guaranteed)

**Benefits:**

* **Dialog integrity**: RE-INVITE always reaches the Asterisk that owns the session
* **No stateful proxy needed**: Kamailio remains fully stateless — it just reads the Route header
* **No coordination**: Neither the LB nor Kamailio needs to track which Asterisk owns which dialog
* **Independent scaling**: Adding or removing Asterisk pods during live calls does not affect existing dialogs

.. note:: **AI Implementation Hint**

   When integrating SIP endpoints, ensure your SIP stack correctly handles the
   ``Record-Route`` and ``Route`` headers returned by VoIPBIN. Most standard SIP
   libraries (PJSIP, Sofia-SIP, etc.) handle this automatically. Do not strip or
   modify these headers, as doing so will cause RE-INVITEs to be routed to the
   wrong Asterisk instance.

```

**Step 3: Verify insertion**

```bash
grep -n "RE-INVITE and Within-Dialog\|Record-Route\|Route header\|dialog integrity" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 4: Commit**

```bash
git add bin-api-manager/docsdev/source/architecture_rtc.rst
git commit -m "NOJIRA-update-rtc-architecture-docs

- bin-api-manager: Add RE-INVITE and within-dialog routing section"
```

---

### Task 4: Add Kamailio Proxy Section

**File:**
- Modify: `bin-api-manager/docsdev/source/architecture_rtc.rst`
- Append new section at the end of the file (after SIP Session Recovery)

**Step 1: Find the end of the file**

```bash
wc -l bin-api-manager/docsdev/source/architecture_rtc.rst
tail -10 bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 2: Append the Kamailio Proxy section**

Add this block at the end of the file:

```rst
Kamailio Proxy - Provider Health Monitor
-----------------------------------------

The **Kamailio Proxy** is a lightweight Go sidecar service that runs alongside each
Kamailio instance. It is **not** in the SIP signaling path — no call traffic passes
through it. Its sole responsibility is provider health monitoring.

.. code::

    Position in Architecture:

    +------------------+     +--------------------+
    |  Kamailio        |     |  Kamailio Proxy    |
    |  (SIP signaling) |     |  (management only) |
    |                  |     |                    |
    |  Handles INVITE, |     |  o SIP OPTIONS     |
    |  RE-INVITE, etc. |     |    probes to PSTN  |
    |                  |     |    providers       |
    +------------------+     +--------------------+
                                       |
                                       | Health status
                                       v
                              +--------------------+
                              | bin-route-manager  |
                              | (skips unhealthy   |
                              |  providers when    |
                              |  routing calls)    |
                              +--------------------+

**How it works:**

1. ``bin-route-manager`` periodically requests a health check for each configured provider
2. Kamailio Proxy sends a SIP ``OPTIONS`` request to the provider
3. The provider responds (or times out)
4. Kamailio Proxy reports the health result back to ``bin-route-manager``
5. Route manager marks the provider healthy or unhealthy accordingly
6. Outbound calls avoid unhealthy providers until they recover

**Key characteristics:**

* **Sidecar deployment**: One Kamailio Proxy per Kamailio instance
* **No SIP traffic**: Does not proxy or route any call signaling
* **Passive health checks**: Only sends SIP OPTIONS probes on request
* **Tight coupling with route-manager**: Designed specifically for ``bin-route-manager`` integration
```

**Step 3: Verify**

```bash
grep -n "Kamailio Proxy\|SIP OPTIONS\|Provider Health\|sidecar" \
  bin-api-manager/docsdev/source/architecture_rtc.rst
```

**Step 4: Commit**

```bash
git add bin-api-manager/docsdev/source/architecture_rtc.rst
git commit -m "NOJIRA-update-rtc-architecture-docs

- bin-api-manager: Add Kamailio Proxy section"
```

---

### Task 5: Rebuild HTML and Commit

**Step 1: Check Sphinx virtual environment exists**

```bash
ls /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs/bin-api-manager/docsdev/.venv_docs 2>/dev/null \
  || echo "venv not found"
```

If not found, create it:

```bash
cd bin-api-manager/docsdev
python3 -m venv .venv_docs
source .venv_docs/bin/activate
pip install sphinx sphinx-rtd-theme sphinx-wagtail-theme sphinxcontrib-youtube
```

**Step 2: Clean rebuild**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs/bin-api-manager/docsdev
source .venv_docs/bin/activate 2>/dev/null || true
rm -rf build
python3 -m sphinx -M html source build 2>&1 | tail -20
```

Expected output ends with: `build succeeded` (warnings are OK, errors are not).

**Step 3: Verify the rebuilt page exists**

```bash
ls build/html/architecture_rtc.html
```

**Step 4: Stage and commit both RST and HTML**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs
git add bin-api-manager/docsdev/source/architecture_rtc.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-update-rtc-architecture-docs

- bin-api-manager: Rebuild HTML docs after architecture_rtc.rst updates"
```

---

### Task 6: Final Verification

**Step 1: Check all required content is present**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-update-rtc-architecture-docs

echo "=== Diagram topology ===" && \
grep -c "External.*Load Balancer\|Internal Asterisk LB\|Internal Kamailio LB" \
  bin-api-manager/docsdev/source/architecture_rtc.rst

echo "=== Dispatcher module ===" && \
grep -c "dispatcher" bin-api-manager/docsdev/source/architecture_rtc.rst

echo "=== Bidirectional decoupling ===" && \
grep -c "[Dd]ecoupl" bin-api-manager/docsdev/source/architecture_rtc.rst

echo "=== RE-INVITE section ===" && \
grep -c "RE-INVITE and Within-Dialog" bin-api-manager/docsdev/source/architecture_rtc.rst

echo "=== Kamailio Proxy section ===" && \
grep -c "Kamailio Proxy" bin-api-manager/docsdev/source/architecture_rtc.rst

echo "=== HTML built ===" && \
ls bin-api-manager/docsdev/build/html/architecture_rtc.html
```

Expected: all counts ≥ 1, HTML file exists.

**Step 2: Check no Sphinx errors in the built output**

```bash
grep -i "error\|WARNING.*architecture_rtc" \
  bin-api-manager/docsdev/build/html/architecture_rtc.html 2>/dev/null | head -5
```

Expected: no critical errors.

**Step 3: Review git log**

```bash
git log --oneline -6
```

Expected: 5 commits on top of base (Tasks 1–5).

---

## Completion Checklist

- [ ] VoIP Stack diagram shows full topology (External LB, Kamailio Farm, Internal Asterisk LB, Asterisk Farm, Internal Kamailio LB)
- [ ] Dispatcher module explained in Kamailio section
- [ ] Single-slot-to-Asterisk-LB design documented
- [ ] Bidirectional decoupling section with ASCII diagram present
- [ ] RE-INVITE / within-dialog routing section added with Route header explanation
- [ ] AI Implementation Hint in RE-INVITE section
- [ ] Kamailio Proxy section added (no API endpoints)
- [ ] HTML rebuilt cleanly (no Sphinx errors)
- [ ] Both RST source and `build/` committed together
