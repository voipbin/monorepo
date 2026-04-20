# Design: RTC Architecture Documentation Update

**Date:** 2026-04-21
**Branch:** NOJIRA-update-rtc-architecture-docs
**File to update:** `bin-api-manager/docsdev/source/architecture_rtc.rst`

---

## Problem Statement

The current `architecture_rtc.rst` has two issues:

1. **Inaccurate topology diagram** — The existing ASCII diagram does not reflect the actual
   production architecture. It omits the External Load Balancer, the internal Kamailio LB,
   and the internal Asterisk LB. This misleads readers and developers.

2. **Missing components and concepts** — The following are entirely undocumented:
   - RE-INVITE / within-dialog routing (bypasses Asterisk LB via Route headers)
   - Bidirectional decoupling between Kamailio and Asterisk
   - Kamailio Proxy (health check management sidecar, added in PR #787)

---

## Actual Architecture (from diagram review)

### Full SIP Signaling Topology

```
SIP/WebRTC Client
    |
    | INVITE / RE-INVITE / 200 OK
    v
External Load Balancer (Signal Gateway)   ← internet-facing edge
    |                                        distributes to Kamailio Farm
    | INVITE
    v
Kamailio Farm (Kamailio 1 / 2 / 3 ...)
  - Uses Kamailio dispatcher module for inter-Kamailio load balancing
  - Has ONE dispatcher slot → Internal Asterisk LB (not individual Asterisks)
    |
    | INVITE (new dialog)
    v
Internal Asterisk LB                      ← k8s Service or hardware LB
    |                                        Asterisk pods can scale freely
    v
Asterisk (Call) PBX Farm

    |
    | 200 OK / 200 OK(RE-INVITE)
    v
Internal Kamailio LB                      ← Asterisk sends responses here
    |                                        Kamailio instances can scale freely
    v
Kamailio (any instance, e.g. Kamailio 2)
    |
    v
External Load Balancer → Client
```

### RE-INVITE (Within-Dialog) Routing

```
Client → External LB → Kamailio (any) → Asterisk(X) directly
```

- The initial INVITE sets up the dialog; Asterisk(X) records its address in SIP Route headers.
- Subsequent in-dialog requests (RE-INVITE) carry these Route headers.
- Kamailio reads the Route header and forwards directly to Asterisk(X), **bypassing the
  Internal Asterisk LB**.
- This guarantees the RE-INVITE reaches the same Asterisk that owns the dialog.

---

## Key Design Principles

### 1. Bidirectional Decoupling

| Direction | Mechanism | Effect |
|---|---|---|
| Kamailio → Asterisk | Single dispatcher slot → Asterisk LB | Add/remove Asterisk pods: **no Kamailio config change** |
| Asterisk → Kamailio | Asterisk sends to Internal Kamailio LB | Add/remove Kamailio instances: **no Asterisk config change** |

Neither component needs to know about the other's individual instances.
Both can be scaled independently at any time.

### 2. Dispatcher Module for Kamailio Distribution

Kamailio uses its built-in **dispatcher module** for:
- Distributing incoming SIP traffic across Kamailio Farm instances (via External LB or
  internal routing)
- Routing 200 OK responses from the Internal Kamailio LB to any Kamailio instance

The dispatcher is **not** used for routing to Asterisk — only the single LB slot is used.

### 3. Route-Header-Based Dialog Routing

SIP Route headers inserted by Asterisk during dialog setup ensure that all subsequent
in-dialog requests are delivered to the correct Asterisk instance without any stateful
coordination in Kamailio.

---

## Kamailio Proxy (New Component — PR #787)

**Role:** Management sidecar alongside Kamailio. Does **not** participate in SIP signaling.

**Responsibilities:**
- Monitors provider health by sending SIP OPTIONS probes
- Receives health check requests from `bin-route-manager`
- Updates provider health status based on probe results
- Allows route-manager to skip unhealthy providers when routing outbound calls

**Architecture position:** Sits alongside each Kamailio instance; not in the SIP traffic path.

---

## Changes to `architecture_rtc.rst`

### Section 1: VoIP Stack Overview — Replace diagram

Replace the current simplified ASCII diagram with one that shows the full topology:
External LB → Kamailio Farm → Internal Asterisk LB → Asterisk Farm (for new calls), and
Asterisk Farm → Internal Kamailio LB → Kamailio Farm (for responses).

### Section 2: Kamailio section — Add dispatcher + decoupling explanation

Extend the existing Kamailio section to explain:
- Dispatcher module usage for Kamailio-side load balancing
- Single slot to Asterisk LB (no per-Asterisk entries)
- Why this enables independent scaling of Asterisk

### Section 3: New — RE-INVITE and Within-Dialog Routing

New subsection after the Kamailio section explaining:
- Route headers in SIP and their role
- How in-dialog requests bypass the Asterisk LB
- Why this guarantees dialog integrity without stateful Kamailio

### Section 4: New — Kamailio Proxy

New section at the end explaining:
- What the Kamailio Proxy is (management sidecar, not in SIP path)
- Provider health monitoring role
- Integration with route-manager

---

## Files to Change

| File | Change Type |
|---|---|
| `bin-api-manager/docsdev/source/architecture_rtc.rst` | Update existing content + add new sections |
| `bin-api-manager/docsdev/build/` | Rebuild HTML after RST changes |

No new RST files are needed. All changes go into the single existing file.

---

## Success Criteria

- [ ] VoIP Stack diagram accurately reflects External LB, Kamailio Farm, Internal Kamailio LB, Internal Asterisk LB, and Asterisk Farm
- [ ] Dispatcher module usage is explained clearly
- [ ] Bidirectional decoupling (Kamailio ↔ Asterisk) is documented
- [ ] RE-INVITE / within-dialog routing section exists with Route header explanation
- [ ] Kamailio Proxy section exists (no API endpoints)
- [ ] HTML rebuilt and committed alongside RST changes
