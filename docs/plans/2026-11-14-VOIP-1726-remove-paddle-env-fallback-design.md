# VOIP-1726: Remove the Paddle environment fallbacks and the hook signature check

## Goal

1. `bin-billing-manager` reads `paddle.api_key` and `paddle.webhook_secret_key` from the database only.
2. The Paddle webhook signature is verified by `bin-billing-manager` alone. `bin-hook-manager` no longer holds the secret or checks the signature.
3. A webhook is rejected (fail closed) while `paddle.webhook_secret_key` is not stored.

Out of scope: `bin-pipecat-manager`, the MCP OAuth keys of `bin-ai-manager`, `confighandler.SourceEnv`, the OpenAPI enum and generated code (they stay because other services still use the environment lookup).

## Decisions

- Not stored: `VerifyWebhook` returns `paddlehandler.ErrCredentialNotConfigured` (distinct from `ErrKeyUnavailable` and from `confighandler.ErrKeyNotConfigured`). The listen handler answers 500, the hook service passes a 500 on, so the sender may retry. A rejected signature stays 401 (hook: 400). The alternative 401 mixes a server side configuration state with a forged request and may not be retried by the sender; answering 200 and dropping the event loses it.
- The retry behavior of the sender (4xx and 5xx, and whether a retry carries a new signature timestamp) is not verified. Events rejected while no secret was stored may have to be resent from the sender dashboard.
- The hook service keeps the body size limit, the check that the `Paddle-Signature` header exists (400 if missing), and forwards the header unchanged. It no longer verifies the signature.
- `CreatePortalSession` returns `ErrCredentialNotConfigured` before any request leaves when the API key is not stored. The caller still answers 400.
- `configkeys` declares no environment name for the two keys, and `billingconfig.New` passes a nil environment lookup (like email, message, tts and transcribe). `config list` then reports the source `db` or `none`, no `env_name`, and `env_shadowed` is always false.

## Deployment and rollback

Precondition: both keys are stored (`billing-control config list`: source `db`, `has_secret` true) and the webhook secret equals the value that Paddle uses.

| hook | billing | precondition holds | precondition does not hold |
|---|---|---|---|
| old | old | as before | billing skips, hook verifies |
| new | old | billing verifies with the stored secret | no signature check (fail open) |
| old | new | normal | billing rejects with 500 |
| new | new | normal | billing rejects with 500 |

Deploy the billing service first and wait until every replica runs the new image, then deploy the hook service. To roll back the billing service, roll back the hook service first or together with it, and confirm the precondition again. Rolling back the hook service needs the old manifest and an unchanged `PADDLE_WEBHOOK_SECRET_KEY` value (update it first if the stored secret was rotated). The variable and the cluster secret key are not deleted by this change; remove them in a separate step after the rollback window.

## Verification

Unit tests for the not-stored cases (webhook 500 and no event processing, portal session without a request), the unchanged 401 and read error 500, the nil environment lookup (a process variable is ignored), the golden key table without an environment name, and the hook service without any signature code. Full service verification, the RST rebuild, and a repository grep of the removed names.
