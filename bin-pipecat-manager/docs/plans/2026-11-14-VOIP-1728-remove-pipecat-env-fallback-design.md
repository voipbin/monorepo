# VOIP-1728 Remove the environment fallback of the pipecat platform keys

## 1. Ticket review (against origin/main 41517e3b0)

Valid. The platform keys of `bin-pipecat-manager` still fall back to the environment in two layers:

- Go: `internal/pipecatconfig/main.go` passes `os.LookupEnv` to `confighandler.NewHandler`, and every key in `internal/configkeys/main.go` declares an `Env`.
- Runner: `scripts/pipecat/run.py` `_platform_key` returns `os.getenv(env_name)` when the request has no value, and `load_local_endpoints()` reads `LOCAL_LLM_ENDPOINTS` from the environment at import time.

No merged or open PR removes it. The stored keys were added by VOIP-1723 and the billing fallback was removed by VOIP-1726. Consumers found by grep beyond the first analysis: log texts in `pipecatconfig`, comments in `pkg/pipecatcallhandler/platformkeys.go` and `pkg/pipecatcallhandler/main.go` (WithPlatformKeyResolver), `cmd/pipecat-control/main.go:74`, `cmd/pipecat-manager/main.go:153`, `internal/config/config.go:37`, `pkg/listenhandler/v1_configs_test.go:114` (the assertion `k.EnvName == ""` is inverted to expect an empty `env_name`; asserts `EnvName != ""` for every key; the key list used there is checked in the implementation), `bin-tts-manager/docs/operations.md` (110, 153) and `bin-tts-manager/CLAUDE.md:43` (say pipecat reads `ELEVENLABS_API_KEY` from its own environment), the `pkg/confighandler` row of `bin-common-handler/CLAUDE.md`, `scripts/pipecat_realtest/*` (set dummy env values and call the factories without `platform_keys`), and the Go tests listed below.

Other services: `bin-ai-manager` reads its Gemini key from its own database store (`aiconfig`), ai-manager no longer reads `GOOGLE_API_KEY` (the variable remains only as an unused declaration in its own stack and k8s files, outside this card; after this PR it has no consumer at all, which is a one line cleanup proposal for the summary). The shared `confighandler` keeps its env source for the ai-manager MCP OAuth keys. The public API field `env_name` is `omitempty` (`configrpc/server.go`, `res.EnvName = k.Env`), so it disappears from the pipecat keys only; the OpenAPI schema is unchanged. The self hosting installer lives outside this repository and is a separate card; the RST that describes it is corrected here only where it states the runner reads the environment.

Scope reduction by decision of the maintainer: no existing installation exists, so no upgrade or migration code is written. Only the fallback code is removed.

## 2. Design

### Behavior after the change

- Go: the handler gets a nil environment lookup (same as billing), and no key declares `Env`. A key without a stored row has source `none`, even when a variable of the old name is set in the process. `Resolve` is otherwise unchanged (a key that can not be read is left out of the request with one warning).
- Runner: the platform key comes from the request `platform_keys` only. `_platform_key(platform_keys, name, vendor)` returns the value, or raises `ValueError("<vendor> is not configured")`. It never returns `None` or `""`, because the SDK clients fall back to the process environment when they get `None`. The error is raised before the SDK object is built. The key of the customer still wins for openai, grok and gemini: `key or _platform_key(...)` does not evaluate the platform key when the customer key is present.
- Vendors: openai, xai, google (Gemini LLM), cartesia, elevenlabs, deepgram, platform openrouter. The vendor argument for openrouter is `OpenRouter`, so the existing text `OpenRouter is not configured` stays, and the `or ""` and `if not api_key: raise` block of the `platform_openrouter` branch is deleted as dead code.
- Local endpoints: `load_local_endpoints()` and the import-time call are removed. `_local_endpoints_for` parses the request value only; a missing value gives an empty registry, so the existing fixed error `local engine model is not available` applies. The module registry variable is removed with its only writer.
- Google STT and TTS use the service account (Application Default Credentials), not an API key, and are not subject to the not-configured rule.
- Out of scope and unchanged: `GOOGLE_APPLICATION_CREDENTIALS` (service account file path), `LOCAL_LLM_MODELS`, `LOG_LEVEL`, `POD_IP`, the shared `confighandler` env source (ai-manager uses it), the common API enumeration of sources.

### What is lost

- `pipecat-control config clear` now leaves a key without any value: the vendor sessions fail with "not configured" (the help text no longer says the variable applies again).
- The recovery path of the fallback: a database read that is out of the 2 second bound or a decrypt error now ends the session (for sessions that need the platform key: openai, grok and gemini only without a customer key, platform openrouter, cartesia, elevenlabs, deepgram, local) with "<vendor> is not configured" (a local endpoint ends with the fixed `local engine model is not available`). What the caller sees is unchanged in kind: the runner answers 400 with the text, the Go side keeps only `non-200 status` (and the body length when the request has no endpoint, `pythonrunner.go:160-167`), logs and cancels the session, so the user sees the session end without a classified error. The vendor name is visible in the runner log only. No classification is added (out of scope).
- `LOCAL_LLM_MODELS` stays an ai-manager and pipecat setting while the endpoints move to the database: a model can be listed while its endpoint is not stored. The enable order in `bin-ai-manager/docs/operations.md` changes to "store the endpoints first".
- The runner can no longer be started alone from environment variables (local development, self hosting): keys must be stored. Deployment order: the values must be stored in `pipecat_configs` before the new image runs (a separate DB card checks this). The previous image keeps its fallback, so rolling back the image works only while the environment lines of the old stack definition are still present. The PR removes the lines from the stack definition file, but does not delete any Komodo variable.

### Files

- Go: `internal/pipecatconfig/main.go` (nil lookup, header and function comments, log texts, remove `os` import), `internal/configkeys/main.go` (remove `Env` of 8 keys, comments), `pkg/pipecatcallhandler/platformkeys.go` (comment), `internal/config/config.go` (comment), `cmd/pipecat-control/config.go` (help texts).
- Go tests, in detail: `internal/pipecatconfig/main_test.go` (lines using `k.Env`, `SourceEnv`, the 8 source count, the Secret mask case that used an environment value now uses a stored row, the no-master-key case keeps its meaning with a stored row), `internal/configkeys/main_test.go` (golden compares `Env`; now asserts empty), `cmd/pipecat-control/config_test.go` (69-101, 234-250), `main_help_test.go` checked for help text assertions. Summary: `pipecatconfig/main_test.go` (env set, source is `none`, no value in the request; replaces the env source cases), `configkeys/main_test.go` (no `Env`), `cmd/pipecat-control/config_test.go` (env set, list shows `none`), `cmd/pipecat-manager/komodo_compose_test.go` (existing file: add a test that reuses `composeServiceBlocks` and asserts, by variable name per line, that none of the 8 key variables is in any of the 4 service blocks while `LOCAL_LLM_MODELS` stays in the two manager blocks and `GOOGLE_APPLICATION_CREDENTIALS` stays in the two runner blocks; `render-image-tag.sh` and `komodo-api-deploy.sh` are read in the implementation to confirm they do not require the removed variables).
- Python: `run.py` (also the log text `LOCAL_LLM_ENDPOINTS is not a valid json object` and the comment block at 680-684 become request value based); `scripts/pipecat_realtest/*` (remove the dummy env loop, pass `platform_keys` to the factories). Local endpoint tests: `load_local_endpoints` and `run._local_endpoints` are removed, so tests that use them as helpers (`test_local_llm.py`, `test_platform_keys.py` including the registry save and restore and the "module registry unchanged" assertions) are rewritten to pass `platform_keys["local_llm.endpoints"]` to `create_llm_service` and to use `parse_local_endpoints`; the environment reading case is replaced by an ignore case. Tests `test_run.py`, `test_platform_keys.py`, `test_local_llm.py`, `test_init_pipeline.py` change to: the environment value is ignored, a missing request key is rejected before the SDK class is called, and each vendor is covered.
- Deployment: `komodo/docker-compose.yml` (both stacks: 5 Go variables, 8 runner variables, the VOIP-1482 comment is shortened to the service account part), `komodo/environment.env` unchanged except check.
- Docs: `bin-pipecat-manager/docs/{operations,dependencies,domain,architecture}.md`, `bin-pipecat-manager/CLAUDE.md`, `bin-ai-manager/docs/operations.md` (31, 93, 102, 122, 159, 447), `bin-tts-manager/docs/operations.md` (110, 153), `bin-tts-manager/CLAUDE.md` (43), `bin-ai-manager/CLAUDE.md`, `bin-common-handler/CLAUDE.md`, and the RST files `service_config_overview.rst`, `service_config_struct_service_config.rst`, `self_hosting_envvars.rst`, `self_hosting_providers.rst` with a clean rebuild of `docsdev/build`. Dated files under `docs/plans/` are history and are not changed. `docs/domain.md` and `docs/architecture.md`: architecture.md lines 46 and 80 say database first, then the environment and are changed; domain.md is checked by grep in the implementation and left as is if it has no such statement. `test_custom_endpoint_net.py` and `custom_endpoint_net.py` assert that the environment has no effect on customer endpoints and are not changed. The key rotation restart notes in `self_hosting_providers.rst` (about 87, 91, 261) are rewritten, because a stored key is read by the next session and needs no restart, and the `service_config_overview.rst` lines (27, 37, 47) drop `pipecat` from the environment source statements: after this change only the `mcp_oauth` keys of ai declare an environment name (tts-manager already passes no lookup). The whole of `self_hosting_providers.rst` is re-grepped for the variable names (54-61, 81, 87, 91, 107, 120-125, 135, 142, 178, 261-264).

### Verification (replaces the validator test, which can not see key origin)

1. Go: with the process variable set, source is `none`, `Resolve` returns nothing for the key, and no env name is declared (loop over all keys).
2. Python: per vendor, a process variable set and no request key raises "not configured" and the SDK class is not called; a request key is passed to the SDK; the customer key beats the platform key and the platform key is not looked up.
3. Local: a process `LOCAL_LLM_ENDPOINTS` is ignored by a `local.` model. Import time reading is proven in a subprocess that sets the variable, imports `run`, and expects the local model to be rejected with the fixed message (reverting the import call turns it red).
   Customer key short circuit: with a customer key, `_platform_key` patched to raise is not called.
   SDK not called: the SDK classes are patched at module level and `assert_not_called` is used.
4. Commands: Go: the full verification workflow of the root CLAUDE.md in `bin-pipecat-manager`. Python (CI does not run it, so the local output is the PR evidence): from `scripts/pipecat`, `UV_PROJECT_ENVIRONMENT=<scratch venv> uv sync --frozen` once, then `uv run --frozen --with pytest --with pytest-asyncio pytest -q` (baseline before the change: 697 passed). Gates: `bash scripts/check-test-conventions.sh`, `bash scripts/check-docs.sh`, and `scripts/check-service-docs.sh` as the hook runs it.
   The subprocess proof of the import time rule runs `python -c "import conftest, run"` with `cwd=scripts/pipecat` and an `env=` that holds only the test variable (the conftest mocks the pipecat modules, which a bare import does not have).
   Residual grep, expected 0 outside `docs/plans/`: `own environment`, `runner may still`, `environment variable applies again`, `else the environment`, `wins over the environment`, `or an environment variable`, `os.LookupEnv` in `internal/pipecatconfig`.

Verification 5. Docs: after the clean rebuild, `git diff --stat` of `docsdev/build` is checked for unrelated bulk changes (builder version drift).
Verification 6. Rollback note: rolling back the image also means restoring the removed environment lines of the stack definition (reverting this PR), because the old image still falls back to them.

### Implementation order (tests first)

1. Go: write the failing tests (source none with the variable set, no `Env` declared, compose has no vendor variables, listenhandler test), see RED, then change code, then GREEN.
2. Python: write the failing tests per vendor, see RED, change `run.py`, GREEN; update the older tests (a case that expected the environment becomes an ignore case; a case that expected a default registry from the environment is deleted).
3. Compose files, then docs, then the docsdev clean rebuild.

### Mutation table (each must turn a test red)

| Mutation | Expected red test |
|---|---|
| `_platform_key` returns `os.getenv(...)` again | per vendor environment ignored test |
| `_platform_key` returns `""` or `None` | per vendor not-called and `ValueError` tests |
| `key or _platform_key(...)` becomes `_platform_key(...) or key` | customer key short circuit test (patched `_platform_key` raises) |
| `load_local_endpoints` or its import time call restored | import time subprocess test |
| `Env` restored on one key | all keys have empty `Env` test (the source none test stays green, because the handler has no lookup; it turns red only together with `os.LookupEnv` restored) |
| one vendor variable restored in a stack block | per block absence test (by variable name, not by value) |
| `os.LookupEnv` restored with empty `Env` | not red, and harmless (no lookup happens without a declared name) |

### Python test migration

`test_local_llm.py` (18-20, 24, 33, 50, 56, 103, 141) and `test_platform_keys.py` (34-36, 181-210, 235, 245-246, 380): the helper use of `load_local_endpoints` moves to `platform_keys` in `create_llm_service` or to `parse_local_endpoints`; the log and error secret leak checks (56, 245-246) move to `parse_local_endpoints` and are kept. `test_run.py` environment uses (70, 113, 874, 964, 998, 1018, 1040, 1052-1057, 1071, 1085, 1110-1119, 1197, 1206): the cases that expected the environment key are turned into environment ignored cases or get the key in `platform_keys`; the BYOK and customer endpoint cases that already prove the environment is unused are kept.
