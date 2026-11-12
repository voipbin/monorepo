# VOIP-1721: remove the environment fallback of provider secrets

Status: design v6 (2026-11-12, design and plan round fixes applied). Base: origin/main ee526650b.

## 1. Ticket review (first section, per the review rule)

Verdict: valid.

- bin-ai-manager, bin-email-manager, bin-message-manager, bin-transcribe-manager and bin-tts-manager
  build their config handler with `confighandler.NewHandler(reg, store, crypto, cfg.EnvValue, ...)`
  and declare 12 secret keys with an `Env:` name. The resolver (`bin-common-handler/pkg/confighandler/resolve.go`)
  reads the database when the group anchor row exists, otherwise the environment, otherwise the default.
- The 12 `Config` fields (`EngineKeyChatGPT`, `GoogleAPIKey`, `SendgridAPIKey`, `MailgunAPIKey`,
  `MailgunWebhookSigningKey`, `AuthtokenMessagebird`, `AuthtokenTelnyx`, `AWSAccessKey`, `AWSSecretKey`
  (transcribe), `AWSAccessKey`, `AWSSecretKey`, `ElevenlabsAPIKey` (tts)) have no production consumer
  besides `EnvValue`.
- The five services already call `<svc>config.LogSources` at start (ai and email log a missing key as
  a warning, message, transcribe and tts as info). So the start-up check is a change of levels and
  wording, not new code.
- The master keys (`*_CONFIG_ENCRYPTION_KEYS`) are environment only and stay.
- Not in scope: the deployment descriptors of the five services (`komodo/docker-compose.yml` and
  `k8s/deployment.yml` keep injecting the variables; after this change the injection is dead for the
  new image, and a rolled back image may still read it), bin-number-manager (it reads its own
  Telnyx variable, a separate setting), bin-pipecat-manager (it reads `GOOGLE_API_KEY` and `ELEVENLABS_API_KEY` from its own
  environment), the shared resolver, the OpenAPI schema and generated models, monorepo-javascript.

## 2. Decisions

D1. Remove the environment name from the 5 key declarations (`Key.Env` is empty), delete the 12 `Config`
fields, the `EnvValue` method, the env constants, the viper bindings and the flags of the five services,
and pass a nil `EnvLookup` to `confighandler.NewHandler`. `lookupEnv` returns "not set" for a nil lookup,
so a key resolves to the database value or to none.
Why this and not removing env support from the shared library: the shared library, `source=env` and
`env_shadowed` are part of the admin API contract that api-manager and the admin UI consume during a
rolling deploy. Both stay as they are, and the values simply never occur (`env_shadowed` is false,
`env_name` is omitted because it is `omitempty`). The library can drop them when it is useful.

D2. Start-up warning. `LogSources` stays the single place. Changes: the `env_shadowed` log field is
removed (it is always false now); a key with source `none` is a warning when the service cannot do its
job without it and an info line when the provider is optional:
- ai: both keys warn (as now). email: all three warn (as now).
- message: the Telnyx token warns (it is the first provider of every send); the MessageBird token is info
  (it only serves the targets Telnyx refused).
- transcribe: info (GCP is the default provider; AWS is optional). tts: info for all three (Google
  Cloud TTS is the primary provider; AWS and ElevenLabs are optional).
The text names the key only (never a value) and says that the key is read from the database only.
The AWS pair warning of tts (keys from different sources) and of transcribe stays. `env_shadowed` is
removed from the `LogSources` lines only; the JSON output field of `*-control config list`, the admin API
and the OpenAPI schema keep it (always false). A database read error stays a warning and never stops the start. The message and tts/transcribe info lines
state "not configured" so the operator can see the state in the log.

D3. The AWS pair rule of transcribe and tts stays: a pair is used only when both keys come from the
database; one stored key without the other is refused (tts: `ErrAWSSourceMismatch`) or logged
(transcribe). Only the doc comments change, since the environment is no longer a source.

D4. Operations text. `config clear` now ends in "no key". Documents state: keys are entered only with
`<svc>-control config set` or the admin API; an image rolled back to the release before this change
reads the environment again as long as the deployment still has the variables; a key that is rotated in
the database only is not rotated for such an image; `GOOGLE_API_KEY` and `ELEVENLABS_API_KEY` are also
read by bin-pipecat-manager from its own environment, so those two are replaced in both places.

## 3. Plan (files)

Past dated plans under `docs/plans/` and `bin-*/docs/plans/` are history and are not edited.

Per service (ai, email, message, transcribe, tts), all paths below are inside `bin-<svc>-manager/`:

1. `internal/config/{main,config}.go`: delete the env constants (and the comment that says they are read
   through `EnvValue`), the `EnvValue` method, the Config fields, the flag registrations, the viper
   bindings and the `viper.GetString` loads of the keys. Keep `EnvConfigEncryptionKeys` and its field.
   The services use `viper.AutomaticEnv()`; a key is read from the environment only when code calls
   `viper.GetString`/`GetInt` on it. Removing the load of the key is what makes the old variable ignored
   (not a barrier of `AutomaticEnv`), so a deployment that still injects the variable starts normally.
2. `internal/configkeys/main.go`: drop `Env:`; rewrite the package and `Keys` comments that say the keys
   "were read from the environment" (ai lines 3 and 22 included). `internal/configkeys/main_test.go`:
   remove the `Env` column of the golden table and the `k.Env` assertion; add an assertion that every key
   has an empty `Env`. `messageconfig/main_test.go`: remove the branch on `k.Env == "TELNYX_TOKEN"`.
3. `internal/<svc>config/main.go`: pass nil to `NewHandler`; fix the comments (package, `New`, the
   supplier docs, tts `ErrKeyUnavailable` and AWS comments, transcribe `AWSKeys` comment; its `accessRes.Source == secretRes.Source` test stays, it is the guard for
   a pair whose two keys differ in their source and costs nothing); `LogSources`
   per D2 (message: `SourceNone` of the Telnyx key is a warning, other `none` stay info; remove the
   `env_shadowed` field from both fields and message strings).
4. Tests of the five `internal/<svc>config/main_test.go`: every case that fed a value through
   `config.Config{...Key: "env-..."}` is rewritten, not just deleted. Three kinds survive: (a) a stored
   value is used; (b) with no row the result is empty/`none` and the supplier returns an empty value with
   no error; (c) a stored row is used and a clear gives `none`. The AWS pair cases (tts `env pair`,
   `db pair shadows the env pair`, `db access with env secret`; transcribe equivalents) are redefined: both
   stored gives the pair, only one stored gives the tts mismatch error or the transcribe empty pair plus
   warning, none gives empty. Tests of `LogSources` assert the level per key and that the line holds no
   `env_shadowed` field (hook test).
5. Other Go tests that name the removed fields or the env behavior, each rewritten for the new behavior:
   `cmd/*-control/config_test.go` (the `config.Config{...Key: "env-value"}` literals: ai `:67`, email `:66`,
   message `:67,233`, transcribe `:67,233`, tts `:71,265`; env fixture, `source: env`, `env_shadowed: true` assertions: the clear
   case now ends in `source: none`; `config list` JSON keeps `env_shadowed` (false) and no `env_name`);
   ai `cmd/ai-manager/platform_keys_test.go` and `builder_wiring_test.go` (the table that used
   `Config.GoogleAPIKey`/`EngineKeyChatGPT` is rebuilt on a handler with stored rows or a stub supplier);
   ai `internal/config/{main_test,builder_test}.go`; email `internal/config/config_test.go`
   (`MAILGUN_WEBHOOK_SIGNING_KEY` case); message `internal/config/config_test.go`; transcribe
   `internal/config/{load_test,main_test}.go`; tts `internal/config/main_test.go`;
   transcribe and tts `cmd/*-manager/startup_log_test.go` (their `secrets` list holds more fields than the
   removed ones: only the AWS and ElevenLabs names are removed (transcribe `:25-26`, tts `:24-26`), the
   `NumField` accounting stays consistent);
   the listenhandler config tests that require `EnvName != ""` from the real registry (ai
   `pkg/listenhandler/v1_configs_test.go:114`, email `pkg/listenhandler/configs_test.go:74`, message,
   transcribe `v1_configs_test.go:114`, tts `:115`): they assert `EnvName` is empty;
   message `cmd/message-control/main_test.go:54-55` and `cmd/message-manager/run_service_test.go:131-132`
   (they set `AuthtokenMessagebird`/`AuthtokenTelnyx` on the Config and stop compiling); the per-service `internal/config/envvalue_test.go` is deleted
   only in its `EnvValue` cases; the tests of the master key (`Test_masterKeyIsEnvOnly`,
   `Test_masterKeyUnprefixedIgnored` where they exist, and the flag cases for the master key at message
   `:55-63`, transcribe `:37-49`) stay or move to the config tests.
6. `*-control` help text: ai `config.go:71,114`, email `:60,102`, message `:63-70,113`, transcribe
   `:65-66,109`, tts `config.go:69,112` and `main.go:42`: remove the statements about the environment
   precedence and about the environment variable "applying again" after a clear (tts `config.go:67`
   included); state that a cleared key is not configured. The comments "stored first, else environment"
   in `cmd/*-manager/main.go` and `cmd/*-control/main.go` (ai-manager `:142`, email-manager `:156`,
   message-manager `:163`, message-control `:72`, transcribe-manager `:159`, transcribe-control `:76`,
   tts-manager `:155`) are fixed too.
7. Docs: each `CLAUDE.md` (ai `:23,25`; email run example `:51-53` and resolution order; message run
   example and `AUTHTOKEN_*` text; transcribe `:69`; tts `:43`), each `docs/operations.md` (config table,
   resolution order, "stored value wins/environment fallback" sections, email `env_shadowed`, the
   rollback and rotation text per D4) and `docs/architecture.md` / tts `docs/domain.md` where they mention
   the fallback. The extracted data files `docs/.docs-gen/bin-<svc>.extracted.json` and the config table are
   regenerated with `bash docs/reference/extractor.sh <service-dir>` (those data files are at the
   repository root `docs/.docs-gen/`, not inside the service directory; if the extractor reports no
   change, nothing is committed there).
8. `bin-api-manager/docsdev/source/`: `self_hosting_providers.rst` (the provider variables are now input
   with `config set`, with the upgrade note and the pipecat note), `self_hosting_first_login.rst`
   (email provider setup text), `service_config_overview.rst` (source `env` is not produced by the
   current services), `service_config_struct_service_config.rst` (example without `env_name` and with
   `env_shadowed: false`). Then `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html
   source build` and `git add -f bin-api-manager/docsdev/build/`. If `self_hosting.html` has its text from
   an included source, the included source is found by grep and fixed too.

Shared code stays as is: `bin-common-handler` (including `requesthandler/config_test.go` fixtures that
use `env_name`), OpenAPI, generated models.

## 4. Verification (goal-driven)

- A key whose old environment variable is set resolves to source `none` when the database has no row
  (test with a handler built by the service `New`).
- A handler whose lookup function returns a value for the old variable name still resolves `none`,
  because `Key.Env` is empty (the two safeguards are tested separately: nil lookup, and a fake lookup
  passed straight to `confighandler.NewHandler` with the service registry).
- A stored value is used (unchanged tests stay green).
- `LogSources` per D2 (hook test), key name only, no `env_shadowed` field.
- Start with the old variable still exported: the config load ignores it without an error (a test sets
  the old variable with `t.Setenv`, loads the config and checks the load succeeds).
- `*-control config list` JSON: `env_shadowed` is false and `env_name` is absent.
- Per service: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`;
  then the root `scripts/check-*.sh` gates.
- Residue scan, three patterns over the five service directories (excluding `vendor/`, `docs/plans/`):
  (1) the 12 variable names (`ENGINE_KEY_CHATGPT|GOOGLE_API_KEY|SENDGRID_API_KEY|MAILGUN_API_KEY|MAILGUN_WEBHOOK_SIGNING_KEY|AUTHTOKEN_MESSAGEBIRD|AUTHTOKEN_TELNYX|AWS_ACCESS_KEY|AWS_SECRET_KEY|ELEVENLABS_API_KEY`, matched case-insensitively so the lower case flag names
  (`engine_key_chatgpt`, `google_api_key`, ...) are found too);
  (2) the Config identifiers, restricted so the legitimate `configkeys.Key*` constants do not match:
  `cfg\.(EngineKeyChatGPT|GoogleAPIKey|SendgridAPIKey|MailgunAPIKey|MailgunWebhookSigningKey|AuthtokenMessagebird|AuthtokenTelnyx|AWSAccessKey|AWSSecretKey|ElevenlabsAPIKey)|Config\{[^}]*(EngineKeyChatGPT|GoogleAPIKey|SendgridAPIKey|MailgunAPIKey|AuthtokenTelnyx|AWSAccessKey|ElevenlabsAPIKey)|\bEnvValue\b`
  must have no hit; (3) `env_shadowed|EnvShadowed` may remain only in `cmd/*-control/config.go` and its
  `config_test.go` (the JSON output field kept by decision, asserted false). Hits of pattern (1) are allowed only in `komodo/docker-compose.yml`,
  `k8s/deployment.yml`, the tests that assert the old variable is ignored, and the sentences of
  `CLAUDE.md` and `docs/operations.md` that explain the change; the PR notes list each allowed hit.
- Meaningful old-variable start check: a test using `reflect` asserts that `config.Config` has no field
  with a removed name, and a second test sets the old variable with `t.Setenv`, loads the config and
  asserts the master key and the other fields are unaffected (a load error would fail it).
- Mutation lens for the tests: `New` passing `os.LookupEnv` instead of nil (killed by the test with the old
  variable exported and a service-`New` handler, with `Key.Env` restored in the test registry only);
  `Key.Env` restored for one key (killed by the golden test); a fake lookup that returns a value while a
  stored row exists still gives the stored value; `awsSourcesDiffer` changed to `a != b` (killed by the
  both-none case); a stored AWS key alone gives the mismatch error by `errors.Is`; a log line holds no
  secret text (a fake stored secret is written and the whole log output is searched); a database read
  error in `LogSources` logs a warning and returns.
- Not verifiable here: a live provider call (cost). The deployment effect is for the QA and the post
  deploy log check.

## 5. Risks

- A deployment whose database has no row for a key loses that provider at the next deploy. Before the
  merge the person who verifies it checks, per service, `<svc>-control config list`: every key the
  deployment uses must show `source=db` and `has_secret=true`, not `db_undecryptable`. A key with
  `source=env` or `none` at that time means the provider stops after the deploy. The check is by the
  verifier, outside this change; the check is per key, because the keys are independent rows.
- During a rolling deploy the `env_shadowed` flag in the admin API can differ per replica (an old replica
  reports true while the row and the variable both exist; a new one reports false). Both are harmless.
- Rollback asymmetry: an image from before this change reads the environment again. A key that is rotated
  in the database only is not rotated for that image. The deployment variables must stay until the
  rollback window of this change has ended; removing them is a later change.
- Rolling update: an old replica (reads the environment) and a new replica (database only) coexist for
  minutes. With the database rows present both resolve the same value (the database wins in the old
  code too).
- Self-hosted installs that pass the keys only through compose environment lose them at the upgrade.
  That migration is a separate change in the install repository.
