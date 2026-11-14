# VOIP-1734 Remove the stored platform secret code and the service_configs admin API

## 1. Verified claims

Checked on origin/main (5ab4e203c, VOIP-1731 merged).

- Every platform secret is read from the environment only (`pkg/envsecret`). The stored settings
  (`<svc>_configs`, `number_providers.secret_ciphertext`) are written by the admin API and the
  `*-control config` commands and read by no use path. Verified per service: `aiconfig.Supplier`,
  `KeyForBaseURL`, `MCPOAuthClient`, `billingconfig.Supplier/PaddleKeys`, `emailconfig`,
  `messageconfig`, `ttsconfig`, `transcribeconfig.AWSKeys`, `providerhandler/resolve.go`.
- No other open PR touches these paths (open PRs: dependency bumps and an unrelated change).
- `bin-call-manager/pkg/outboundconfighandler` is unrelated (name only).
- Customer-owned secret paths use `secretcrypto` with `ENGINE_BASE_KEY_ENCRYPTION_KEYS` and
  `MCP_SECRET_ENCRYPTION_KEYS` and do not share the master keys of this removal.
- `hasDBSecret` is used (`providerhandler/admin.go` `mergeConfig`), see D7.
- `STT_KEY_UNAVAILABLE`: the only implementation of the key function (`transcribeconfig.AWSKeys`)
  never returns an error, so the branch in `streaminghandler.Start` is unreachable in the service.
  It is kept, see D9.

## 2. Goal and non-goals

Remove the code that stores, encrypts, rotates and serves the global secrets, and the public admin
API of it. No new feature, flag or abstraction.

Not changed: tables and migrations (dropping them is a later change), `number_providers.enabled`,
`priority`, `config`, `pkg/secretcrypto`, `pkg/envsecret`, the customer-owned secret paths,
compose, k8s and komodo files (they still pass the `*_CONFIG_ENCRYPTION_KEYS` and
`NUMBER_SECRET_ENCRYPTION_KEYS` variables; the code no longer reads them and a later change removes
the lines).

## 3. Decisions

D1. bin-common-handler. Delete `pkg/confighandler` (all files including `configrpc` and `storetest`),
`requesthandler/config.go` and its test, the config methods of the `RequestHandler` interface and
mock, `NumberV1NumberProviderUpdateSecret` and `DeleteSecret` (client, interface, mock, tests).
Update `CLAUDE.md` and `docs/usage.md`. Keep `secretcrypto` (after this change ai-manager is its only service user; the customer owned paths
need it, and moving it into ai-manager is a cleanup candidate named in the summary) and `envsecret`.
`docs/usage.md` ("used by") is updated. The moved `IsDuplicateKey` (D4) takes `mysqlDuplicateEntry` and
the `mysql` import with it. The `internal/<svc>config/main_test.go` files are rewritten around
`Supplier` and the environment reads (not just trimmed), because they build their handlers on the
store.

D2. `internal/configkeys` of ai, billing, email, message, tts, transcribe, pipecat stays as the list of
names. Removed in all: `Table`, `HistoryTable`, `Keys()`, `NewRegistry()`, the length constants and the
`confighandler` import. Per service:
- ai: remove the `Key*` store names, `EnvNameOfKey`, `MCPOAuthClientSecretKey`, `MCPOAuthClientIDKey`,
  `mcpOAuthKeys`; keep `Env*`, `EnvNames()`, `MCPOAuthEnvNames`, `MCPOAuthVendors`, `Vendor*`
  (`isMCPOAuthVendor` stays if `MCPOAuthEnvNames` or a kept helper needs it, else it goes).
- billing, email, message, tts, transcribe: remove the `Key*` store names and `EnvName(key)`; keep `Env*`.
- pipecat: has no `Env*` constants. It keeps the `Key*` constants because they are the key names of the
  `platform_keys` map of the runner request (wire contract used by
  `pkg/pipecatcallhandler/platformkeys.go`); the comment is rewritten. The test that checked every
  `Key*` against `Keys()` is replaced by a test over the explicit list of `Key*` constants that
  `platformkeys.go` maps (a table in `platformkeys_test.go`).
After this change `PlatformKeyResolver`, `WithPlatformKeyResolver` and `platformkeys.go` have no
production caller (none did before either); they stay as the wire-name mapping and are listed as a
cleanup candidate in the summary, not removed here.

D2a. Tests of `internal/configkeys/main_test.go` (`Keys()`, `NewRegistry()`, key counts) are rewritten
for the kept names in the seven services (a short table of `Env*` names, and for ai the
`MCPOAuthEnvNames` and vendor checks).

D3. `internal/<svc>config`: remove `New`, `maxSecretPlaintext`, the `confighandler` and `secretcrypto`
imports and the stored-settings text in package comments. `ErrKeyUnavailable`: message and
transcribe have no user, so the variable, its comment and tests are removed. billing and email have no
such variable in their config package. ai: the error is still
named by `builderhandler` (alias in `parse.go`, `errors.Is` checks) although no supplier returns it
(the environment read cannot fail); it stays, and its comment is corrected to say that no producer
remains. tts returns it from the AWS pair check (`ttsconfig/main.go`), so it stays with a corrected comment.
pipecat: `New`, `KeyResolver`, `LogSources` and the whole `pipecatconfig` package are removed (no
remaining content); `cmd/pipecat-manager` and `cmd/pipecat-control` stop importing it.

D4. `pkg/dbhandler/config_store*.go` (all seven services) are deleted. ai-manager's `config_store.go`
also defines the exported `IsDuplicateKey` (used by `aicallhandler/draft_decision.go`): it and its test
(`Test_isDuplicateKey`) move to another file of the package (`dbhandler/main.go` or a new
`duplicate.go`) before the deletion; no other helper of those files has a user outside them. The email
test file is `pkg/listenhandler/configs_test.go` (not `v1_configs_test.go`). Only ai, email and tts expose
`ConfigStore()` on the `DBHandler` interface (`main.go`, `mock_main.go`, regenerated with mockgen); the
other four call `dbhandler.NewConfigStore(sqlDB)` directly from `cmd`. `pkg/listenhandler` (the file `v1_configs.go` with `processConfig` in ai, billing, message, pipecat,
transcribe, tts; email handles it inline in `main.go`): remove the
`configServer` field, the constructor parameter, the `/v1/configs` match, the `configrpc.LogSafe`
guard and `v1_configs_test.go`. The listen handler constructors lose one parameter in every service,
so every call site and test is updated.

D5. Tests that count commands: `cmd/billing-control/main_test.go` compares the root command list and the
`config` subcommand list (`clear`, `list`, `set`); it and any equivalent in the other `*-control`
packages are updated to the new command set (they pass at compile time and fail at run time otherwise).
`cmd/<svc>-control/config.go` and its test are deleted (ai: also `cmd/ai-manager/platform_keys_test.go`
`failingStore` and `Test_platformKeys_neverReadTheStore`, which depend on `aiconfig.New` and
`confighandler.Store`; they are deleted or rewritten without the store; email: `cmd/email-manager/run_test.go`
expectations of `ConfigStore()` and `storetest.NewMemory()`) and the `config` command is unregistered
(check that `main.go` is the only caller of the config command constructor). The binaries and their
other commands stay.

`sqlDB` of the managers. After the config store call goes, check each `cmd/<svc>-manager/main.go`:
ai, billing, email, message, transcribe and tts still pass `sqlDB` to `NewDBHandler`. pipecat uses
`sqlDB` only for `NewConfigStore` (`main.go:157`), so `createDBHandler` stops returning the
connection (signature, caller, `main_test.go`, the function comment and the `database/sql` import
change) and the local `startupConfig` that only fed `pipecatconfig.New` goes too. The migration run in
`createDBHandler` stays.

tts-control. `initHandler` opens and closes a DB connection on the `create` command path; the
`ttshandler` it builds does not use the DB (grep: no DB use in `pkg/ttshandler`), and the connection is
unrelated to `initConfigHandler`. The connection is removed (`openTTSHandler` loses its close function;
`initcommand_test.go` and `main_test.go` follow). `DatabaseDSN` stays in the config because `migrate`
needs it. Behavior change: `tts-control create` no longer fails when the database is down. The
`CLAUDE.md` sentence that says `create` connects to the database is corrected.

D6. The master key variables stay in the deployment definitions; removing those lines is the work of
a later change, and the docs say so (D10). `internal/config`: remove `ConfigEncryptionKeys`, `EnvConfigEncryptionKeys` and the viper
`BindEnv`/`GetString` lines in ai, billing, email, message, pipecat, transcribe, tts and number
(`NumberSecretEncryptionKeys`). Nothing else reads the variables, so presence or absence has no effect
(the compiler guarantees it once the fields are gone; no extra test). Tests that referenced the
fields are changed file by file: `internal/config/{envvalue,providerkeys,load,masterkey,envkey,main,
config}_test.go` as applicable, and the `komodo_compose_test.go` of billing, email, pipecat
(these use `config.EnvConfigEncryptionKeys` and, in pipecat, assert that four containers carry the
variable). The compose and k8s definitions keep the lines (D10), so those assertions are deleted, not
inverted; assertions about the platform secret variables stay. The `komodo_compose_test.go` of
message, transcribe and tts also use `config.EnvConfigEncryptionKeys` and break at compile time; ai
and number use string literals. Handling by test: the master key value tests (`Test_komodoComposePassesTheMasterKeyList`,
`Test_komodoComposePassesTheMasterKeyListToTheGoServicesOnly`, `Test_komodoComposeMasterKeyOccurrences`
and the k8s optional-reference tests) are deleted when they only check that variable; when a table
also checks other names (billing `Test_k8sDeploymentPassesTheSecretsAsOptionalReferences`), only the
master key entry is removed from the table. The compose and k8s files keep the lines, so nothing that
remains asserts them.
The `startup_log_test.go` of pipecat, transcribe, tts and number compare the number of `Config` fields
with a `secrets` list by reflection; the `ConfigEncryptionKeys`/`NumberSecretEncryptionKeys` entries
are removed from those lists (otherwise they fail at run time, not at compile time).

D7. number-manager.
- Remove `UpdateSecret`, `DeleteSecret` (interface, mock, implementation, `secret.go`), the
  `crypto` field and constructor parameter of `providerhandler`, `NewSecretCrypto` calls in
  `cmd/number-manager` and `cmd/number-control`, `NumberSecretEncryptionKeys`, listenhandler routes
  and handlers of `/v1/number_providers/<name>/secret`, `number-control provider secret set|delete`.
- `hasDBSecret` and its branch in `mergeConfig` are removed. Behavior change: before, an update that
  removed a required config key was rejected with 400 only when the row held a stored secret; now it
  is accepted (and `usable` and the environment fill-in of resolve decide whether the provider can
  serve). Tests: update that removes `connection_id` is accepted and the info shows `usable` false
  when no environment value fills it; with the environment value filled it stays usable.
- `provider.Provider` loses `SecretCiphertext`, `KeyVersion`, `SecretUpdatedBy`, `TMSecretUpdate`;
  `field.go` loses the four `Field*` names. `GetDBFields` and `PrepareFields` are driven by struct tags, so
  the columns that stay in the table are neither selected nor written. `key_version` is NOT NULL
  DEFAULT 0 and the others are nullable (migration `000002`), so inserts keep working. Test:
  with the sqlite test schema, a row seeded with a non-empty ciphertext, `key_version` 3 and the audit
  columns is read and updated through the trimmed struct without error and the legacy columns keep
  their values; `NumberProviderEnsure` (insert, duplicate key ignored) on a new name works and leaves
  `key_version` at its default 0.
- `provider.ErrSecretKeyNotConfigured` (defined in `models/provider/provider.go`, used by `secret.go`,
  the 503 mapping in `listenhandler/main.go`, `number-control` and tests) is removed with them. `logSafeRequest` in the number
  listenhandler stays (it covers the whole `/v1/number_providers` prefix, whose config routes remain);
  its `HasSuffix(uri, "/secret")` clause is then redundant and is removed with `regV1NumberProvidersIDSecret`.
- Orphans after the removal, all deleted or adjusted: in `providerhandler` `secret.go` (`maxSecretBytes`,
  `encryptSecret`, `decryptSecret`, `NewSecretCrypto`), `mergeConfig`'s `desc` parameter and error
  return (always nil after `hasDBSecret` goes; the signature and its call in `Update` are simplified),
  the `strings` import of `admin.go`, the `utilHandler` field (used only for `TimeNow()` in the secret
  methods; with its import and constructor argument, if no other user remains after grep); in
  `cmd/number-control`: `newProviderHandler` returns an error only because of `NewSecretCrypto`, so
  its signature, `initNumberHandler`, `initProviderHandler` and their tests change, and the test that
  expects a bad key list to fail goes; `provider.go` loses `maxSecretStdinBytes`, `readSecret`,
  `cmdProviderSecret*`, `runProviderSecret*`, `secretEnvNames`, `printSecretNotice` and the imports they
  alone need; `cmd/number-manager/main.go` loses the `crypto`/`errCrypto` block; in
  `listenhandler/models/request/v1_number_providers.go` the `V1DataNumberProvidersIDSecretPut/Delete`
  DTOs. Tests touched: `models/provider` (`provider_test.go`, `provider_redaction_test.go`,
  `webhook_test.go`), `pkg/dbhandler/provider_test.go` (uses removed `Field*`, `KeyVersion`,
  `SecretCiphertext`; the legacy-column test seeds the columns with raw SQL), `providerhandler`
  (`main_test.go` `NewProviderHandler(nil, db, crypto)`, `secret_test.go`, `admin_test.go`,
  `admin_errors_test.go`, `twilio_test.go`, `resolve_env_test.go` as they use the removed items),
  `listenhandler/v1_number_providers_test.go`, `cmd/number-control/{main,provider}_test.go`,
  `cmd/number-manager/startup_log_test.go`.
- `Info` and `WebhookMessage` lose `key_version`, `secret_updated_by` and `tm_secret_update`.
  `has_secret` and `credential_source` stay (they describe the environment). Breaking change of the
  public response; the alternative (keep the three fields always empty and mark them deprecated) was
  rejected because a field that can never carry a value misleads readers. The only known consumer
  is the admin screen, which stops reading them first (see D10).
- `Provider.String()` redaction of the ciphertext goes with the field; `Credentials` redaction stays.

D8. api-manager and OpenAPI. In `server/number_providers.go` the helpers `numberProviderMaxBodyBytes`
and `bindNumberProviderBody` stay (the config update route uses them) but their comments about a
1024 byte secret are corrected. The `server/service_configs.go` body helpers go with the file.
The `NumberManagerNumberProviderCredentialSource` enum keeps `db` and `db_undecryptable` (not produced
since the previous change; removing enum values of a response is outside this change; named as a
cleanup candidate in the summary). Remove `servicehandler/service_config*.go` and tests, the interface and
mock entries, `server/service_configs.go` and test, `NumberProviderUpdateSecret`,
`NumberProviderDeleteSecret` and their routes and tests, `docs/routing.md` rows, the service_config
and secret sections of `docsdev/source` (RST files, index, number_provider pages, self-hosting
pages) and rebuild `docsdev/build` clean. OpenAPI: delete `paths/service_configs/*` (4 files),
`paths/number_providers/id_secret.yaml`, the schemas that only they use
(`ServiceConfig*`, `NumberManagerNumberProviderSecretRequest`) and the three response fields;
rewrite the sentences that mention a stored secret (`openapi.yaml` credential source and `has_secret`
descriptions, `tm_update`, `paths/number_providers/id.yaml` "a secret is stored" 400 sentence,
`main.yaml`). In `openapi.yaml` also remove the `ServiceConfig` tag, the `ServiceConfigService` and
`ServiceConfigKeyName` parameters, the schemas `ServiceConfigServiceName`, `ServiceConfigKey*`,
`ServiceConfigHistory*` and the enum schemas only they use (verify each with grep before removal), and
the two path references. Regenerate `bin-openapi-manager/gens/models`,
`bin-api-manager/gens/openapi_server` and `bin-api-manager/gens/openapi_redoc` (`openapi.json`,
`api.html`); the redoc step is skipped silently when `npx` is missing, so its presence is checked and a
second run must show no diff. The consumers of the removed public routes are not in this repository
(the admin screen is the only known one; the OpenAPI and RST changelog of the removed routes is the
record for other readers).

D9. Leftovers of the previous change. OpenAPI service_configs wording goes with the paths. tts-control
DB connection: D5. pipecat `KeyResolver`, `LogSources`: D3. `ErrKeyUnavailable` comments: D3.
`STT_KEY_UNAVAILABLE`: unreachable in the service because `AWSKeys` returns no error, but the key
function types (`transcribeconfig.AWSKeysFunc`, `streaminghandler.AWSKeyFunc`) carry an error by
contract, the tests inject it, and removing the typed reason changes a signature that has nothing to
do with stored settings. It stays; the summary names it as a cleanup candidate.

D10. Rollout. The admin screen change that stops calling the removed API is deployed first. Deploy order: the screen first, then api-manager (the public routes disappear), then number-manager and
the seven services that lost `/v1/configs` (ai, billing, email, message, pipecat, transcribe, tts); an
old api-manager that sends a config or number secret RPC to a new service gets 404 (admin only path).
Old and new images read secrets identically (environment only), the tables stay, so the previous
image together with its deployment definition is a valid rollback. Rollback stops being possible
once a later change drops the tables.
Routes that no longer exist are not logged specially: deploy only after no caller of the removed
public routes remains, and deploy api-manager before the services.
Stored data: the ciphertext rows in `<svc>_configs` and `number_providers.secret_ciphertext` stay
after this change and cannot be cleared through the code any more; their deletion belongs to the change
that drops the tables. The docs state: "the code does not
read `<SVC>_CONFIG_ENCRYPTION_KEYS`/`NUMBER_SECRET_ENCRYPTION_KEYS`; the deployment lines are removed
later". Historic design docs under `docs/plans` are kept unchanged.

Implementation rule: the lists above are the known consumers. The implementation finds the rest by
building every module, running `go vet` and the full tests, and the residual grep (section 4); a
consumer found that way is changed with the same intent and recorded in the summary.

## 4. Verification plan

1. Per module in the worktree root, sequentially: `go mod tidy`, `go mod vendor`, `go generate ./...`,
   `go build ./...`, `go test ./...`, `golangci-lint run`, for every module (bin-common-handler is
   changed, so all modules are in scope), compared with the baseline of origin/main.
2. Residual grep (expect 0 outside `docs/plans`, `migrations`, compose, k8s, komodo, and the customer
   owned paths): `pkg/confighandler|\bconfighandler\b|configrpc|ConfigEncryptionKeys|EnvConfigEncryptionKeys|\bServiceConfig|service_configs|UpdateSecret|DeleteSecret|hasDBSecret|ErrSecretKeyNotConfigured|LogSafe|NewConfigStore|ConfigStore\(\)|configkeys\.(Table|HistoryTable)|ConfigV1|NumberV1NumberProvider(Update|Delete)Secret`,
   plus `NewSecretCrypto` limited to `bin-number-manager` (ai `mcpserverhandler/secret.go` and its tests
   are customer owned and keep the name), with `outboundconfighandler`, pipecat `RTVIServiceConfig` and sentinel `loadServiceConfig` excluded (unrelated names),
   and the master key variable names in Go sources and docs wording. `key_version` and
   `secret_ciphertext` are not in the pattern (customer owned paths use them); the leftover list is read
   by hand and must not contain `MCP_SECRET_ENCRYPTION_KEYS` or `ENGINE_BASE_KEY_ENCRYPTION_KEYS`.
   Other test files that use removed items and are changed with them: billing
   `paddlehandler/signature_test.go` (uses `confighandler.ErrKeyNotConfigured`; the billing-owned
   `ErrKeyUnavailable` stays). Docs in scope: `CLAUDE.md`, `docs/architecture.md`, `docs/operations.md` of the seven services,
   `CLAUDE.md`, `docs/domain.md`, `docs/architecture.md`, `docs/operations.md` of number-manager,
   `bin-common-handler/CLAUDE.md` and `docs/usage.md`, `bin-api-manager/docs/routing.md`.
3. OpenAPI and redoc regeneration produce no diff on a second run; docsdev clean build has no warning
   that the baseline does not have.
4. Every `scripts/check-*.sh` gate that CI runs is executed on the committed tree.
5. New tests (written with the implementation): D7 legacy-columns row and the required-key update,
   command registration (no `config` command in the `*-control` roots), listen handler answers not
   found for `/v1/configs` and for `/v1/number_providers/<name>/secret`, pipecat `Key*` mapping table.
6. `go vet ./...` and `go test -run xxx ./...` (compile of every test) in every module, in addition to
   the normal test run.
7. The sqlite test schemas `bin-<svc>/scripts/database_scripts_test/table_*configs*.sql` (email:
   `scripts/database_scripts/`) stay (the tables stay);
   `scripts/check-*.sh` is run to confirm that orphan schema files are not flagged.
