# stored settings (`tts_configs`) for bin-tts-manager and the `tts` service of the admin API

## 1. Requirements check (against origin/main 5526516e6)

`bin-tts-manager` reads three credentials from the environment once at start (`internal/config/main.go`:
`AWSAccessKey`, `AWSSecretKey`, `ElevenlabsAPIKey`, loaded by `LoadGlobalConfig`). The same three are the
only credentials it handles (GCP uses application default credentials from a file, not from `Config`).
`bin-email-manager` and `bin-ai-manager` already have the
stored-settings feature and the admin API; this change brings the same to tts-manager. Evidence, all read in code:

- Consumers of the three values (grep of `config.Get().AWS*` and `ElevenlabsAPIKey`, nothing else uses them):
  `cmd/tts-manager/main.go:143` (`NewTTSHandler`), `:152` (`NewStreamingHandler`), `cmd/tts-control/main.go:41-42`.
- Batch path: `audiohandler.NewAudioHandler` builds one Polly client in the constructor and keeps it
  (`audioHandler.awsClient`); it returns nil on error and `NewTTSHandler` then returns nil (a fatal at start).
  Streaming path: `awsHandler.createClient` builds a client in `Init` and `elevenlabsHandler.connect`
  writes the key into the handshake header in `Init`; `Init` runs per message (section 5). The Polly client construction exists in both packages.
- `migrations/` has `000001_baseline` only; `main.go` runs `migrationhandler.Run` before the cache and RabbitMQ.
- Listener: `regV1Speeches = /v1/speeches` has no anchor at all, `regV1Speakings` only an end anchor, the four speaking id routes and the six
  streaming routes (`resV1Streamings*`) only an end anchor. `processRequest` builds its log entry with the whole
  request (`"request": m`) first, and the default branch logs `m.Data`.
- Komodo: `komodo/docker-compose.yml` passes `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`, `ELEVENLABS_API_KEY`. The same
  Komodo variables are read by other services: pipecat-manager (`ELEVENLABS_API_KEY`, four containers,
  `run.py:517`) and transcribe-manager (`AWS_ACCESS_KEY`, `AWS_SECRET_KEY`). Open PRs for the other services: none
  (checked at the base commit); this change does not read or touch their files.
- Design note: "use the atomic group rule for the AWS pair". `confighandler` supports atomic
  groups, but the anchor of a group must be exactly one secret key (`registry.go findAnchor`), so a group of two
  secrets is refused by `NewRegistry`. See section 4.

Everything else was confirmed in code. The change is an addition (new tables, new keys, new route mount);
with no row the service behaves exactly as before.

## 2. Change points (one PR, no split)

| # | Module | Change |
|---|--------|--------|
| 1 | bin-tts-manager `migrations` | `000002_tts_configs_create_table_all`, `000003_tts_config_histories_create_table_all` (the SQL of ai 000004 and 000005 with the table names changed), `Test_configTables`, collation guard test |
| 2 | bin-tts-manager `internal/configkeys`, `internal/ttsconfig`, `internal/config` | three key declarations, the handler wiring and the suppliers, `EnvValue` and the master key list `TTS_CONFIG_ENCRYPTION_KEYS` |
| 3 | bin-tts-manager `pkg/dbhandler` | `config_store.go` (the ai-manager implementation), `ConfigStore()` on `DBHandler` (+ mock), sqlite test schema `scripts/database_scripts_test/table_tts_configs.sql` and `table_tts_config_histories.sql` |
| 4 | bin-tts-manager `pkg/audiohandler`, `pkg/ttshandler`, `pkg/streaminghandler`, `cmd/tts-manager`, `cmd/tts-control` | keys come from supplier functions read at the time of use, no key string argument |
| 5 | bin-tts-manager `cmd/tts-control/config.go` | `config list|set|clear` (the ai-control implementation) |
| 6 | bin-tts-manager `pkg/listenhandler` | `configrpc.Server` mount, early branch, `v1_configs.go`, tests |
| 6b | bin-tts-manager `cmd/tts-manager/main.go` | `runListen` and `runListenNormal` get a `configServer` argument; `runListenPod` keeps its signature and passes nil to `NewListenHandler` (config requests use the shared queue only, and a config-like request on the pod queue gets the fixed 404); creation order of the handlers (section 5) |
| 6c | bin-tts-manager `go.mod`, `go.sum` | test only dependencies for the store tests (section 12); `go mod tidy` and `go mod vendor` as always |
| 7 | bin-tts-manager `komodo/docker-compose.yml` | one line (the variable it refers to has to exist before the merge, which the PR text says), placed after the `ELEVENLABS_API_KEY` line (the media sidecar service only serves files and needs nothing) `TTS_CONFIG_ENCRYPTION_KEYS=[[BIN_MANAGER__TTS_MANAGER_CONFIG_ENCRYPTION_KEYS]]` |
| 8 | bin-common-handler `pkg/requesthandler/config.go`, `config_test.go` | `tts` maps to `QueueNameTTSRequest`, label `tts/configs` (fixed text) |
| 9 | bin-api-manager `pkg/servicehandler/service_config.go`, `server/service_configs.go` (comment), new `service_config_tts_test.go`, `service_config_test.go`, `server/service_configs_test.go` | `serviceConfigServices["tts"] = true`; the comment about the 512 byte limit names the keys neutrally; tests for the four operations with `tts` and for other values |
| 10 | bin-openapi-manager `openapi/openapi.yaml`, `paths/service_configs/key_secret.yaml` | enum `tts` (`ServiceConfigServiceNameTts`), neutral texts; regenerate `gens/models`, api-manager `gens/openapi_server`, `gens/openapi_redoc/openapi.json` |
| 11 | docs | RST `service_config*.rst` + `docsdev/build` pages, `bin-api-manager/docs/routing.md`, `bin-common-handler/docs/usage.md`, tts-manager `docs/architecture.md`, `docs/operations.md`, `CLAUDE.md`, `docs/workflows/service-owned-migrations.md` table list. The self hosting page `self_hosting_providers.rst` is not changed: it describes the environment variables and a restart, which stay valid; a stored value (which needs the master key list that the install stack does not pass yet) is an optional override that makes the restart unnecessary for that key |

`pkg/dbhandler/config_store.go` is a third near-copy of the same store (email, ai, tts). It is copied on purpose:
moving it to `bin-common-handler` would edit two services that are not part of this task, and a new abstraction is out of scope. The package would qualify for promotion (three services) and the other services will add more copies, so
promoting it is a suggestion for a separate change.

Not changed: bin-common-handler `confighandler` and `secretcrypto` (no interface change, no mock regeneration),
number-manager, email-manager, ai-manager, other services. `docs/workflows/service-owned-migrations.md` gets the two table names in the owner list and two description rows
(the format of the `ai_configs` rows).

## 3. Keys and classification

Three keys, all `KindSecret`, `TypeString`, `MaxLen` 512 (the longest of them is an ElevenLabs key or an AWS secret
of about 40 bytes; 512 is the precedent of the email and ai keys and leaves room for a session token style value).
Names match `^[a-z0-9][a-z0-9._]{0,127}$`. A name never changes its kind. None of the three environment names is
reserved or ends with `_ENCRYPTION_KEYS` (`registry.go` `reservedEnv` and the suffix check), so `NewRegistry` accepts
them; the golden test builds the registry. `TTS_CONFIG_ENCRYPTION_KEYS` is the master key list, not the `Env` of a key.

| Key | Env (read when the database has no row) | Used by |
|-----|-----------------------------------------|---------|
| `aws.access_key` | `AWS_ACCESS_KEY` | Polly in the batch fallback and in the AWS streaming provider |
| `aws.secret_key` | `AWS_SECRET_KEY` | same, together with the access key |
| `elevenlabs.api_key` | `ELEVENLABS_API_KEY` | ElevenLabs streaming handshake |

Stay in the environment (infrastructure, bootstrap, or not a secret): `GCP_TTS_ENDPOINT` (an endpoint, not a secret),
`GOOGLE_APPLICATION_CREDENTIALS` (a file path of the GCP identity, not part of `Config`), `POD_IP`, `HOSTNAME`
(routing identity, read with `os.Getenv`), `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_*`, `PROMETHEUS_*`. Infrastructure
credentials stay in the environment by the design of the stored settings (the settings hold API credentials and
operating parameters, not bootstrap values). Nothing else is added.

## 4. Resolution and the AWS pair

Order per key: stored value, then the environment, then nothing (`confighandler`; no code change there). The read
happens for every use and no key value is cached (by design a read per use keeps every replica in step and needs no invalidation; the only thing kept is the Polly client built from the last pair, section 5). An error that is not "no row" (database error,
undecryptable, invalid) is an error and is never replaced by the environment value.

The two AWS keys form one credential. The shared library offers an atomic group only for one secret plus plain
values; a group of two secrets is refused (section 1). Options considered:

- A. Make `aws.access_key` a plain value in an atomic group anchored by the secret. Rejected: the admin API can
  only write secrets (`PUT .../secret`), so the access key could not be changed through it, and the value would be
  stored unencrypted.
- B. Allow several secrets in an atomic group (an explicit anchor in the library). Rejected for now: it changes a
  library that several services use. The other service that reads the same AWS pair has no open change today, but if
  it adds its own configs later, two edits of `registry.go` would collide. The trigger to reconsider is the second
  service that needs it (then one library change serves both).
- D. One secret `aws.credentials` holding both values (a single write makes a rotation atomic). Rejected: the
  environment has two variables (`AWS_ACCESS_KEY`, `AWS_SECRET_KEY`) and a key declares one `Env`, so a combined
  value would need a made-up format that the environment fallback and the admin API (`env_name` per key) can not
  express. The price of
  choosing C instead of D is the rotation window described below (two writes instead of one).
- C (chosen). Keep both as independent secrets and make the consumer refuse a mixed pair. `ttsconfig.AWSCredentials`
  reads both keys and returns the pair only when both come from the same side: if exactly one of them has the source
  `db`, it returns `ErrKeyUnavailable` (the error says the sources differ, no value). Both from the environment, both
  stored, or none: as before. This gives the atomic effect (a stored key and an environment key are never combined)
  with no library change, in the service that needs it.

Limits of C, stated plainly: it is a check of the sources, not a transaction. When both keys are already stored and
an operator changes only one of them (or reads them while a write lands between the two reads), both are `db` and
the pair is accepted although it mixes an old and a new value; AWS then rejects the signature (an AWS error, not a
consumer error) until the second write is done. A rotation is therefore two back-to-back writes with a short window
in which AWS calls fail.

Consequence: while only one of the two is stored, AWS calls fail (closed) and GCP is unaffected. The operator stores
both, back to back; `config list` shows the sources, `env_shadowed` and an explicit warning at start (`LogSources`)
when the sources of the two differ. `config clear` of one of them has the same effect, so clear both.

## 5. Supply of the keys to the consumers

`ttsconfig` defines `KeyFunc func(ctx) (string, error)` and `AWSCredentialsFunc func(ctx) (access, secret string, err error)`.
Each call reads the keys with one 5 second bound for the whole read (`WithTimeout` to change). A source of none returns empty values and
no error, as the service always did (the provider then refuses; unchanged behavior).

- audiohandler: `NewAudioHandler(ctx, awsCreds, gcpEndpoint)`. The `awsClient *polly.Client` field is replaced by the credential supplier, a client builder and a one client
  slot; `awsAudioCreate` reads the credentials and gets the client through `pollyClient`, which calls the existing
  `awsGetClient`. The credentials are read for every call; the client built from them is reused while the pair is unchanged and
  a client exists (one mutex guarded slot of the last pair and its client; an empty slot, `slotClient == nil`, never
  counts as a match, so the first call and a pair of two empty strings still build a client, as the old constructor
  did), and a changed pair builds a new client. Without the reuse every cache miss would build a new HTTP transport (a TLS
  handshake and idle connections per call), and the batch path is the hot path. The slot is not a cache of the keys:
  every call reads both keys again and compares them with the pair of the slot. A replaced client is not closed
  explicitly; its idle connections close by the transport idle timeout (a rotation is rare).
  A key change is used by the next request, no restart.
  Behavior change: the constructor no longer builds an AWS client, so a failure of `LoadDefaultConfig` (a broken
  `AWS_PROFILE` or shared config file) is no longer a start failure; it becomes an AWS error of the request and the
  fallback. A nil handler now comes from GCP client creation only.
- Construction order in `cmd/tts-manager/main.go`: the suppliers need the database handler and the config handler, so the
  database handler (and `LogSources`) is created before the tts and streaming handlers. A Redis connection failure
  (a fatal) can therefore come before a failure to build the tts handler; both stop the start as before.
- streaminghandler: `NewStreamingHandler(req, notify, podID, elevenlabsKey KeyFunc, awsCreds AWSCredentialsFunc)`;
  `elevenlabsHandler.connect` calls the supplier before the dial, `awsHandler.createClient` before building the
  client. `Init` runs through `runStreamer`, which `SayInit` calls when the streaming has no vendor session (and the
  session is reset when a message finishes), so the read happens per message of a call, not per call.
- The Polly client construction stays in two places (audiohandler `awsGetClient`, streaminghandler `createClient`):
  they differ in region constant names and lifetime, and sharing them would need a new package for 12 lines; no new
  abstraction. Noted as a follow-up suggestion only.
- Flags: `Config` keeps the three fields and the three flags (`--aws_access_key` and so on); `EnvValue` returns
  the loaded value (flag or environment), like ai-manager, so "no row behaves as before" holds for a flag user too.
- `tts-control create` needs the same keys. It uses the same suppliers on a database connection opened for the
  command (`DATABASE_DSN` is in the container environment, as for `migrate`). Without it, after a key moved into the
  database the CLI would silently use the stale environment key. `config` and `migrate` need only the database. A consequence: `create` now needs the database too, so with the
  database down it fails even for GCP (the read itself is bounded to 5 seconds and only happens when an AWS
  attempt is made; the connection is opened at the start of the command). `create` is a manual tool; accepted.
- Availability change: before this change AWS and ElevenLabs requests did not depend on the database. After it, an AWS
  batch attempt (cache miss), an AWS streaming message and an ElevenLabs streaming message read the database even when
  no row is stored. While the database can not be read these calls fail (never replaced by the environment value) and
  each read can take up to the 5 second bound; GCP batch and GCP streaming are unaffected. Recovery is the database
  itself, or a revert deploy of this change (the tables stay and are ignored, the environment keys apply again).
- Failure behavior per call path (read error or mixed pair):
  - batch `ttshandler.Create`: the AWS attempt fails with the error, and the existing loop falls back to the other
    provider (silently for the caller: `buildAttempts` in `ttshandler/tts.go` lists GCP with an empty voice id after AWS, so a request for
    provider `aws` is then served by GCP with its default voice, and a request for `gcp` that fails has AWS as its second attempt;
    the existing metric `speech_fallback_total{from_provider="aws"}` is the signal, and a rising rate of it after a
    key change means the stored pair is wrong or mixed); GCP never reads the database. A GCP primary that succeeds, and every cache hit, make no read.
    Both attempts failing returns "all providers failed" with the error text of each (no value in it).
  - streaming AWS and ElevenLabs: `Init` fails, so `SayInit` fails for that message (the existing error path of a
    failed vendor handshake); the next message reads again. The read latency is part of the first audio of each
    message. A key changed during a call is used from the next message. GCP streaming does not read the database.
  - Load: one (ElevenLabs) or two (AWS) primary-key reads on a small table per AWS or ElevenLabs request (a batch
    cache miss, or a streaming message), no cache. The read of one call is bounded by one 5 second context
    for the whole pair (not 5 seconds per key); the callers have no deadline of their own for it
    (`processRequest` uses a background context), so this bound is the only one.

## 6. Migrations

`000002_tts_configs_create_table_all` and `000003_tts_config_histories_create_table_all`: the SQL of ai 000004 and
000005 with `ai_` replaced by `tts_` everywhere, including the names of the constraints and indexes (`chk_tts_configs_*`,
`idx_tts_config_histories_*`; a CHECK name must be unique in the schema) (platform wide, no customer column, the CHECK that uses `JSON_TYPE` is cast to
binary, the collation lesson of the email configs migration: a string function compared with a literal fails on a database whose default collation differs). One statement per file, `CREATE TABLE IF NOT EXISTS`, down files drop the
table with `-- DOWN-LOSSY`. No table options (the baseline of tts keeps `utf8mb3` only on `tts_manager_speaking`).
Impact analysis (schema change): no destructive operation (two new empty tables); an old image ignores the tables and
a new image applies them before it listens, so no old/new code meets the wrong schema; empty table, no lock; rollback
is a revert deploy (the tables stay and are ignored); no data loss; the Alembic stream is frozen and untouched; no
other service uses these table names. The two table names are added to `docs/workflows/service-owned-migrations.md`.
`scripts/verify-service-migrations.sh bin-tts-manager` is run on MariaDB 12.3 and MySQL 8.0 if the containers can
be started; where they can not, that part stays unchecked. Any difference the script reports is listed in the PR text with its cause.

## 7. Listener

The design is the one of the ai-manager listener (`bin-ai-manager/pkg/listenhandler/v1_configs.go`, `processRequest`). `NewListenHandler` gets a last argument `configServer *configrpc.Server` (nil allowed; one call
site in `runListenNormal` and one in `runListenPod`; the normal listener gets the server and the per pod listener gets nil). `processRequest` starts with

```
if h.configServer.Match(m) || configrpc.LogSafe(m) != m { return h.processConfig(m) }
```

before the log entry exists (the entry holds `request: m`). `processConfig` recovers a panic with a fixed log text and
a bare 500, answers a fixed 404 when `Match` is false, and observes the latency histogram with the fixed type
`/v1/configs`. Config requests arrive on the shared queue `bin-manager.tts-manager.request` only (the per pod queue
`...request.<HOSTNAME>` carries streaming control). The per pod listener therefore gets a nil server: `Match` is
nil safe and false, so a config-like request on that queue reaches the same early branch and gets the fixed 404, with
nothing logged (a test covers the nil server).

Overlap of the existing route patterns with config URIs (all read from `listenhandler/main.go`):

| Pattern | Anchors | Can match a config URI? |
|---------|---------|--------------------------|
| `/v1/speeches` | none | only if the URI contains the text, for example a query string value of a mistyped secret request; closed by the early branch for every URI that starts with `/v1/config` or ends with `/secret` (`LogSafe(m) != m`); a secret body travels in the request body, never in the URI, and such a request always has one of those two URI shapes |
| `/v1/speakings(\?|$)` | end | same, through a query value |
| `/v1/speakings/<uuid>$`, `/say$`, `/flush$`, `/stop$` | end | a key can not contain `/`; a path like `/v1/configs/a/v1/speakings/<uuid>` is a non-config URI that the guard answers with 404 |
| `/v1/streamings$`, `/v1/streamings/<uuid>(/say_add|/say_init|/say_stop|/say_finish)?$` | end | same |

No current route has a URI that starts with `/v1/config` or ends with `/secret`, so the guard changes no existing
behavior (pinned by a test over every route).

## 8. Master key and deploy order

`internal/config`: `EnvConfigEncryptionKeys = "TTS_CONFIG_ENCRYPTION_KEYS"`, field `ConfigEncryptionKeys`, bound with
`viper.BindEnv` only (no flag; a flag value shows in the process list). A malformed or empty list never stops the
start: it is logged without the value, secrets can not be stored, and a stored secret reports `db_undecryptable`
(the environment keys keep working). `komodo/docker-compose.yml` gets the one mapping line.

Needed outside this PR: (i) the deployment variable that the compose line refers to has to exist before this change
is deployed (the deployment tool substitutes it at deploy time); (ii) the install stack has to pass
`TTS_CONFIG_ENCRYPTION_KEYS` to the container. Entering a value in a running environment is outside this
change; this change only prepares the CLI and the API. Without the master key list the service starts, `config set`
is refused and a stored secret is reported as `db_undecryptable`; the environment keys keep working only for keys that
have no stored row. A stored AWS pair or ElevenLabs key whose master key list is missing or broken is an error for
its callers (never replaced by the environment): batch silently falls back to GCP, streaming fails.

## 9. Admin API contract (API contract)

- `service` values: `email`, `ai`, `tts`; any other value 404 `RESOURCE_NOT_FOUND`. `ServiceConfigServiceName` is used
  by the path parameter and `ServiceConfig.service` only; a client typed `'email' | 'ai'` must widen its type.
- Endpoints unchanged: `GET /service_configs/tts`, `PUT /service_configs/tts/keys/{key}/secret` (`secret`, `revision`),
  `DELETE /service_configs/tts/keys/{key}?revision=N`, `GET /service_configs/tts/keys/{key}/histories`.
- Keys of `tts` (all `kind` secret, `type` string, at most 512 bytes): `aws.access_key` (`env_name` AWS_ACCESS_KEY),
  `aws.secret_key` (`env_name` AWS_SECRET_KEY), `elevenlabs.api_key` (`env_name` ELEVENLABS_API_KEY).
  `aws.access_key` and `aws.secret_key` must be stored as a pair (section 4); the UI should say so.
- Errors and permission: as for `email` and `ai` (project super admin only; direct access refused).
- Two lists and rolling deploy: `serviceConfigServices` (api-manager) and `configQueue` (requesthandler) are separate
  lists, and the same conclusion holds for `tts` as for `ai`: only api-manager has it, the probe of a secret write gets
  `ErrConfigServiceUnknown` before any RPC; only requesthandler has it, 404 from the allow list; both have it and a
  tts-manager replica is older, the probe gets a bare 404 and the secret write answers 503 before the secret is sent.
  tts-manager runs two replicas and both lists share the queue `bin-manager.tts-manager.request`, so the probe and
  the secret write are two RPCs that can reach different replicas. The probe lowers the chance but does not
  guarantee that a secret reaches a replica that serves the config routes. The defense is the order: store a secret
  (API or CLI) only after the deploy of this change has finished on both replicas.

Single PR or two stages: the allow lists (requesthandler, api-manager) could be a second stage that is merged after
tts-manager has been deployed, which would close the window in which the API can send a secret to an old replica.
One PR is kept: the change is one feature (a service that stores its keys and the API that manages them) and the
two parts are tested together (the listener tests use the real `confighandler`; the requesthandler and api-manager
tests pin the same `tts` value). The window is bounded: both services deploy on merge, and the first secret is entered
after the deploy has been confirmed on both replicas, so the enforcement is the operating rule below, not code. If a
split is wanted later, the second stage is items 8 to 11 of the table (allow lists, enum, RST, routing docs).

## 10. Operations (also written into `docs/operations.md`)

- Which key goes where: the AWS pair is also read by transcribe-manager (the same Komodo variables today). The two
  services will have separate stored settings (per service tables, no shared table), so the same AWS credential must be entered in
  both when it is rotated, tts first or transcribe first, both within the same maintenance step. Check with
  `config list` in each service (`source` of both AWS keys is `db`, `env_shadowed` as expected). Symptom of a
  mismatch: one service authenticates and the other fails at AWS (Polly error in tts, an STT error in transcribe),
  or tts logs that the two AWS sources differ. The trigger to reconsider (a central config service) is that three or more services share a key; nothing is
  synchronized between services here.
- `elevenlabs.api_key`: pipecat-manager reads the same Komodo variable as an environment value in its Python runner
  and has no database. Storing a new key in tts-manager does not change the runner; rotate the Komodo variable (and
  redeploy pipecat-manager) too, and do not remove the variable while pipecat-manager uses it. `config list` of
  tts-manager shows `env_shadowed` for it as long as the variable is set.
- Master key list format `<version>:<base64 of 32 bytes>[,...]`, the highest version encrypts; keep it in the secret
  store of the deployment, never in a flag or in the repository. Without it the service starts and secrets can not
  be stored.
- Do not enter a value while a rolling deploy is half done: an older replica ignores the stored value and does not
  serve the admin API. The finished deploy of this change on both replicas is a precondition of the first value entry.
- `tts-control create` reads the keys the same way and therefore needs the database; `config` and `migrate` need
  only the database and the master key list.
- A fallback from AWS to GCP after a key change is silent for the caller; watch
  `tts_manager_speech_fallback_total{from_provider="aws"}`.
- Recovery: `tts-control config clear --key <key> --actor-id <uuid>` makes the environment value apply again.

## 11. Concurrent changes of other services

As of the base commit `5526516e6` (2026-11-11) no open change adds configs to transcribe-manager or message-manager.
The shared points are `serviceConfigServices`, `configQueue`, the OpenAPI enum and its generated files, the RST pages
and `docsdev/build`, `routing.md` and `docs/usage.md`. This change adds its line after the last existing line in each
list (not in alphabetical order), so a later merge of another service conflicts on a neighbouring line at worst. A
generated file is never merged by hand: on a conflict the branch is rebased on main and the files are regenerated and
verified again, whichever change merges second.

## 12. Verification plan

- Test infrastructure: the tts-manager `dbhandler` tests use `sqlmock` today. The store contract needs a real
  engine, so this change adds the sqlite driver and the SQL file loader the ai-manager tests use (`go-sqlite3`,
  cgo, already built in CI for other services) with a new `scripts/database_scripts_test/` directory with `table_tts_configs.sql` and
  `table_tts_config_histories.sql` (read by a small loader in the test `TestMain`, so the SQL file loader library of ai-manager is not added), plus the MariaDB/MySQL DSN test; no Dockerfile or CI change (the sqlite driver
  is a test only import). The `dbhandler` test package gets a `TestMain` that loads the two schema files into one shared in-memory database
  (the existing tests use `sqlmock` and are not affected); the tables are cleared by each config test.
- Unit tests: key declarations (golden), resolution order and sources, the AWS pair (db+db, env+env, db+env,
  env+db, none+db, read error, undecryptable), a read error is never replaced by the environment, a key change
  reaches the next request without restart for the Polly client (a local fake Polly server), the ElevenLabs
  header (a local fake websocket server) and the AWS streaming client; `tts-control config` (output and logs never
  hold the value, the value only from stdin); store contract tests (sqlite, and a real MariaDB/MySQL through
  `TTS_CONFIG_TEST_MYSQL_DSN` like ai-manager); migration structure tests; listener tests (four routes with the real
  `confighandler`, other routes unchanged, no secret in any log line, response, metric label or panic path); requesthandler
  `tts` routing; api-manager `tts` allowed, others 404; no regression for `email` and `ai`.
- Mutation checks on a scratch copy: remove the mixed pair check, cache a key value instead of reading it for each call (named test: a changed pair reaches the next request), replace a read error by the
  environment value, move the early listener branch after the log entry, remove the `recover`, log the request body.
  Each must fail a named test.
- `go build`, `go vet`, `go test ./...`, `golangci-lint run` in the four modules; `scripts/check-*.sh`; generated
  files regenerated and checked for drift; real MariaDB/MySQL runs of the store tests and of the migrations as far
  as the environment allows. No external provider (AWS, ElevenLabs, GCP) is called: fake servers on loopback and
  fake key values only.

## 13. Out of scope (one line each)

square-admin screen and monorepo-javascript; infra-secret and install changes; the `REDIS_DB` versus
`REDIS_DATABASE=1` name mismatch of the tts-manager compose (not changed here; the service probably uses Redis database 0 today because it reads `REDIS_DB`);
the legacy `k8s/deployment.yml`; per customer keys; reencrypt; cache and event invalidation; unifying the two Polly
client constructors; other services' configs.

## 14. Detailed design (signatures and tests)

`internal/configkeys` (copy of the ai-manager package with the tts names): constants `Table = "tts_configs"`,
`HistoryTable = "tts_config_histories"`, `KeyAWSAccessKey = "aws.access_key"`, `KeyAWSSecretKey = "aws.secret_key"`,
`KeyElevenlabsAPIKey = "elevenlabs.api_key"`, `Keys()`, `NewRegistry()`. Golden test: names, kinds, types, env names, max
length, registry builds.

`internal/config`: `EnvConfigEncryptionKeys`, field `ConfigEncryptionKeys` (bound with `viper.BindEnv` only),
`func (c *Config) EnvValue(name string) (string, bool)` for the three environment names (the loaded value, flag or
environment; an empty value is "not set"), `ResetForTest()` like ai-manager if a test needs it.

`internal/ttsconfig`:

```go
var ErrKeyUnavailable = errors.New("the TTS provider key could not be read")
var ErrAWSSourceMismatch = errors.New("the AWS access key and secret key come from different sources")

type KeyFunc func(ctx context.Context) (string, error)
type AWSCredentialsFunc func(ctx context.Context) (accessKey, secretKey string, err error)

func New(cfg *config.Config, store confighandler.Store, opts ...confighandler.Option) (confighandler.Handler, error)
func ElevenlabsKey(h confighandler.Handler, opts ...SupplierOption) KeyFunc
func AWSCredentials(h confighandler.Handler, opts ...SupplierOption) AWSCredentialsFunc
type SupplierOption func(*supplierConfig)
func WithTimeout(d time.Duration) SupplierOption // default 5 seconds for one whole read
func LogSources(ctx context.Context, h confighandler.Handler)
```

`AWSCredentials`: one `context.WithTimeout` for both reads; `h.Secret` for the access key then the secret key; any
error returns `fmt.Errorf("%w: %w", ErrKeyUnavailable, err)`; when exactly one `Resolved.Source` is `db`, it
returns `ErrKeyUnavailable` wrapping `ErrAWSSourceMismatch`; otherwise the revealed pair. `LogSources` logs key name,
source and `env_shadowed` per key (never a value), logs source none at info level (a provider key is optional), errors for `db_undecryptable` and
`db_invalid`, and warns once when the two AWS sources differ in the way `AWSCredentials` refuses.

`pkg/audiohandler`: `NewAudioHandler(ctx, awsCreds ttsconfig.AWSCredentialsFunc, gcpEndpoint string) AudioHandler`;
`audioHandler` has `awsCreds`, `awsBuild func(ctx, accessKey, secretKey string) (*polly.Client, error)` (the constructor
sets `awsGetClient`), and the slot (`mu sync.Mutex`, `slotAccess`, `slotSecret`, `slotClient`). `pollyClient(ctx)` calls
`awsCreds`, returns the slot client when `slotClient != nil` and the pair is equal, else `awsGetClient` and replaces the slot. `awsAudioCreate`
calls `h.pollyClient(ctx)` where it used `h.awsClient` (the method uses `awsBuild` when set and `awsGetClient`
otherwise, and returns an error when `awsCreds` is nil, so a literal `audioHandler{}` never dereferences nil). Tests
construct `audioHandler{awsCreds: <fake>, awsBuild: <builder of a local fake client>}`.

`pkg/ttshandler`: `NewTTSHandler(awsCreds, gcpEndpoint, mediaBucketDirectory, localAddress, requestHandler)`.

`pkg/streaminghandler`: `NewStreamingHandler(req, notify, podID, elevenlabsKey ttsconfig.KeyFunc, awsCreds ttsconfig.AWSCredentialsFunc)`,
`NewElevenlabsHandler(req, notify, keyFunc)`, `NewAWSHandler(req, notify, creds)`; `connect` and `createClient` call the
supplier first and return its error wrapped.

`pkg/dbhandler`: `NewConfigStore(db *sql.DB) confighandler.Store`, `ConfigStore()` on `DBHandler` (mock regenerated).
The exported `IsDuplicateKey` of the ai copy is not copied (nothing in this service calls it).

`cmd/tts-control`: `loadConfig(cmd)` (bind flags and load the global config, used by the root and by the `config`
group), `cmdConfig()` with `list`, `set --key --actor-id` (value from stdin, one line, at most 4096 bytes), `clear --key
--actor-id`, history source `cli`; `initHandler` (a function today) opens the database and builds the suppliers with the same
`ttsconfig.New`. Variables replaced by tests: `configOut`, `configIn`, `openConfigHandler`, `openTTSHandler` (the `create` path).

`pkg/listenhandler`: `NewListenHandler(..., configServer *configrpc.Server)`, field `configServer`, `v1_configs.go`
(`processConfig`), early branch in `processRequest`.

Test list (names are the test contract):

- `internal/configkeys`: `Test_Keys_golden` (names, kinds, types, env names, max length; the registry builds).
- `internal/ttsconfig` with the real `confighandler` on the in-memory store of its test helper:
  `Test_AWSCredentials_dbDb`, `_envEnv`, `_none`, `_noneEnv`, `_mixed` (db+env, env+db, none+db, db+none), `_readError`,
  `_undecryptable`, `_oneTimeoutForPair`, `_changeReflected` (a row written between two calls is used by the next
  call); `Test_ElevenlabsKey_stored`, `_env`, `_readError`, `_undecryptable`, `_changeReflected`;
  `Test_New_noMasterKey*`; `Test_LogSources_*` (no value in any line, one warning when the two AWS sources differ,
  `db_undecryptable` at error level, source `none` at info level because a provider key is optional); and
  `Test_noSecretInErrorsOrLogs` (a sentinel key value never appears in an error string or a log line of the
  suppliers or of `LogSources`).
- `pkg/audiohandler`: `Test_pollyClient_reuseAndRotate`, `Test_pollyClient_concurrent` (`-race`),
  `Test_pollyClient_nilCreds`, `Test_pollyClient_emptyPair` (none+none builds a client or returns an error, never a nil client or a panic), `Test_pollyClient_nilBuild` (a nil `awsBuild` uses `awsGetClient`; nothing panics), `Test_pollyClient_buildError`, `Test_awsAudioCreate_keyChangeNextRequest` (fake Polly
  server through `awsBuild`), `Test_awsAudioCreate_supplierError` (the error carries no key text).
- `pkg/streaminghandler`: `Test_elevenlabs_connect_headerFromSupplier`, `Test_elevenlabs_connect_supplierError`,
  `Test_awsHandler_createClient_*`, `Test_NewStreamingHandler_*`.
- `pkg/dbhandler`: `Test_configStore_contract` (sqlite, and a real MariaDB or MySQL through
  `TTS_CONFIG_TEST_MYSQL_DSN`), `Test_configStore_*`, the CHECK constraint tests.
- `migrations`: `Test_configTables`, `Test_noCollationDependentCompare`, `Test_embeddedMigrations`.
- `pkg/listenhandler`: `Test_processRequest_configs` (four routes with the real `confighandler`),
  `Test_processRequest_config_existingRoutesUnchanged` (every existing route), `Test_processRequest_config_nilServer`
  (fixed 404, nothing logged), `Test_processRequest_configsHoldNoSecret` (no log line, response or metric label holds
  the secret, for the routed, failing, mistyped, overlapping and panicking cases), `Test_processConfig_panicIsRecovered`. The per pod listener with a
  real server is not verified by a test (the decision is nil).
- `cmd/tts-control`: `Test_config_set_list_clear`, `Test_config_set_eachKey`, `Test_config_list_noValue`,
  `Test_config_errors` (stdin limits, actor and key required), `Test_readValue`, `Test_initHandler_databaseError`,
  `Test_runCreate_releasesWhatInitHandlerOpened`.
- `bin-common-handler` and `bin-api-manager`: `tts` routing and permission tests next to the `ai` ones.

### 14.1 Existing tests that the signature changes touch

Must change (compile or behavior):

| Package | Tests and symbols |
|---------|-------------------|
| `pkg/audiohandler` | `gcp_client_test.go` `Test_NewAudioHandler` (new arguments; the subtest "returns nil when the aws client cannot be created" no longer holds because the constructor builds no AWS client: it becomes a `pollyClient` error path test; the `h.awsClient == nil` assertion goes), `aws_wav_test.go` (five `audioHandler{awsClient: ...}` literals become `awsCreds` and `awsBuild`) |
| `pkg/ttshandler` | `new_handler_test.go` (`Test_NewTTSHandler_missingGCPCredentials` and the constructor calls) |
| `pkg/streaminghandler` | `constructors_test.go`, `aws_init_test.go`, `elevenlabs_init_test.go` (the `accessKey`, `secretKey`, `apiKey` fields become supplier fields) |
| `pkg/listenhandler` | `main_test.go` (`NewListenHandler` call); tests that build `&listenHandler{}` keep a nil `configServer` |
| `cmd/tts-manager` | `runlisten_test.go`: the `runListenNormal` calls (two) and the `runListen` calls (two) get the new argument, the `runListenPod` calls (two) do not; `startup_log_test.go`: the new `ConfigEncryptionKeys` field goes into the `secrets` list so the start log still never prints it |
| `bin-common-handler` `pkg/requesthandler` | `config_test.go` `Test_configQueue`: the row that uses `tts` as an unknown service must change (it now maps to `QueueNameTTSRequest` and the label `tts/configs`); the unknown values become `nothing` and similar, and a `tts` success row is added |
| `bin-api-manager` | `pkg/servicehandler/service_config_test.go` (the row "a service that has no config api" uses `tts` and expects not found; it moves to another unknown value such as `voice`, because `nothing` is already used by the row above it), `server/service_configs_test.go` `Test_serviceConfigs_serviceNameIsNotCheckedByTheBinding` (uses `tts` as a value that is not allowed; it moves to `voice` so the test keeps its meaning; a change for meaning, not for compilation) |
| `internal/config` | `main_test.go` (field comparison, `EnvValue`, the master key list is env only) |

Regression check only (no edit expected): `audiohandler/aws_test.go`, `aws_errorpaths_test.go` (`awsGetClient` is
unchanged), `ttshandler/main_test.go`, `streaminghandler/aws_test.go`, `elevenlabs_test.go`,
`cmd/tts-manager/rundaemon_test.go`, `cmd/tts-control/initcommand_test.go`, `main_test.go` (they do not reach
`initHandler`; new tests cover it).

Regression of the other services: the `email` and `ai` rows of the same files stay green, the unknown values `nothing`
and `EMAIL` stay 404, and in `requesthandler/config_test.go` a value that is not `tts`, `email` or `ai` returns
`ErrConfigServiceUnknown`.

### 14.2 Client slot, test seams and ordering

- Slot concurrency: `pollyClient` reads the pair and builds the client outside the lock; under the lock it only
  compares with the slot and replaces it (last write wins). The mutex is never held during `LoadDefaultConfig` or a
  network call. A nil `awsCreds` returns an error (a programming error, never a nil dereference). Test
  `Test_pollyClient_concurrent` runs parallel calls with `-race`.
- Test seam for the Polly client: `audioHandler` has `awsBuild func(ctx, accessKey, secretKey string) (*polly.Client, error)`
  (the constructor sets `awsGetClient`; `pollyClient` uses `awsGetClient` when the field is nil, so a literal `audioHandler{}` does not panic); tests inject a builder that returns a client pointed at a local fake server with
  retries disabled, as the existing `newLocalPollyClient` does. No environment variable is used.
- Order inside `elevenlabsHandler.connect`: `getVoiceID` (an RPC) first, then the key supplier, then the dial. In
  `awsHandler.Init`: `createClient` (supplier inside) first, then `getVoiceID`. `Test_elevenlabs_connect_supplierError`
  sets the voice RPC expectation first.
- `LogSources` runs once at start with a 5 second bound; the start does not wait for more.
- `tts-control` `initHandler` opens the database with `commondatabasehandler.Connect`, builds the config handler on
  `dbhandler.NewConfigStore(db)` (not `NewDBHandler`, which needs Redis), and closes the connection when the command
  returns: it returns the handler and a close function, and a package variable `openTTSHandler` (replaced by tests, like
  `openConfigHandler`) wraps it for `runCreate`.
- Migration checks: `Test_configTables` asserts the file names, the `chk_tts_` and `idx_tts_` name prefixes, one `CREATE TABLE IF NOT EXISTS` statement per up file, no
  customer column, the `-- DOWN-LOSSY` marker and the `DROP TABLE IF EXISTS` of each down file;
  `Test_noCollationDependentCompare` and `Test_collationDependentFuncs` (copied from ai-manager) the cast guard; `scripts/verify-service-migrations.sh bin-tts-manager` runs
  on MariaDB 12.3 and MySQL 8.0: the one step down and up (versions 3, 2, 3, which runs the down of `000003`), and the
  default collation database, where the down runs repeatedly to version 0 and up again (3, 0, 3), which runs the
  down of `000002` as well; the same check runs in CI as the migration collation job. The daemon start check (V1) of
  the script needs `DAEMON_BLOCKS=1` for this service: the daemon connects to the message broker first and keeps
  retrying, so it never exits by itself (the same is true on main before this change); it migrated the database first. A manual run of `tts-control
  migrate down` twice on a scratch database is an extra look, not a requirement. The result of the script goes into
  the PR text.
- Mutation list (each must fail a named test):
  - `ttsconfig`: the mixed pair check removed (`Test_AWSCredentials_mixed`); a key value cached in the supplier
    instead of read per call (`Test_AWSCredentials_changeReflected`, `Test_ElevenlabsKey_changeReflected`); a read
    error or an undecryptable value replaced by the environment value in either supplier
    (`Test_AWSCredentials_readError`, `_undecryptable`, `Test_ElevenlabsKey_readError`, `_undecryptable`); the 5 second
    bound applied per key (`Test_AWSCredentials_oneTimeoutForPair`); a key value put in an error string or log line
    (`Test_noSecretInErrorsOrLogs`).
  - `audiohandler`: the `slotClient != nil` condition removed (`Test_pollyClient_emptyPair`); the keys read once at
    construction instead of per call (`Test_awsAudioCreate_keyChangeNextRequest`); the slot compare removed so a changed pair reuses the old client, or the pair compared by the old
    value only (`Test_pollyClient_reuseAndRotate`); the lock held during the build (`Test_pollyClient_concurrent`).
  - consumers: the supplier bypassed in the ElevenLabs handshake or the AWS streaming client
    (`Test_elevenlabs_connect_headerFromSupplier`, `Test_awsHandler_createClient_*`); a supplier error ignored and the
    request continued with an empty key (`Test_elevenlabs_connect_supplierError` and `Test_awsHandler_createClient_*` (both also assert that the error holds no key text),
    `Test_awsAudioCreate_supplierError`).
  - CLI: `config set` or `list` printing or logging the value or the stdin text (`Test_config_set_list_clear`,
    `Test_config_set_eachKey`, `Test_config_list_noValue`); the actor not required (`Test_config_errors`).
  - schema and keys: a constraint name left as `chk_ai_`, a down file that does not drop, a customer column added
    (`Test_configTables`); an env name or a kind changed (`Test_Keys_golden`).
  - store: the revision check or the duplicate key handling removed (`Test_configStore_contract`).
  - contract: the `tts` mapping or the label removed in requesthandler (`Test_configQueue`), `tts` removed from the
    allow list (the api-manager `tts` tests).
  - listener: one of the four delegations missing (`Test_processRequest_configs`), an existing route changed
    (`Test_processRequest_config_existingRoutesUnchanged`); the early branch moved after the log entry (`Test_processRequest_configsHoldNoSecret`); `Match` or the
    `LogSafe` guard removed (`Test_processRequest_configsHoldNoSecret`); the `recover` removed
    (`Test_processConfig_panicIsRecovered`); the request body added to a log line (`Test_processRequest_configsHoldNoSecret`).
