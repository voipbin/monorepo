# VOIP-1723: DB secret stores for billing, pipecat and the MCP OAuth keys of ai

Summary note of the design. The code is the source of truth.

## Purpose

Services that read provider secrets only from environment variables get the same
stored-setting mechanism as email, message, tts and transcribe: secrets are kept
encrypted in the service database (`<svc>_configs`, `<svc>_config_histories`),
managed through the Service Config API (`/service_configs/{service}`) and the
`<svc>-control config` commands. This step adds the store and the read path and
keeps the environment variable as the fallback. Removing the fallback is a later step.

## Keys

| Service | Keys | Environment name |
|---|---|---|
| billing | `paddle.api_key`, `paddle.webhook_secret_key` | `PADDLE_API_KEY`, `PADDLE_WEBHOOK_SECRET_KEY` |
| pipecat | `openai.api_key`, `deepgram.api_key`, `cartesia.api_key`, `xai.api_key`, `elevenlabs.api_key`, `google.api_key`, `openrouter.api_key`, `local_llm.endpoints` | the upper case name of the key, for example `OPENAI_API_KEY`, `LOCAL_LLM_ENDPOINTS` |
| ai | `mcp_oauth.<vendor>.client_secret` and `.client_id` for github, linear, asana, hubspot (atomic group) | `MCP_OAUTH_<VENDOR>_CLIENT_SECRET` and `_CLIENT_ID` |

Secrets allow 512 bytes. `local_llm.endpoints` is one line of JSON and allows 2020 bytes.
The `client_id` of the MCP OAuth keys is a parameter, not a secret, so it is written with
`ai-control` only; write it before the secret.

## Read path

- A key resolves db, then env, then default. The service passes `os.LookupEnv` to the
  config handler (email, message, tts, transcribe and the LLM keys of ai pass none).
- billing reads the Paddle keys through the resolver. pipecat
  resolves the platform keys when a session starts and passes only the keys that the
  session needs to the runner; the runner keeps its environment variables as the fallback.
  A change of a stored key is used from the next session.
- The master key list comes from `BILLING_CONFIG_ENCRYPTION_KEYS` and
  `PIPECAT_CONFIG_ENCRYPTION_KEYS`; only the Go service reads it.
- The Service Config API forwards through `requesthandler.ConfigV1*` to the queue of the
  service; `billing` and `pipecat` are new members of the allowed service names.

## Error semantics

- A row that can not be decrypted is never replaced by the environment value. The key
  reports `db_undecryptable` (`db_invalid` for a value that does not fit the declaration).
- A read error is an error, not "no row". billing answers a failed key read with 500.
- billing answers a rejected webhook signature (wrong, expired or malformed) with 401 and a
  key read or decrypt failure with 500. The hook service maps only the unauthorized answer
  to a validation error (400); every other billing error keeps the current behavior.
- The signature timestamp window stays at 300 seconds.
- Secrets, signatures and plaintext values are never written to logs or events.
- The API shows `source=env`, `env_name` and `env_shadowed` for the first time, for the keys
  that declare an environment name.

## Rolling order and prerequisites

1. Apply the new migrations (`billing_configs`, `billing_config_histories`,
   `pipecat_configs`, `pipecat_config_histories`) before the services start.
2. Set the master key variables for billing and pipecat before a secret is written.
3. Deploy billing, pipecat and ai first, then api-manager. Until a service has the
   config RPC, the API answers 503 for it.
4. Hook and billing must both run the new image before the stored webhook secret is used.
   The secret stored in billing must equal the hook environment value
   `PADDLE_WEBHOOK_SECRET_KEY`: the hook still checks its own value first, and a request that
   passes the hook but is rejected by billing reaches the sender as a 400.
5. Check the current environment value of `LOCAL_LLM_ENDPOINTS` and of the MCP OAuth client
   ids against the length limits before storing them.

## Follow-up

- Remove the environment fallback of billing, pipecat and the MCP OAuth keys after the
  values are stored and verified.
- Review the 400 mapping of the hook service when the fallback is gone.
- The RST pages of the Service Config API and the self hosting pages describe the
  fallback; revise them together with the removal.
