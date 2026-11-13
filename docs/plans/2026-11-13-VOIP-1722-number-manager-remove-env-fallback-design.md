# VOIP-1722: remove the environment credential fallback of bin-number-manager

Status: design v3 (2026-11-13). Base: origin/main 44cbfc3db.

## 1. Ticket review (first section, per the review rule)

Verdict: valid.

- `pkg/providerhandler/resolve.go` `resolve()` uses the credential of the `number_providers` row when the
  row holds an encrypted secret (a decryption failure turns the provider `db_undecryptable` and never
  falls back). Without a row secret it uses `h.env[name]` (`CredentialSourceEnv`). The env credential is
  built by `NewEnvCredentials` from `TELNYX_TOKEN`, `TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID`,
  `TWILIO_SID`, `TWILIO_TOKEN`; both daemon (`cmd/number-manager/main.go`) and CLI
  (`cmd/number-control/main.go`) pass it to `NewProviderHandler`.
- The store already exists: `number_providers` (secret ciphertext, key version, `config`), admin RPC
  `/v1/number_providers`, public API `/number_providers`. So the "only services that already have a DB
  store" condition holds and nothing new (store, schema, migration) is needed.
- Not in scope: komodo/k8s env lines, deployment variables and deployment secrets (kept for the rollback window,
  as in VOIP-1721), voipbin/voipbin install, monorepo-javascript, the master key
  `NUMBER_SECRET_ENCRYPTION_KEYS` (env only, stays).

## 2. Decisions

D1. Code removal in bin-number-manager: `NewEnvCredentials`, `providerHandler.env`, the `env` argument of
`NewProviderHandler`, `Descriptor.RequiredEnvConfig`, the env branch of `resolve`, the two call sites,
`internal/config` fields `TwilioSID/TwilioToken/TelnyxConnectionID/TelnyxProfileID/TelnyxToken` with their
flags, bindings and loads, and the `telnyx_*` fields of `logStartup`.
`resolve` after the change: row with DB secret as now; otherwise source `none`, `hasSecret` false, `usable`
false, unusable counter reason `no_secret`. A row without a secret therefore never serves, whatever its
`config` holds.

D2. Public API enum. `credential_source` loses `env` (response-only enum; the service never emits it
again). Remove `NumberManagerNumberProviderCredentialSourceEnv` from `bin-openapi-manager/openapi/openapi.yaml`
and the `models/provider.CredentialSourceEnv` constant; regenerate the openapi-manager and api-manager
generated code, the redoc files and the docsdev build. Compatibility (checked in code; api-manager: no response validation, `git grep ValidateResponse|openapi3filter` finds nothing and `server/number_providers.go` returns the handler result with `c.JSON`; square-admin is outside this repository, read in monorepo-javascript `square-admin/src/views/providers/number_providers_list.js:42` and `src/types/api.ts:1218`): api-manager
passes the field through as a string type and does not validate response enums; the only consumer outside
this repo is square-admin, which has `'env'` in its type union and badge map; an unknown or missing
value falls back to a plain badge (`CREDENTIAL_BADGES[...] ?? {label: String(...)}`), and the service
never emits `env` after deploy, so square-admin needs no change for correctness (the dead `env` entries
there can be removed in a later UI change; this PR does not touch monorepo-javascript).
Why different from VOIP-1721, which kept `source=env`: that enum is shared by five services and a
shared library; this one is owned by the number provider API alone.

D3. `DeleteSecret` removes the secret and the provider is left with source `none` (not usable, release
and search fail with "no usable credential") until a new secret is stored. Doc comment, OpenAPI
description, tests and docs updated. No behaviour change in the DB write.

D4. Daemon start with a malformed `NUMBER_SECRET_ENCRYPTION_KEYS`: unchanged behaviour (log, continue with
`NewSecretCrypto("")`, DB secrets `db_undecryptable`), only the comment loses "the environment credentials
apply". The start log states that no provider credential is read from the environment.

D5. Start-up check (minimal). After the handler is built the daemon calls `PurchaseCandidates` once (existing
interface method, no mock change) and logs one Warn line when the list is empty and an Info line with the
count otherwise, in a separate function `logPurchaseReadiness(ctx, ProviderHandler)` in `cmd/number-manager` called in `runServices` right after the handler is built and before the listen and subscribe services start (tested with the ProviderHandler mock: empty list, non-empty list, read error that does not stop the start). A read error is logged and never stops the start. It reuses the wording of
`warnIfNoPurchaseCandidate` (admin.go, unchanged). No per-provider walk, no new metric.

D6. Documents and comments (same PR; first-pass list, the final `git grep` proves completeness):
- number-manager: `CLAUDE.md:20,50,73`; `docs/operations.md:8,73-75,110-115,139,143,146,152`
  (after `DeleteSecret` the provider is `none` and needs a new secret; db_undecryptable recovery);
  `docs/architecture.md:33,74`; `docs/domain.md:47,57,62,71`; regenerate
  `docs/.docs-gen/bin-number-manager.extracted.json` with `bash docs/reference/extractor.sh bin-number-manager`;
  comments: `providerhandler/main.go:44-52,94`, `resolve.go`, `admin.go:275`,
  `cmd/number-manager/main.go:148-153`, `numberhandler/available_number.go:17`, `models/provider`.
- OpenAPI sources: `openapi.yaml` enum, x-enum-varnames, description (about 6592-6602), `usable`
  description (about 6890), `paths/number_providers/id_secret.yaml:49-51`. Regenerate with `go generate ./...`
  in bin-openapi-manager and bin-api-manager; redoc generation skips silently without `npx`, so check it
  exists and that `git grep` finds no `env` enum member in `gens/openapi_redoc/*`.
- RST: `number_provider_overview.rst:21,26,34,42,44`, `number_provider_struct_number_provider.rst:37,75-76`,
  `number_provider_tutorial.rst:36,130,146`, `self_hosting_providers.rst:316-330` (rewritten: the service no
  longer reads these variables; credentials are stored with the admin API; the install bundle is a
  separate change). Clean sphinx rebuild, `git add -f bin-api-manager/docsdev/build/`.
- `bin-message-manager/docs/operations.md:120` (text only).
- Not touched: `docs/plans/*` history, `ServiceConfigKeySource` (different enum that keeps `env`).

D7. Left untouched on purpose: `komodo/docker-compose.yml`, `k8s/deployment.yml` of every service
(dead injection after this change; removal after the rollback window), voipbin/voipbin install.
`k8s/deployment.yml` of message-manager is not touched (not this service).

## 3. Merge gate (not done by this PR)

This is PR 2/2. PR 1/2 (a separate card) adds the `number-control` provider command; both touch
`cmd/number-control/main.go`, so this branch is rebased on main after PR 1/2 merges and conflicts keep the
PR 1/2 command. Auto deploy on merge. Before merge the stored `number_providers` of the running system needs,
for Telnyx, a row with a stored secret and `connection_id` (plus `profile_id` if the live setup uses one),
and a Twilio row (secret, `sid` and `trunk_sid`; `UpdateSecret` refuses a Twilio secret without both, and the environment never had a `trunk_sid`, so it must be looked up) if Twilio numbers must still be released. The stored account must equal the environment
one. Check before merge with the admin API: each needed provider shows `credential_source` `db` and
`usable` true; after deploy: same call, no "no provider is usable" start warning, no increase of
`provider_unusable_total{reason="no_secret"}` after the start (the start check itself adds one count for a provider without a secret, so compare against the value right after the start). This PR performs no production write, purchase or key
rotation, and no account value appears in the PR text.

## 4. Test plan

- Per file: `main_test.go` (env rows become "no secret -> none"; row with config but no secret asserts none,
  not usable, `no_secret` increments, Telnyx and Twilio; `envTelnyx()` removed, `newTestHandler` loses the
  env argument, a seed helper stores an encrypted secret and `connection_id`), `admin_test.go` and
  `admin_errors_test.go` (helper uses move to the seed; delete test expects none and not usable; the
  undecryptable then delete scenario ends in none), `twilio_test.go` (env cases move to DB seeds; add a
  test that a stored token and `sid` with surrounding whitespace work; the env trim test is deleted),
  `startup_log_test.go` (field-count invariant: drop the 5 fields from `secrets`/`logged`),
  `internal/config/main_test.go` (assertions removed, master key binding asserted), cmd init and run tests
  compile, start check helper (empty: Warn, non-empty: Info, error: no stop), api-manager
  `server/number_providers_test.go:418,428` (none, usable false, expected JSON), and a check that other
  tests use no `env`.
- providerhandler (summary): resolve table (no row: none; row without secret: none and ignores row config; DB
  secret ok; missing required config; undecryptable; empty key list), `PurchaseCandidates` (none
  without rows, DB rows ordered), `ForRelease`/`ForSearch` with and without DB secret, `DeleteSecret`
  (source none, not usable), admin tests that used env credentials now seed a DB secret, `Test_NewEnvCredentials_twilioWhitespace` deleted; new test: a stored token (trimmed by `UpdateSecret`, admin.go:234) and a `sid` with surrounding whitespace (trimmed in the Twilio factory, main.go:175) work.
- config: no `twilio_*`/`telnyx_*` keys; `NUMBER_SECRET_ENCRYPTION_KEYS` still bound.
- cmd: `startup_log_test` expects no telnyx ids; start check helper test.
- api-manager `server/number_providers_test.go:418,428` use `CredentialSourceNone`.
- Verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`
  in bin-number-manager, bin-api-manager, bin-openapi-manager, bin-common-handler as touched;
  `scripts/check-*.sh`.
- Completeness proof: a final `git grep` of the removed identifiers and variable names, recorded in the
  PR body.

## 5. Rollout / rollback

New image: reads only DB. The previous image also prefers a stored secret, so a stored secret keeps working
after a rollback; the difference is a row without a secret (or after `DeleteSecret`), where the previous
image uses the environment again while the deployment still has the variables, which is why they stay
(removed after the rollback window in a separate step). Old api-manager with new number-manager: the field
is a plain string; no `env` is emitted. New api-manager with old number-manager: `env` may still be
emitted by the old image; api-manager does not validate the enum on responses (checked), so it passes
through.
