# README Update Design

**Date:** 2026-05-09  
**Branch:** NOJIRA-Update-readme  
**Audience:** External developers / open-source users who want to self-host or integrate VoIPbin

---

## Problem

The root `README.md` has three concrete issues that mislead external developers:

1. **Placeholder clone URL** — `github.com/your-org/voipbin.git` is never valid
2. **Overstated scope** — the intro says "the unified backend codebase that powers all VoIPBin services", but this is one of multiple VoIPbin repositories
3. **Stale "coming soon"** — "platform architecture guide (coming soon)" still in the Getting Started section; there is no public link to provide

---

## Decisions

### 1. Intro reframe

Replace the opening paragraph to scope the repo correctly:

> **VoIPbin Monorepo** — the primary backend services repository for the VoIPbin CPaaS platform.
>
> VoIPbin is a cloud-native, open-source CPaaS platform for programmable voice communication. This repository contains the 34 Go microservices that handle call routing, AI pipelines, conferencing, billing, messaging, and more.
>
> This is one of VoIPbin's main repositories. It covers the backend service layer; other infrastructure (SIP proxy, Kamailio, Kubernetes configs) lives in separate repos.

### 2. Directory table — no changes

The 37-row table (34 `bin-*` + 3 `voip-*` proxies) is accurate. `docs/`, `monitoring/`, and `scripts/` are **not** added — they are internal or not ready for external use.

### 3. Clone URL

Replace placeholder with real URL:

```bash
$ git clone https://github.com/voipbin/monorepo.git
$ cd monorepo
```

### 4. Remove stale "coming soon"

Delete the sentence: _"We recommend reading the platform architecture guide (coming soon) before setup."_  
No replacement link — no public architecture guide exists yet.

---

## Out of scope

- Service description rewrites
- New sections (contributing, prerequisites, CI/CD)
- `monitoring/`, `docs/`, `scripts/` directory entries
- Any other structural changes
