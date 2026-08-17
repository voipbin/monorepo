# Design: roll out Komodo-managed deploy to 7 Tier 2 bin-*-manager services (VOIP-1348)

## Goal

Extend the Komodo-managed deploy pattern proven by VOIP-1342
(bin-call-manager, PR #1188) and applied to 16 services by VOIP-1347
(Tier 1, PR #1190) to 7 more services:

ai, customer, number, tts, email, message, registrar
(`bin-<short>-manager` for each).

This is the **third application of an already-verified pattern** — see
[2026-08-16-komodo-call-manager-cutover-design.md](2026-08-16-komodo-call-manager-cutover-design.md)
for the original spike and
[2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
for the Tier 1 rollout. This doc only covers what is new for Tier 2.

## Excluded: bin-hook-manager

`bin-hook-manager` is explicitly **out of scope** for this PR. It needs
`SSL_CERT_BASE64`/`SSL_PRIVKEY_BASE64` Komodo Variables that do not exist
yet — confirmed absent from the known `[[BIN_MANAGER__*]]` set, and the
live GKE/install container uses real non-empty certificate values
(~7600/2272 chars) that cannot be reconstructed or guessed. Tracked
separately as [VOIP-1353](https://voipbin.atlassian.net/browse/VOIP-1353)
(infra-secret work, different repo). No `bin-hook-manager/` files are
touched by this PR.

## What's new for Tier 2: provider-secret env vars

Unlike Tier 1 (whose 16 services shared one identical, secret-free env
block), all 7 Tier 2 services carry one or more extra
provider/integration secrets on top of the common
`DATABASE_DSN`/`RABBITMQ_ADDRESS`/`REDIS_ADDRESS`/`REDIS_DATABASE`/
`PROMETHEUS_*` block. Every one of them was already confirmed present in
Komodo's `[[BIN_MANAGER__*]]` Variables store (synced from
`infra-secret/secrets-source/bin-manager`) — nothing new needed to be
provisioned:

| Service | Extra env vars | Komodo Variable(s) |
|---|---|---|
| bin-ai-manager | `ENGINE_KEY_CHATGPT` | `[[BIN_MANAGER__ENGINE_KEY_CHATGPT]]` |
| bin-customer-manager | `EMAIL_VERIFY_BASE_URL` | plain literal (not a secret — see below) |
| bin-number-manager | `TWILIO_SID`, `TWILIO_TOKEN`, `TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID`, `TELNYX_TOKEN` | `[[BIN_MANAGER__TWILIO_SID]]`, `[[BIN_MANAGER__TWILIO_TOKEN]]`, `[[BIN_MANAGER__TELNYX_CONNECTION_ID]]`, `[[BIN_MANAGER__TELNYX_PROFILE_ID]]`, `[[BIN_MANAGER__TELNYX_TOKEN]]` |
| bin-tts-manager | `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`, `ELEVENLABS_API_KEY` | `[[BIN_MANAGER__AWS_ACCESS_KEY]]`, `[[BIN_MANAGER__AWS_SECRET_KEY]]`, `[[BIN_MANAGER__ELEVENLABS_API_KEY]]` |
| bin-email-manager | `SENDGRID_API_KEY`, `MAILGUN_API_KEY` | `[[BIN_MANAGER__SENDGRID_API_KEY]]`, `[[BIN_MANAGER__MAILGUN_API_KEY]]` |
| bin-message-manager | `AUTHTOKEN_MESSAGEBIRD`, `AUTHTOKEN_TELNYX` | `[[BIN_MANAGER__AUTHTOKEN_MESSAGEBIRD]]`, `[[BIN_MANAGER__AUTHTOKEN_TELNYX]]` |
| bin-registrar-manager | `DATABASE_DSN_ASTERISK` (in addition to the renamed `DATABASE_DSN_BIN`), `DOMAIN_NAME_EXTENSION`, `DOMAIN_NAME_TRUNK` | `[[BIN_MANAGER__DATABASE_DSN_ASTERISK]]`, `[[BIN_MANAGER__DOMAIN_NAME_EXTENSION]]`, `[[BIN_MANAGER__DOMAIN_NAME_TRUNK]]` |

All values were read directly from each service's own block in
`voipbin/install/docker-compose.yml.dist` (not a secondhand summary), then
mapped by key name to the confirmed-existing Komodo Variable set.

### bin-customer-manager's `EMAIL_VERIFY_BASE_URL` is a plain literal, not a secret

`EMAIL_VERIFY_BASE_URL` is not in the known `[[BIN_MANAGER__*]]` set. It
was checked and found to be a public-facing base URL used to build
email-verification links, not a credential — and its default is already
hardcoded in the Go binary itself
(`bin-customer-manager/internal/config/main.go`, the
`email_verify_base_url` flag defaults to `"https://api.voipbin.net"`,
VoIPbin's real production API host). The Komodo compose file sets it as a
plain literal (`EMAIL_VERIFY_BASE_URL=https://api.voipbin.net`), matching
that code default — not a `[[BIN_MANAGER__...]]` reference, and not a
guess.

### bin-registrar-manager: dual database DSN

`bin-registrar-manager` is the only Tier 2 service with two database
connections: `DATABASE_DSN_BIN` (the shared `bin_manager` schema) and
`DATABASE_DSN_ASTERISK` (the Asterisk realtime schema, for SIP
registration state). Both map to already-existing, separate Komodo
Variables (`[[BIN_MANAGER__DATABASE_DSN_BIN]]` and
`[[BIN_MANAGER__DATABASE_DSN_ASTERISK]]`) — no new provisioning needed.

## What's identical to Tier 1 / bin-call-manager

- **Compose file shape:** `<service>/komodo/docker-compose.yml`, one
  service block, `image: voipbin/bin-<short>-manager:__IMAGE_TAG__`,
  `json-file` logging (10m/3 files), `container_name:
  voipbin-<short>-manager`, `restart: always`, `command:
  ./<short>-manager`, attached to the shared `production` network
  (`external: true`), no `install_default` detour.
- **No healthcheck:** all 7 services build on
  `gcr.io/distroless/static-debian12` (confirmed per-service Dockerfile
  `FROM` line) — no shell/wget available for a Docker healthcheck.
- **Binary name convention:** `./<short>-manager` confirmed per service
  via each `Dockerfile` and `cmd/` directory listing (each `cmd/` also
  has a `<short>-control` CLI binary, not used here). No exceptions.
- **CI mechanics reused as-is, unmodified:**
  `.circleci/scripts/render-image-tag.sh` and
  `.circleci/scripts/komodo-api-deploy.sh` are already generic. The 7 new
  `<service>-deploy` CI jobs call them exactly as `bin-call-manager-deploy`
  and the Tier 1 jobs do, with the same
  `CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3` budget.
- **These 7 services had no deploy job at all before this PR** (only
  `-test`/`-build`) — same as Tier 1. This PR adds new `-deploy` jobs; it
  does not replace or conflict with anything currently running.
- **Komodo Variables (`[[BIN_MANAGER__*]]`):** resolved by Komodo itself
  from its global Variables store, already synced from infra-secret.
  Nothing in this PR or CI supplies these values.

## Background-loop audit

Grepped all 7 services' source trees for `time.NewTicker`,
`time.Tick(`, and `cron\.` (the same check that caught
`bin-route-manager`'s carrier health-check ticker in Tier 1). Only
`bin-tts-manager` has hits, all in `pkg/streaminghandler/` (websocket
write-pacing and provider keepalive tickers for GCP and ElevenLabs
streaming sessions). These are scoped to individual per-connection
streaming sessions, not a global background scheduler — an old/new
container overlap does not compound their cost the way route-manager's
provider-wide health-check loop did (that loop runs continuously and
probes every configured carrier regardless of active traffic). No
service in this batch needs a shortened cutover overlap window.

## Cutover procedure

Identical to Tier 1's standard flow — see
[2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md#cutover-procedure-standard-15-of-16-services)
for the full step-by-step. Summary, per service, after this PR merges and
its `-deploy` job runs for the first time:

1. Deploy via CI — the new container joins as an **additional** RabbitMQ
   competing consumer alongside the old one (zero-gap-safe, confirmed by
   the VOIP-1342 pilot).
2. Verify at the **application log level**, not just Docker
   running-state — `docker logs` the new container and confirm a
   connection-success line for RabbitMQ/DB/Redis and no
   `CRITICAL`/`Error`-level lines from the boot path (VOIP-1342's
   false-positive lesson: "running" state alone does not mean the app is
   healthy).
3. Remove the old container once the new one is confirmed healthy at the
   log level. Check the old container's actual name first — some
   services in `voipbin/install/docker-compose.yml.dist` pin
   `container_name: voipbin-<short>-mgr`, others leave it
   Compose-assigned.
4. Comment out the removed service's block in bm-nyc-01's live
   `/opt/voipbin/install/docker-compose.yml` (SSH), preventing the
   install-managed compose project from recreating the old container.
5. Confirm a single RabbitMQ consumer remains for the service's queues.

## Testing

No Go source changes — this PR only touches `komodo/docker-compose.yml`
files (new, 7), `.circleci/config_work.yml` (CI wiring), 7 services'
`docs/operations.md`, and this design doc. The 5-step Go verification
workflow (`go mod tidy && go mod vendor && go generate && go test &&
golangci-lint`) is a no-op here and was not run. What was verified
instead:

- `.circleci/config_work.yml` parses as valid YAML
  (`python3 -c "import yaml; yaml.safe_load(open(...))"`).
- Each of the 7 new `-deploy` jobs is present under `jobs:` and wired
  into its service's workflow block with `requires: [<service>-build]`,
  matching the Tier 1/bin-call-manager pattern.
- `git status` confirms only `komodo/` (7 new compose files), 7
  services' `docs/operations.md`, this design doc, and
  `.circleci/config_work.yml` changed — no `.go` files touched, and
  nothing under `bin-hook-manager/`.

## Explicitly out of scope

- `bin-hook-manager` (see above — VOIP-1353 dependency).
- Running any of the 7 new `-deploy` jobs, or performing any cutover
  step, is **not** part of this PR. This PR ends at an open, reviewable
  PR — the actual deploy/cutover for each service is a separate
  follow-up step after human review, same as VOIP-1342 and VOIP-1347.
