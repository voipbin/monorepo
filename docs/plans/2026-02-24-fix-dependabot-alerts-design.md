# Fix Dependabot Alerts

## Problem

33 open dependabot alerts across 4 unique vulnerabilities:

| Package | Ecosystem | Severity | Current | Patched | Affected |
|---------|-----------|----------|---------|---------|----------|
| `filippo.io/edwards25519` | Go | Low | 1.1.0 | >= 1.1.1 | 30 Go services |
| `protobuf` | pip | High | 5.29.5 | >= 5.29.6 | bin-pipecat-manager |
| `pillow` | pip | High | 11.3.0 | >= 12.1.1 | bin-pipecat-manager |
| `nltk` | pip | Critical | 3.9.2 | None | bin-pipecat-manager |

## Approach

Single branch with two categories of fixes.

### Part A: Go — edwards25519 bump

Run `go get filippo.io/edwards25519@v1.1.1` + `go mod tidy` + `go mod vendor` in all 30 affected services. This is a transitive dependency of `go-sql-driver/mysql` — the bump is safe and mechanical.

Affected services: bin-agent-manager, bin-ai-manager, bin-api-manager, bin-billing-manager, bin-call-manager, bin-campaign-manager, bin-common-handler, bin-conference-manager, bin-contact-manager, bin-conversation-manager, bin-customer-manager, bin-email-manager, bin-flow-manager, bin-hook-manager, bin-message-manager, bin-number-manager, bin-outdial-manager, bin-pipecat-manager, bin-queue-manager, bin-registrar-manager, bin-route-manager, bin-sentinel-manager, bin-storage-manager, bin-tag-manager, bin-talk-manager, bin-transcribe-manager, bin-transfer-manager, bin-tts-manager, bin-webhook-manager, voip-asterisk-proxy.

### Part B: Python — Upgrade pipecat-ai 0.0.100 -> 0.0.103

Update `pyproject.toml` to pin `pipecat-ai>=0.0.103` and regenerate `uv.lock`.

This resolves:
- `protobuf` >= 5.29.6 (pipecat-ai 0.0.103 requires `~=5.29.6`)
- `pillow` >= 12.1.1 (pipecat-ai 0.0.103 allows `<13,>=11.1.0`)

Does NOT resolve:
- `nltk` — no upstream fix available (3.9.2 is latest, vulnerability affects <= 3.9.2)

### Code changes for pipecat upgrade

VAD `stop_secs` default changed from 0.8 to 0.2 in pipecat 0.0.102. To preserve current behavior, explicitly set `stop_secs=0.8` on `SileroVADAnalyzer`.

The `vad_analyzer` parameter on transport params is deprecated (v0.0.101) but still functional and used in official examples. No migration needed now.

## Files Changed

- All 30 Go services: `go.mod`, `go.sum`, `vendor/`
- `bin-pipecat-manager/scripts/pipecat/pyproject.toml` — bump pipecat-ai version
- `bin-pipecat-manager/scripts/pipecat/uv.lock` — regenerated
- `bin-pipecat-manager/scripts/pipecat/run.py` — explicit VAD stop_secs

## Verification

- All 30 Go services: `go mod tidy && go mod vendor && go test ./...`
- Python: `uv lock` succeeds with updated versions
