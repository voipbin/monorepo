# VOIP-1723: stored Paddle credentials for bin-billing-manager

## Goal

Store the Paddle API key and the Paddle webhook secret in the database, encrypted, so that an operator can enter and rotate them with `billing-control config` or the admin API with no restart. The environment variables stay as the fallback while no value is stored.

## Decisions

- Keys: `paddle.api_key` (fallback `PADDLE_API_KEY`) and `paddle.webhook_secret_key` (fallback `PADDLE_WEBHOOK_SECRET_KEY`). Both are independent secrets of at most 512 bytes.
- Tables: `billing_configs` and `billing_config_histories`, added by migrations `000002` and `000003` (new tables only, `CREATE TABLE IF NOT EXISTS`, one statement per file). CHECK constraint names carry the table prefix (`chk_billing_configs_*`, `chk_billing_config_histories_*`) because the names are unique per schema.
- Resolution happens for every use, never cached: a stored value wins, then the environment variable (read with `os.LookupEnv`), then "not configured". A read error, a timeout (2 seconds) or a decrypt error is an error and never falls through to the environment.
- `paddlehandler.NewPaddleHandler` takes two key functions. `CreatePortalSession` reads the API key before it builds the request. A new `VerifyWebhook` method checks the webhook signature.
- Webhook signature: the same algorithm and 300 second window as the hook service. No secret means the check is skipped. A rejected signature is answered 401. An unreadable secret is answered 500 (fail closed). The new log lines carry no signature and no body.
- The config requests (`/v1/configs`) are routed before the request log entry exists, as in the other services that store settings.
- The `PaddleAPIKey` config field and the `--paddle_api_key` flag are removed. Nothing reads them any more.
- The master key list is `BILLING_CONFIG_ENCRYPTION_KEYS` (environment only, no flag), passed by the Komodo stack and by the Kubernetes deployment (optional secret reference).

## Rollout order

1. Deploy the hook service and every replica of the billing service with this change. The new billing service skips the signature check while no secret is stored, so behavior is unchanged.
2. Pass the master key list to the billing service.
3. Store `paddle.webhook_secret_key` with the same value that the hook service holds, then check one webhook. Store `paddle.api_key` if it should come from the database.

An older billing replica ignores stored values, and an older hook service does not forward the signature. Do not store the webhook secret before both are on the new image.

## Rollback

Revert the image. The tables stay and are ignored by the older code. The older code reads `PADDLE_API_KEY` from the environment again, so keep the environment value in step until the rollback window ends.
