# VOIP-1602: stop logging credentials in startup config logs

## 1. Ticket review (result: valid)

Checked against origin/main acb0b19d1.

- Confirmed: 15 services log the whole `Config` struct at startup, one line each in `cmd/<name>-manager/main.go`:
  agent:68, customer:66, flow:70, number:98, outdial:64, pipecat:75, queue:71, schedule:99, talk:67 (`logger.WithField("config", cfg)`),
  timeline:73, transcribe:105, transfer:99, tts:96, webchat:64, webhook:102.
- None of the `Config` structs has json tags, so the joonix formatter (a JSON formatter) writes every field under its Go name in clear text.
- Secret fields per struct (read from `internal/config`):
  - 14 services (all except timeline): `RedisPassword`; timeline has no redis fields
  - all 15: `DatabaseDSN`, `RabbitMQAddress` (an amqp connection url, `amqp://user:password@host:port`, which carries credentials; registrar masks the same value as a secret at `bin-registrar-manager/cmd/registrar-manager/main.go:70`)
  - number: `TwilioSID`, `TwilioToken`, `TelnyxToken`
  - transcribe: `AWSAccessKey`, `AWSSecretKey`
  - tts: `AWSAccessKey`, `AWSSecretKey`, `ElevenlabsAPIKey`
  - timeline: `HomerAuthToken`
- Precedent: bin-registrar-manager logs an explicit field list and masks secrets with its own `redacthandler` (service private).
- Not a target: ai, api, billing, conference, conversation log a plain "Starting" line without config.

## 2. Scope of the change

The change covers the 15 `WithField("config", ...)` calls listed above (found by searching `bin-*` for `WithField("config"` and for `cfg` or `config.Get()` passed to `WithField(s)`, `Infof`, `Errorf` and `Sprintf`, excluding vendor, `_test.go` and `mock_`).
The `*-control` commands (33) do not log the config at startup.
Kept as is: `bin-timeline-manager/cmd/timeline-manager/main.go:119` logs `ClickHouseAddress` under the key `dsn`; it is a host address.

## 3. Design

- Per service, move the log call into `logStartup(log *logrus.Entry, cfg *config.Config)` in `cmd/<name>-manager/main.go`. `runDaemon` calls it with the existing `log` entry (talk: `logger`).
- `logStartup` writes an explicit list of the non-secret `Config` fields as `logrus.Fields` (keys are the flag names), message text unchanged. The secret fields above are omitted, not masked: no new helper, no shared code. Non-secret fields (addresses of redis, prometheus, base urls, ids, bucket names) stay.
- Omitted fields are dropped entirely, so the log no longer shows the database host or the broker host. Unlike registrar (which masks with its private `redacthandler`), no value or part of a value is logged. Masking would need a helper (a new shared package is out of scope) and a partial value still shows the user name.
- Kept fields: every `Config` field that is not in the secret list, key = the flag (viper) name that `internal/config` of the same service reads the field from, so there is no ambiguity about acronyms (for example `prometheus_listen_address`, `redis_database`, `telnyx_connection_id`, `schedule_tick_interval_sec`). The test enforces that every such field is present.
- Kept url or host fields: `HomerAPIAddress`, `ClickHouseAddress`, `GCPTTSEndpoint`, `PasswordResetBaseURL`, `EmailVerifyBaseURL`. Their flags and docs describe plain addresses (the Homer credential is the separate `HomerAuthToken`; the two base urls default to a public https url), so they are not credentials.
- Rejected: masking per field (needs a helper, partial values still leak the user name), json tags on the struct (also affects other users of the struct), a shared helper in `bin-common-handler` (admission rule, ask first).
- Rollback: revert the commit; behaviour apart from the log line is unchanged.
- pipecat keeps `LocalLLMModels()` out of the `Config` struct only because the whole struct used to be logged (`internal/config/config.go:107-109`, `docs/operations.md:16`, `local_test.go:48`). The code stays as is; the two comments and the doc cell are reworded so they no longer claim the daemon logs the whole `Config`.
- pipecat `config.Get()` returns a value, so its call passes a pointer to a local copy.
- Verification per service: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` for all 15.

## 4. Tests

One new `cmd/<name>-manager/startup_log_test.go` per service, `Test_logStartup`:
- holds two explicit tables: `secrets` (Go field names that must not be logged) and `logged` (Go field name to expected log key, the flag name from `internal/config`, for example `PrometheusListenAddr` to `prometheus_listen_address` in customer, `RedisDB` to `redis_db` in tts, `AnalysisModelStage1` to `analysis_model_stage1` in timeline),
- asserts by reflection that `secrets` and `logged` together are exactly the set of `Config` fields (no overlap, none missing, no typo), and fails with `t.Fatalf` for a field Kind other than string and int (all 15 structs have only those today),
- fills every string field with a sentinel `SENTINEL_<Field>_END` and every int field with a distinct value (1001, 1002, ...), renders `logStartup` through a real logrus logger with the joonix formatter into a buffer, and parses the one output line as JSON,
- asserts that each `logged` field appears under exactly its expected key with its own value (key and value pair, so a wrong key or a dropped field fails; no substring matching), that no `secrets` sentinel appears anywhere in the raw output, and that the message is present,
- reads `main.go` of the package and asserts that it has a call line matching `(?m)^\s*logStartup\(` (a declaration alone does not match) and no `WithField("config"` call (pattern `WithField\(\s*"config"`), so reverting the call site in `runDaemon` is caught too,
- uses unique helper names inside the test function (timeline's `main_test.go` shares `package main`) and does not call `config.Get()` or touch the viper globals.
A field added later is in neither table, so the test fails until the author classifies it as secret or logged.
Reproduction: a scratch copy where `logStartup` still holds the old `WithField("config", cfg)` statement is run with the same test and must fail for every secret field and for the source check.

## 5. Docs

The post-edit hook `scripts/check-service-docs.sh` may warn that `cmd/*/main.go` changed without `docs/architecture.md`; the change touches one log call and no routing or event list, so the warning is intentionally ignored.
Compatibility: the nested `config` key disappears from the startup entry; log queries that read `config.*` (outside the repository) are not checked.
Limit of the source check: it cannot prove that the call sits inside `runDaemon` or that the real config is passed.
Note: the database host and the broker host no longer appear in the startup log.
No change to `docs/conventions`: a new logging rule is outside the request.

Checked `docs/operations.md` and `CLAUDE.md` of all 15 services: only pipecat's `operations.md:16` states that the daemon logs the whole `Config` (agent's line 10 is about the reset url, not the log). All 15 have an `operations.md`; none of the other 14 mentions the config log. Only pipecat is adjusted.
