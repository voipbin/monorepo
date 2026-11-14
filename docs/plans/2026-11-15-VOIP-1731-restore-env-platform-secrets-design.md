# VOIP-1731: Platform secrets are read from environment variables again

## 1. Goal and scope

The platform wide secrets of ai, billing, email, message, pipecat, transcribe, tts and number are read from environment variables, and only from there (no database fallback). Customer owned credentials (the engine base key and the MCP server secrets and OAuth tokens of bin-ai-manager) and their master keys are not touched.

In scope: code that reads the secrets, `komodo/docker-compose.yml` and `k8s/deployment.yml` variable lines, the Python runner of bin-pipecat-manager, service docs, the compose tests, the RST self hosting pages that name the variables.

Out of scope: removal of the stored settings code, the admin API, OpenAPI of `/service_configs`, the table drop, square-admin, `PADDLE_PRICE_ID_*`, `LOCAL_LLM_MODELS`, `GOOGLE_APPLICATION_CREDENTIALS_JSON`, and the creation of the deployment variables.

## 2. Variable names

One variable per external service key, shared by every service that uses it.

| Service key | Variable | Services |
|---|---|---|
| openai key | `OPENAI_API_KEY` | ai (was `ENGINE_KEY_CHATGPT`), pipecat |
| google key | `GOOGLE_API_KEY` | ai, pipecat |
| xai, openrouter, deepgram, cartesia | `XAI_API_KEY`, `OPENROUTER_API_KEY`, `DEEPGRAM_API_KEY`, `CARTESIA_API_KEY` | pipecat |
| elevenlabs | `ELEVENLABS_API_KEY` | tts, pipecat |
| local llm endpoints | `LOCAL_LLM_ENDPOINTS` | pipecat |
| aws pair | `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` | tts, transcribe |
| sendgrid, mailgun | `SENDGRID_API_KEY`, `MAILGUN_API_KEY`, `MAILGUN_WEBHOOK_SIGNING_KEY` | email |
| messagebird | `MESSAGEBIRD_API_KEY` (was `AUTHTOKEN_MESSAGEBIRD`) | message |
| telnyx | `TELNYX_API_KEY` (was `AUTHTOKEN_TELNYX` and `TELNYX_TOKEN`), `TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID` | message (key), number |
| twilio | `TWILIO_ACCOUNT_SID` (was `TWILIO_SID`), `TWILIO_AUTH_TOKEN` (was `TWILIO_TOKEN`) | number |
| paddle | `PADDLE_API_KEY`, `PADDLE_WEBHOOK_SECRET` (was `PADDLE_WEBHOOK_SECRET_KEY`) | billing |
| MCP OAuth app | `<V>_MCP_OAUTH_CLIENT_ID`, `<V>_MCP_OAUTH_CLIENT_SECRET`, V in GITHUB, LINEAR, ASANA, HUBSPOT | ai |

The compose value is `[[BIN_MANAGER__<NAME>]]` with the same `<NAME>` as the variable; k8s uses a `secretKeyRef` of the secret `voipbin` whose key is the variable name, always with `optional: true` (a missing variable must not stop the pod; the service then treats the key as not configured). bin-hook-manager does not verify the Paddle signature (bin-billing-manager does), so it gets no variable.

Relation to the lines that existed before the stored settings (from `git show a5ac068ae 2ec2fac22 1a738c8c9 41517e3b0`):

| Service | Variable now | Line before the stored settings | Kind |
|---|---|---|---|
| ai | `OPENAI_API_KEY` | `ENGINE_KEY_CHATGPT` (k8s secret key `OPENAI_API_KEY`) | renamed |
| ai | `GOOGLE_API_KEY` | same | restored |
| ai | `<V>_MCP_OAUTH_*` (8, compose only) | `MCP_OAUTH_<V>_CLIENT_*` | renamed |
| email | the three variables | same names; the signing key had the deployment variable `EMAIL_MANAGER_MAILGUN_WEBHOOK_SIGNING_KEY` and no k8s line | restored, signing key renamed |
| message | `MESSAGEBIRD_API_KEY`, `TELNYX_API_KEY` | `AUTHTOKEN_MESSAGEBIRD`, `AUTHTOKEN_TELNYX` (k8s secret key `TELNYX_TOKEN`) | renamed |
| number | `TELNYX_API_KEY`, the two Telnyx ids, `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` | `TELNYX_TOKEN`, `TWILIO_SID`, `TWILIO_TOKEN` renamed; the two Telnyx ids same | renamed, restored |
| transcribe, tts | `AWS_*`, `ELEVENLABS_API_KEY` | same | restored |
| pipecat | the eight runner variables | same (runner containers of both stacks) | restored |
| billing | `PADDLE_API_KEY`, `PADDLE_WEBHOOK_SECRET` | compose never had a Paddle line; k8s had `PADDLE_API_KEY`; the webhook secret was a variable of bin-hook-manager (`PADDLE_WEBHOOK_SECRET_KEY`) | new in compose, renamed |

Every name that is new or renamed needs a deployment variable with the same name before this change is deployed. Equal values behind a shared name (for example the ai and the pipecat OpenAI key, the message and the number Telnyx key) are a precondition of the shared name that is checked outside the code.

## 3. Design decisions

D1. Where the environment is read. A new shared package `bin-common-handler/pkg/envsecret` (used by the 7 Go services ai, billing, email, message, number, transcribe and tts, so it meets the admission rule) with `Lookup(name) (value string, state State)` only. Each service keeps its supplier functions (`KeyFunc`, `TokenFunc`, `AWSCredentialsFunc`, ...), so no handler signature below the supplier changes. Only the supplier constructors lose the `confighandler.Handler` argument and read the environment at the time of use (nothing is cached, so a replica restart is the only way a value changes, as before the stored settings). `confighandler` itself is not changed: it stays the stored settings code until its removal ticket, and no read path consumes it.

D2. Declaration. Each `internal/configkeys` declares the variable name of every key as a constant (`Env...`). `Key.Env` is not set and `NewHandler` still receives a nil lookup: a handler with a lookup would resolve the database first, and the admin listing must not suggest that the handler reads the environment. The constants are the single list of names that the suppliers and the start check use.

D2a. Stored settings next to the environment. The stored settings code and the admin API stay until the removal ticket and are not read. To keep an operator from rotating a key in the wrong place, `<svc>-control config set` and `clear` of the affected keys, and `number-control provider secret set` and `delete`, print one line to stderr that names the variable that is read and says that the stored value is not used. The docs say the same. The OpenAPI text of the number provider secret endpoints (which says that there is no environment credential) is updated to match; the PUT and DELETE of the secret keep working on the stored row only. A stored value that is left behind is stale after an environment rotation; an image rollback therefore reads the last stored value (see D8).

D3. Garbage and empty values. `Lookup` trims surrounding whitespace and reports one of three states: unset or empty, set (with the value), or placeholder (a value that starts with `[[`, the text of an unresolved deployment variable). Unset, empty and placeholder all mean "not configured" for a use: the supplier returns an empty value and no error, so the placeholder is never sent to a provider and an optional vendor that is not used costs nothing (an environment read cannot fail, so no supplier returns `ErrKeyUnavailable` for a read; the half AWS pair of tts (D4) and the incomplete MCP OAuth pair of ai (D5) keep their errors). At start, each Go service except pipecat logs one line per variable from its own start check (`<svc>config.LogStates`, which replaces `LogSources`): set (info), not set (info or warn, by the rule each service used for its stored keys), placeholder (warn, with the variable name and the words "unresolved placeholder"). Nothing is logged per call, so a placeholder cannot flood the log. The start never stops. No value or prefix of a value is logged. The placeholder form is documented in the repository (the missing deployment variable note of the compose docs).

D3b. message keeps its behavior for an empty token: `requestexternal` sends the request with the empty credential as it did when the key was not stored, and the provider refuses it; no new guard is added in this change.

D3a. The pipecat Go service reads no key, so it has no start check and its call of `LogSources` (a database read) is removed. The runner logs the state of its variables once at start (a function of `run.py` called from `main.py` after `load_dotenv`, so it logs the final values: set, not set, placeholder, never a value; the image has no `.env` file, which is verified at implementation), because the variables live in the runner container. The pipecat `configkeys` get no `Env` constants; the names are in `run.py` only.

D4. AWS pair (tts, transcribe). Both variables set: the pair. Neither or a placeholder: not configured. Exactly one set: refused as each service refused a half pair before (`ErrAWSPairIncomplete` in tts, empty values in transcribe), logged once at start at warn level without values (a call returns the error only).

D5. ai. `KeyForBaseURL`, `GeminiKey` read `OPENAI_API_KEY` and `GOOGLE_API_KEY`. `MCPOAuthClient` reads the pair of one vendor: client secret set and client id empty is `ErrMCPOAuthIncomplete`; both empty is the empty pair (not configured); client id alone is the empty pair `("", "")` (a test pins the returned values). The variable names are derived from the vendor in `configkeys` (no vendor string elsewhere).

D6. pipecat. The runner reads the environment of its own container: `_platform_key(platform_keys, name, vendor, env_name)` returns the key of the request when it has one (kept so that the contract of the request does not change), else the variable `env_name` (trimmed; a placeholder or an empty value is rejected with `<vendor> is not configured` before the SDK object is built, as now). `LOCAL_LLM_ENDPOINTS` is read from the environment for each call when the request has none (no module state). The Go service no longer wires `WithPlatformKeyResolver` in `cmd/pipecat-manager` and `cmd/pipecat-control`, so no key is sent in the request and no key is held by the Go containers (one place per key, nothing in the request body). `pipecatconfig.NewKeyResolver`, `pipecatconfig.LogSources` and the request key path of the runner (`platform_keys`) stay until the removal ticket, which removes them together; the request key path is also what keeps a new runner working next to an old Go service during a rolling update. The runner still never reads the database. Compose: the eight runner lines in both stacks; the Go containers get none.

D7. number. The `number_providers` row keeps `enabled`, `priority` and the non secret `config` (`connection_id`, `profile_id`, `sid`, `trunk_sid`). The secret comes from the environment only: `TELNYX_API_KEY` for telnyx, `TWILIO_AUTH_TOKEN` for twilio; a stored ciphertext is not decrypted and not used. The three config variables (`TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID`, `TWILIO_ACCOUNT_SID`) fill a config key that the row does not have (or has empty): the explicit config of the row wins, so a deployment without any row behaves as before the stored settings. `trunk_sid` has no variable. An empty or placeholder secret variable gives the source `none` (not usable, release and search too, as with any missing secret); an empty or placeholder config variable is treated as not set, so a required config that is then missing makes the provider unusable for the purchase only, as now. `has_secret` and `credential_source` are defined by the effective secret (`toInfo` uses `r.hasSecret` instead of `hasDBSecret(r.row)`; `UpdateSecret` and `DeleteSecret` return the Info of the effective state, so `has_secret` can stay true after a stored secret is deleted when the variable is set, which the schema text says): `env` with `has_secret` true when the variable holds a value, else `none` with false; the stored ciphertext does not count. `Info.Config` shows the config of the row only: showing the filled values would let a read and write back (`mergeConfig`) freeze a variable value into the row, after which the row would win over a changed variable. The variable fill is applied only when the credentials are resolved for a use, so `config` is the row value and the value in use is the row value or, when the row has none, the variable (a provider that is usable only through the variables shows an empty `config`; the schema text and the operations doc say so, and the docs name the check: a row value left from before hides a changed variable). The admin write checks (`UpdateSecret`, `mergeConfig`) keep checking the config of the row and do not count the variable fill. `CredentialSource` gets `env` back and the enum value is added back to the response schema of the provider API; `db` and `db_undecryptable` are no longer produced (the ciphertext is not read) and stay in the response-only enum so that older readers keep working, and the schema text says they are not produced. `key_version` and `secret_updated_by` keep describing the stored row (legacy, not used); the schema text says so, and `has_secret` is described as the presence of the effective secret.

D8. Rolling deploy and rollback. For pipecat the runner containers must carry the variables before (or in the same deployment as) the Go service that stops sending keys; a new runner with an old Go service is fine, because the request key still wins. The old image reads the stored settings and the new one the environment, and both run during a rolling update, so a key has to hold the same value in both places until the update ends. A stored value that is not rotated together with the variable stays stale; an image rollback uses it. Rollback is the previous image together with the previous deployment definition; the new variables stay unused.

Merge gate (checked outside the code): every new or renamed variable of section 2 exists in the deployment variable store, and the values of shared names are equal. For billing both `PADDLE_API_KEY` and `PADDLE_WEBHOOK_SECRET` are mandatory: without the webhook secret every Paddle webhook is rejected (fail closed), which loses payment events, and without the API key no portal session can be created. Without a variable the compose value stays the placeholder and the service refuses the key (D3); in k8s the optional key is absent and the key is not configured.

## 4. File level changes

1. `bin-common-handler/pkg/envsecret` (new): `Lookup`, `State`, tests; the row of `bin-common-handler/CLAUDE.md`.
2. For each of ai, billing, email, message, transcribe, tts: `internal/configkeys` (env name constants), `internal/<svc>config` (suppliers read `envsecret` and lose the handler argument, `LogSources` becomes `LogStates`), `cmd/<svc>-manager`, `cmd/<svc>-control` (supplier calls, the stderr line of D2a), tests, `docs/operations.md`, `CLAUDE.md`. ai also `cmd/ai-manager/mcp_oauth_credentials.go` and `aiconfig.MCPOAuthClient`.
3. `bin-pipecat-manager`: `cmd/pipecat-manager` and `cmd/pipecat-control` wiring (resolver and `LogSources` removed), `scripts/pipecat/run.py` (environment source, start log of the variables) and its tests, compose (runner lines, two stacks; the Go containers get none).
4. `bin-number-manager`: `pkg/providerhandler` (`resolve.go`, `main.go`, `admin.go` views), `models/provider` (`CredentialSourceEnv`), `cmd/number-manager` start log, `cmd/number-control/provider.go` and tests, docs. OpenAPI: the `credential_source` enum of the provider schemas in `bin-openapi-manager`, the generated code of `bin-openapi-manager` and `bin-api-manager`, and the RST pages that describe `credential_source`.
5. compose and k8s lines (section 2) of ai, billing, email, message, number, transcribe, tts; compose lines of pipecat (it has no k8s directory).
6. Comments and docs that say that the database is the only source (`bin-ai-manager/cmd/ai-manager/builder_wiring.go` hint text, `run.py` comments, the pipecat compose comment about the stored keys, the CLAUDE.md and `docs/operations.md` of ai, billing, email, message, number, pipecat, transcribe, tts) are rewritten in the same change.
7. RST pages that name the variables (`self_hosting_envvars`, `self_hosting_providers`, `service_config_*`, number provider pages) and the rebuilt HTML of the edited pages.

## 5. Verification plan

- Unit tests per supplier: set, empty, placeholder, whitespace, half AWS pair, MCP pair cases; the stored settings store is never touched (a store that fails the test when called).
- Compose and k8s tests: only `bin-pipecat-manager` has a compose assertion to reverse (`Test_komodoComposeHasNoPlatformKeyVariables`); for pipecat the new test pins that the variables are on the runner services and not on the manager services. For ai, billing, email, message, number, transcribe, tts a new test pins that each variable of section 2 is present in compose with the same `BIN_MANAGER__` name, and that each k8s variable has `optional: true`.
- Existing assertions and texts that say the environment is not a source are reversed (or rewritten to the new names) and listed in the RED run; besides the files named below this covers the `internal/config` tests that say that the old variables are ignored (email `config_test.go`, message `masterkey_test.go`, ai `providerkeys_test.go`), `bin-ai-manager/cmd/ai-manager/platform_keys_test.go` and `builder_wiring_test.go`, `messageconfig/main_test.go` (the `keys[0].Env` and the `TELNYX_TOKEN` assertions), the billing tests that name `PADDLE_WEBHOOK_SECRET_KEY` (`config_test.go`, `main_test.go`), and the existing master key compose tests of billing, message and transcribe (extended, not reversed). A regression assertion pins that `Key.Env` stays empty and that the admin listing (`bin-api-manager` `service_configs_test.go`) shows no `env_name` or `env_shadowed`. The list: `ai-control` `config_test.go` and the help text of `config.go`, `cmd/ai-manager/mcp_oauth_credentials_test.go`, `number-control` `provider_test.go`, the `*config` `main_test.go` of each service, the pipecat `run.py` comments and its Python tests (`test_platform_keys.py`, `test_local_llm.py`, `test_run.py`, `test_init_pipeline.py`, `test_upgrade_112.py`). Each new or reversed test is run against the current main files first (RED) and then after the change.
- `envsecret` tests: unset, empty, spaces, `[[X]]` (state Placeholder, empty value), a value that only contains `[[` in the middle (accepted), no value in any error text.
- pipecat Python: request key wins, variable used when the request has none, placeholder rejected, local endpoints from the environment per call, the customer key still wins for openai, grok and gemini and never looks up the platform variable.
- number: `Info.Config` is the row config only, a read and write back does not store a variable value; row config plus env secret, row config wins over the variable, variable fills a missing config key, twilio `trunk_sid` from the row, stored ciphertext ignored, `env` source, `none` for an empty or placeholder secret, `has_secret` follows the effective secret, placeholder config variable treated as not set.
- Full verification workflow in each changed service and in `bin-common-handler`, `bin-openapi-manager` and `bin-api-manager`; repository gate scripts (`scripts/check-*.sh`).
- Not verifiable here: a deployment with real variables and real vendors (no vendor call is made).

## 6. Impact analysis

- Destructive operations: none. No schema or migration change, no table or column is touched.
- Rolling deploy: see D8. A new image without its variables treats the keys as not configured (the calls that need them fail with the service error). In compose the container starts with the placeholder, in k8s the optional key is absent; neither crash loops.
- Stored settings code and admin API remain: a write through them is accepted and has no effect on reads until the removal; the CLI line and the docs say so (D2a).

## 7. Detailed design (signatures and flow)

`envsecret`:

```go
type State int // Unset, Set, Placeholder
// Lookup returns the trimmed value and its state. The value is empty unless the state is Set.
func Lookup(name string) (string, State)
```

Suppliers keep their function types (`KeyFunc`, `TokenFunc`, `emailhandler.SecretFunc`, `AWSCredentialsFunc`, `AWSKeysFunc`, `paddlehandler.KeyFunc`) and their service error: an environment read cannot fail, so the suppliers return an empty value and no error for a key that is not configured (the half AWS pair of tts and the incomplete MCP OAuth pair of ai are the two exceptions) (the error types stay for the stored settings code until its removal). The `confighandler.Handler` argument and `SupplierOption`/`WithTimeout` (an environment read does not block) are removed from the constructors; callers: `ttsconfig.ElevenlabsKey()`, `ttsconfig.AWSCredentials()`, `transcribeconfig.AWSKeys()`, `billingconfig.PaddleKeys()`, `emailconfig.Supplier(envName)`, `messageconfig.MessagebirdToken()`, `TelnyxToken()`, `aiconfig.KeyForBaseURL(baseURL)`, `GeminiKey()`, `Supplier(envName)`, `MCPOAuthClient(vendor)`. `New(cfg, store, opts...)` of each `<svc>config` stays for the `<svc>-control config` commands and the admin API.

AWS pair: tts returns `ErrKeyUnavailable` wrapping `ErrAWSPairIncomplete` (the renamed `ErrAWSSourceMismatch`, with the test of `ttsconfig/main_test.go`) when exactly one variable is set; transcribe keeps its existing handling of an unusable pair (empty values, no error, logged), now by the same condition. Both log the half pair once at start (`LogStates`).

ai MCP OAuth: `configkeys` gets `MCPOAuthEnvNames(vendor) (idName, secretName string, ok bool)` (`<V>_MCP_OAUTH_CLIENT_ID`, `<V>_MCP_OAUTH_CLIENT_SECRET`, `<V>` is the upper case vendor). `MCPOAuthClient` reads the secret first: a value means the id is required (`ErrMCPOAuthIncomplete` when empty); no secret means the empty pair. The existing `CredentialsFunc` adapter in `cmd/ai-manager/mcp_oauth_credentials.go` calls it without the handler.

number resolve (replaces the body below the row read of `resolve.go`). `Descriptor` gets `EnvSecret string` (`TELNYX_API_KEY`, `TWILIO_AUTH_TOKEN`) and `EnvConfig map[string]string` (config key to variable: `connection_id` to `TELNYX_CONNECTION_ID`, `profile_id` to `TELNYX_PROFILE_ID`, `sid` to `TWILIO_ACCOUNT_SID`):

1. `envsecret.Lookup(desc.EnvSecret)`; only the state Set counts as a secret (a placeholder is the same as unset; nothing is logged per resolve).
2. No secret: `source=none`, `hasSecret=false`, `usable=false`, `observeUnusable("no_secret")`, as now.
3. Secret present: `source=env`, `hasSecret=true`; `creds.Config` is a new map (the row, which can be nil, is never written): the row config copied, and for each `desc.EnvConfig` entry whose row value is empty, the variable value when its state is Set. Twilio `sid` is trimmed as before.
4. `requiredConfigMissing` on the filled config gives `required_config` unusable, else `usable = enabled`.
`hasDBSecret`, `decryptSecret` and the `db`, `db_undecryptable` branches are no longer reached by `resolve`; the admin write paths keep using them for the stored row, including the rule of `mergeConfig` that a row with a stored secret keeps its required config keys (unchanged, documented).

Start check of number: `logPurchaseReadiness` text changes from "store a provider secret with the provider admin API" to the variable names; the state of the variables is logged once at start by `providerhandler.LogStates()` (set, not set, placeholder).

pipecat runner: `_platform_key(platform_keys, name, vendor, env_name)`: request value, else `_env_value(env_name)` (strip, `[[` prefix rejected, empty rejected) or `ValueError(f"{vendor} is not configured")`. `_local_endpoints_for(platform_keys)`: request value, else the environment value `LOCAL_LLM_ENDPOINTS` (same strip and placeholder rule); parsed per call. `log_env_state()` is defined in `run.py` and called from `main.py` after `load_dotenv(override=True)`, so it logs the final values.

compose/k8s: exact lines per section 2; k8s entries are `valueFrom.secretKeyRef {name: voipbin, key: <VARIABLE>, optional: true}` placed where the lines were removed. The compose and k8s tests parse the YAML (not grep) and check name, value and `optional`.

Test order (RED first): `envsecret` tests, supplier tests of each service with a store that fails when touched, number resolve tests (including a missing row), compose/k8s tests, python tests; every new or reversed test is run before its change and its failing output is kept as an evidence file outside the repository.

Caller changes that the signatures cause: `newPlatformKeys(cfg)` and `mcpOAuthCredentials()` in `bin-ai-manager/cmd/ai-manager` (no handler), `bin-email-manager/cmd/email-control/main.go`, `bin-message-manager/cmd/message-control/main.go`, `bin-tts-manager/cmd/tts-control/main.go`, `bin-transcribe-manager/cmd/transcribe-control/main.go`. Tests deleted with the removed `WithTimeout` option: the cases in `aiconfig/main_test.go` and `emailconfig/main_test.go`.
