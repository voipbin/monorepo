# VOIP-1723: stored platform keys for bin-pipecat-manager

Short design note. The behavior is documented in `docs/operations.md` (Stored platform keys).

## Goal

Let an operator store the platform API keys of the vendors that the Python runner talks to encrypted in the database, and change them without a restart, with the environment variables staying the fallback.

## Decisions

1. Eight keys in `pipecat_configs` (all secrets): `openai.api_key`, `deepgram.api_key`, `cartesia.api_key`, `xai.api_key`, `elevenlabs.api_key`, `google.api_key`, `openrouter.api_key`, `local_llm.endpoints`. Each declares the environment variable that the runner reads today.
2. The Go service resolves the keys when a session starts and sends them to the runner as `platform_keys` in the start request. The runner has no database access and no master key. The key names are the wire contract.
3. Resolution order in Go: database, then `os.LookupEnv` of the Go process. The runner then uses `platform_keys`, then its own environment. For OpenAI, Grok and Gemini the key of the customer comes first. Platform OpenRouter and local never use a customer key.
4. `runnerStartScript` calls one helper (`sessionPlatformKeys`) for a single AI and for a team, so the same configuration gets the same map. Only the vendors the session uses are resolved; a team uses the union of its members (the members' STT and TTS count only when the call itself has them, as in the runner).
5. A key that can not be read (database error, undecryptable value, invalid value, out of time) is left out and one warning is logged (key name, pipecatcall id, reason class; never a value or the error text). Reads are bounded to 2 seconds for the whole session. The runner falls back to its own environment for a missing key. Nothing is cached, so the next session sees a change.
6. Secrets are `amai.RedactedString` in Go (plain only in the JSON body) and `Field(repr=False)` in Python. The runner's 422 answer is already a fixed text without the input.
7. The master key list `PIPECAT_CONFIG_ENCRYPTION_KEYS` is mapped to the two Go containers only. A compose test pins that the runner containers do not get it.
8. The local endpoints text of a request replaces the runner's environment registry for that session only; the module registry is never changed, so concurrent sessions do not affect each other.
9. Management: `pipecat-control config list|set|clear` and the admin API (`/service_configs/pipecat`) through `/v1/configs` on the existing request queue.

## Schema

Two new tables, `CREATE TABLE IF NOT EXISTS`, one statement per file (`000002_pipecat_configs_create_table_all`, `000003_pipecat_config_histories_create_table_all`). CHECK constraints are named `chk_pipecat_configs_*` and `chk_pipecat_config_histories_*`. The down files drop the new tables and carry `-- DOWN-LOSSY`. No existing table changes.

## Known limits

- `config list` shows the source as seen by the Go process. Three variables (`GOOGLE_API_KEY`, `OPENROUTER_API_KEY`, `LOCAL_LLM_ENDPOINTS`) are only in the runner containers, so they show `none` in the list.
- Keys that other services also hold (OpenAI, Google, ElevenLabs) are stored separately; rotating one means updating both places.
- Exception text of the vendor SDKs in the runner's 400 and 500 answers of a non-custom session is not filtered further; the platform key is not part of the fixed texts the runner builds.
