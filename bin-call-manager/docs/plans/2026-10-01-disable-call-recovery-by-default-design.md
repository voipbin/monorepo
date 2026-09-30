# VOIP-1553: Disable call recovery unless explicitly enabled

Status: design approved (design review rounds 1-2 APPROVED consecutively); implemented in this PR.
Ticket: VOIP-1553. Analysis (closed, R9/R10 and addendum R4/R5 approved): `~/agent-hermes/notes/tracks/VOIP-1553-analysis.md`.
Blocks: VOIP-1555 (Homer repair). Follow-up: VOIP-1556 (recovery redesign).

## 1. Problem

Call recovery (`callHandler.RecoveryStart`) has never succeeded in production because every Homer lookup returns no rows (VOIP-1555). The recovery path beyond the Homer lookup has never run and has verified defects (analysis F3, F7, F8, F9): it selects already-ended calls (no `tm_end` filter, up to 10000 channels per event), re-INVITEs them, re-runs each call's last stored action and overwrites `channel_id`/`bridge_id` before the new leg is answered, and the recovered channel is stored as `TypeNone` so its hangup is not handled as a call hangup.

Today this is harmless only because Homer is broken. Repairing Homer (VOIP-1555) without first disabling recovery would turn the next call-Asterisk container recreation into up to 10000 unwanted re-dials with side effects. Recovery must be off before Homer is repaired.

## 2. Decisions locked (CEO, 2026-10-01)

- D1. Option A from the analysis: recovery is disabled, redesign later (VOIP-1556).
- D2. Implemented as a setting, `recovery_enabled` / env `RECOVERY_ENABLED`. Absent or empty means disabled. Only an explicit true value enables it. Chosen so recovery can be re-enabled after the redesign without a code change.
- D3. Public developer docs (`bin-api-manager/docsdev/...` "SIP Session Recovery") are out of scope for this PR. The CEO is handling them directly.

## 3. Entry points (verified, main 57573bff9)

`RecoveryStart` is the only function that starts recovery (it calls `recoveryRun` per channel). Its callers:

- `pkg/callhandler/event.go:121` `EventSMContainerDied` (automatic, sentinel-manager `container_died`).
- `pkg/listenhandler/v1_recovery.go:32` `processV1RecoveryPost` (manual `POST /v1/recovery`).

`cmd/call-control` builds a callHandler with a recoveryHandler but never calls `RecoveryStart`. `CallV1RecoveryStart` in bin-common-handler has no caller in the monorepo. A guard at the top of `RecoveryStart` therefore blocks every path.

## 4. Design

### 4.1 Config

`internal/config/main.go`:

- Add `RecoveryEnabled bool` to `Config` with a comment stating that false (the default) disables both automatic and manual recovery.
- Register `f.Bool("recovery_enabled", false, "Enable call recovery (automatic on Asterisk container death and manual /v1/recovery). Disabled by default, see VOIP-1553")` and bind env `RECOVERY_ENABLED`.
- Load with `viper.GetBool("recovery_enabled")`. viper uses `cast.ToBool`, which accepts `1/t/T/TRUE/true/True` as true; empty, absent, `false`, and any unparsable value (for example `yes`) become false. Fail-closed by construction.

### 4.2 callHandler

- Add field `recoveryEnabled bool` to `callHandler` and a trailing parameter `recoveryEnabled bool` to `NewCallHandler`.
- Callers (full grep, 2): `cmd/call-manager/main.go:152` passes `cfg.RecoveryEnabled`; `cmd/call-control/main.go:84` passes `config.Get().RecoveryEnabled` (never used there, passed for consistency so the struct is never built with an implicit value).
- Chosen over reading `config.Get()` inside `RecoveryStart` so tests set the flag on the struct directly and the handler has no hidden global dependency.
- Startup log: `cmd/call-manager/main.go` `run` logs once at start, branching on the same `cfg.RecoveryEnabled` value it passes to `NewCallHandler`, Info, `Call recovery is disabled. RECOVERY_ENABLED is not set to true.` when disabled, or Warn `Call recovery is enabled.` when enabled. Every replica emits it on start, which makes the rollout gate (§6) verifiable per instance without triggering recovery. `cmd/call-control` (a CLI) does not log it.

### 4.3 Guard

At the top of `RecoveryStart`, before any channel lookup:

```go
if !h.recoveryEnabled {
    log.Infof("Call recovery is disabled. Skipping the recovery. asterisk_id: %s", asteriskID)
    return nil
}
```

- Info level, one line per call. The message prefix `Call recovery is disabled` is fixed (the gate's Loki query in §6 uses it; code review pins it).
- Returns nil: the automatic path has nothing to retry, and returning an error would only produce an Error log in `EventSMContainerDied` for an intended state.

### 4.4 Manual endpoint response when disabled

`POST /v1/recovery` keeps returning 200 when disabled. Reasons: it has no caller in the monorepo (`CallV1RecoveryStart` is unused, not routed by api-manager); the existing response already means "accepted", since recovery runs per channel in goroutines and the handler never reported per-channel outcome; the Info log records the skip. Adding a distinct status would need a new error type across call-manager and common-handler for an endpoint nobody calls. The redesign (VOIP-1556, F10) owns the manual endpoint's semantics.

### 4.5 Deployment config

- `bin-call-manager/komodo/docker-compose.yml`: no `RECOVERY_ENABLED` line is added. Absent means disabled, which is the target state. Adding `RECOVERY_ENABLED=false` would be equivalent; not adding it keeps the compose file free of a Komodo variable that must then exist.
- `bin-call-manager/k8s/deployment.yml`: unchanged, same reason (k8s is not the live runtime).

### 4.6 Documentation (internal only, D3)

- `bin-call-manager/docs/operations.md`: add a config row for `recovery_enabled` / `RECOVERY_ENABLED` (default `false`); change the line 13 troubleshooting row so it no longer points to `/v1/recovery` for orphaned calls (disabled; use `call-control call update-status`), and drop the inaccurate "pod restarted" premise.
- `bin-call-manager/docs/architecture.md:111`: mark `/v1/recovery` as disabled unless `RECOVERY_ENABLED=true`.
- `bin-call-manager/docs/domain.md:97` item 10: state that recovery is disabled by default (VOIP-1553) and must not be enabled before VOIP-1556.
- `bin-sentinel-manager` docs: README line 5, `docs/architecture.md:153`, `docs/domain.md:88`, `docs/operations.md:28` troubleshooting row, `CLAUDE.md:35`: add that call-manager's recovery is disabled by default, so no recovery after a container death is the expected state. Metric/flap-damping rationale text (`operations.md:171-172`, `architecture.md:126`, `CLAUDE.md:61`) stays: it describes what the events would drive and stays correct once recovery is re-enabled; a one-line note at the top of the operations metrics section covers the disabled state.
- `monitoring/grafana/dashboards/sentinel-manager.json`: row title "Recovery Health (leading indicators)" gets a suffix noting recovery is disabled in call-manager (VOIP-1553). Panel queries unchanged. Panel descriptions (`:262`, `:290`) stay as they are: they describe what the counters mean for recovery once re-enabled, and the row title carries the disabled state.
- `bin-call-manager/CLAUDE.md:58-59` (dependency tree of `callHandler(...)` constructor args): add `outboundConfigHandler` (missing today) and `recoveryEnabled` to the `callHandler(...)` argument list only, no new tree node, so the list matches `NewCallHandler`.
- `bin-call-manager/docs/operations.md:109` (`homer_whitelist`, "Homer recovery endpoint"): unchanged, still accurate (the whitelist is only used by recovery); the new `recovery_enabled` row sits next to it.

## 5. Tests

- `pkg/callhandler/recovery_test.go`: new `Test_RecoveryStart_disabled`: `recoveryEnabled=false`, strict gomock with no expectations on utilHandler/channelHandler; returns nil. New `Test_RecoveryStart_enabled`: `recoveryEnabled=true`, expects the channel lookup (returns empty list), returns nil.
- `pkg/callhandler/event_test.go`: existing `Test_EventSMContainerDied` and `_recoveryError` set `recoveryEnabled: true` (they test the enabled path). Add a case: `asterisk-call` event with a valid id and `recoveryEnabled=false` makes no channel lookup and returns nil.
- `internal/config/main_test.go`: add `recovery_enabled` to the flag registration table and assert its default value is `false` (`flag.DefValue == "false"`). `LoadGlobalConfig` runs under `sync.Once` and is not re-testable per case; the parsing of env values is viper's `cast.ToBool` and is not re-tested here.
- No test calls `NewCallHandler` (tests build `&callHandler{}` directly), so only the two `cmd/` call sites change.
- `TestConfigStruct`: add `RecoveryEnabled: true` and its assertion, consistent with the other fields.
- Verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in bin-call-manager.

## 6. Rollout and VOIP-1555 unblock gate

After merge and CircleCI release approval (CEO), the gate for starting Homer repair is:

1. Every call-manager instance on bm-nyc-01 (Komodo Stack `bin-call-manager`, `deploy.replicas: 2`) runs the new image (Komodo `ListStackServices` image tag equals the merge commit), and no other container running a `voipbin/bin-call-manager` image exists on the host (Komodo `ListDockerContainers`; the legacy install-stack `voipbin-call-mgr` shares the subscribe queue and would process `container_died` without the guard). Checked 2026-10-01: only `bin-call-manager-call-manager-1` and `-2` run.
2. The deployed environment has no `RECOVERY_ENABLED` set to a true value (Komodo `InspectDockerContainer` env).
3. Loki query `{service_name=~"bin-call-manager-call-manager-.+"} |= "Call recovery is disabled. RECOVERY_ENABLED is not set to true."` from the rollout start returns lines from 2 distinct `service_name` values (the replica count), and `|= "Call recovery is enabled"` returns none. The full startup message is matched, so the per-call guard log (same prefix) cannot satisfy it. No manual trigger and no container death needed.

Merge alone is not the gate.

## 7. Out of scope

- Public developer docs (D3).
- Any recovery fix (F3, F4, F5, F7-F10): VOIP-1556.
- Homer repair: VOIP-1555. Homer token logging: VOIP-1554.
- sandbox and install compose comments (other repos).

## 8. Risks

- Someone sets `RECOVERY_ENABLED=true` before VOIP-1556: the config comment, the operations.md row, and the domain.md note all state it must not be enabled before VOIP-1556.
- Loss of recovery value: none in practice, recovery has never succeeded (0 successes in the whole Loki window).

## Review history

- Round 1: APPROVED with MINOR items, all addressed: gate step 3 not actionable (replaced with a per-replica startup log), `CLAUDE.md:58-59` inventory, `operations.md:109` stance, dashboard panel descriptions stance; NITs: no test call sites, `TestConfigStruct`, pinned log prefix.
- Round 2: APPROVED with MINOR items, addressed: gate step 1 also excludes a non-Komodo call-manager container; gate step 3 has an exact Loki selector, full-message match and expected count 2. NITs: startup log and guard use the same value; CLAUDE.md edit is argument-list only.
- PR review round 1: APPROVED. MINOR: env binding and constructor assignment were not covered by tests; added `TestBootstrap_recoveryEnabledEnv` and `Test_NewCallHandler_recoveryEnabled`. NIT: flag help text aligned with §4.1, status line updated. Mutation: removing the constructor assignment fails `Test_NewCallHandler_recoveryEnabled`; removing the explicit `BindEnv` line does not fail the env test because `viper.AutomaticEnv()` (already enabled in `bindConfig`) resolves `recovery_enabled` from `RECOVERY_ENABLED` on its own, so the env-to-value behaviour the test pins holds either way.
- PR review round 2: APPROVED. NIT: operations.md true-value list made exact. NIT (event.go Debug line before the disabled Info) and MINOR (`LoadGlobalConfig` copy untestable under `sync.Once`) accepted as is.
