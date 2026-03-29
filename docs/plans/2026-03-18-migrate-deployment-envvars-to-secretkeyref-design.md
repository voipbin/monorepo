# Design: Migrate Deployment Env Vars to secretKeyRef

**Date:** 2026-03-18

## Problem Statement

29 out of 30 services use `value: ${TEMPLATE_VAR}` pattern for environment variables in their K8s deployment files. This relies on CI/CD template substitution, which means sensitive values pass through the pipeline as plain text. Only `bin-rag-manager` uses the more secure `secretKeyRef` pattern, referencing the existing `voipbin` K8s secret.

## Goal

Convert all 29 services from `value: ${TEMPLATE}` to `secretKeyRef` referencing `voipbin`, following the existing `bin-rag-manager` pattern.

## Approach

### Rules

- Only convert `${TEMPLATE}` vars to `secretKeyRef`.
- Hardcoded values (`/metrics`, `:2112`, `1`, ClickHouse address, etc.) stay as plain `value:`.
- `fieldRef` vars (`POD_NAME`, `POD_IP`, `NODE_IP`, `POD_NAMESPACE`) stay unchanged.
- Env var name stays the same — only the value source changes.

### Conversion Pattern

Before:
```yaml
- name: DATABASE_DSN
  value: ${DATABASE_DSN}
```

After:
```yaml
- name: DATABASE_DSN
  valueFrom:
    secretKeyRef:
      name: voipbin
      key: DATABASE_DSN
```

### Name Mapping (env name differs from secret key)

For most vars, `secretKeyRef.key` matches the env var name directly. These are the exceptions where the secret key differs from the template variable:

| Service | Env Name | secretKeyRef.key | Notes |
|---------|----------|-----------------|-------|
| `bin-ai-manager` | `ENGINE_KEY_CHATGPT` | `OPENAI_API_KEY` | Template was `${AUTHTOKEN_OPENAI}`, secret has `OPENAI_API_KEY` |
| `bin-pipecat-manager` (2 containers) | `OPENAI_API_KEY` | `OPENAI_API_KEY` | Template was `${AUTHTOKEN_OPENAI}`, secret has `OPENAI_API_KEY` |
| `bin-message-manager` | `AUTHTOKEN_TELNYX` | `TELNYX_TOKEN` | Template was `${TELNYX_TOKEN}`, matches secret key |
| `bin-api-manager` | `SSL_PRIVKEY_BASE64` | `SSL_PRIVKEY_API_BASE64` | Template was `${SSL_PRIVKEY_API_BASE64}`, matches secret key |
| `bin-api-manager` | `SSL_CERT_BASE64` | `SSL_CERT_API_BASE64` | Template was `${SSL_CERT_API_BASE64}`, matches secret key |
| `bin-api-manager` | `GCP_BUCKET_NAME` | `GCP_BUCKET_NAME_TMP` | Template was `${GCP_BUCKET_NAME_TMP}`, matches secret key |
| `bin-hook-manager` | `SSL_PRIVKEY_BASE64` | `SSL_PRIVKEY_HOOK_BASE64` | Template was `${SSL_PRIVKEY_HOOK_BASE64}`, matches secret key |
| `bin-hook-manager` | `SSL_CERT_BASE64` | `SSL_CERT_HOOK_BASE64` | Template was `${SSL_CERT_HOOK_BASE64}`, matches secret key |
| `bin-registrar-manager` | `DATABASE_DSN_BIN` | `DATABASE_DSN` | Template was `${DATABASE_DSN}`, matches secret key |
| `bin-timeline-manager` | `GCS_BUCKET_NAME` | `GCP_BUCKET_NAME_MEDIA` | Template was `${GCP_BUCKET_NAME_MEDIA}`, matches secret key |

### Multi-Container Pod

`bin-pipecat-manager` has 2 containers — both need conversion:
- `pipecat-manager`: 13 template vars (DATABASE_DSN, RABBITMQ_ADDRESS, REDIS_*, API keys)
- `pipecat-script-runner`: 6 template vars (CARTESIA_API_KEY, ELEVENLABS_API_KEY, OPENAI_API_KEY, DEEPGRAM_API_KEY, XAI_API_KEY, GOOGLE_API_KEY)

### Secret Key Inventory

The `voipbin` K8s secret contains 51 keys covering all template variables used across deployments. Key gap resolved:
- `REDIS_PASSWORD` — added to secret with empty value (Redis has no password)

## Rollout Strategy

Incremental rollout in 4 alphabetical batches. Each batch is deployed and verified before proceeding.

**Batch 1 (7 services):**
`agent-manager`, `ai-manager`, `api-manager`, `billing-manager`, `call-manager`, `campaign-manager`, `conference-manager`

**Batch 2 (7 services):**
`contact-manager`, `conversation-manager`, `customer-manager`, `email-manager`, `flow-manager`, `hook-manager`, `message-manager`

**Batch 3 (7 services):**
`number-manager`, `outdial-manager`, `pipecat-manager`, `queue-manager`, `registrar-manager`, `route-manager`, `sentinel-manager`

**Batch 4 (8 services):**
`storage-manager`, `tag-manager`, `talk-manager`, `timeline-manager`, `transcribe-manager`, `transfer-manager`, `tts-manager`, `webhook-manager`

## Out of Scope

- `bin-rag-manager` — already uses `secretKeyRef`
- `bin-dbscheme-manager/k8s/job.yml` — no env vars
- `bin-number-manager/k8s/cronjob.yml` — hardcoded RabbitMQ address in command args, not env vars (separate follow-up)
- CI/CD pipeline changes (template substitution still works for unconverted services)
- Secret key additions or renames
- Changes to hardcoded or fieldRef values

## Risks and Mitigations

- **Risk:** Secret key value mismatch causes service failure at startup.
  **Mitigation:** Incremental batch rollout. Verify each batch before proceeding.

- **Risk:** Name mapping error causes wrong secret key reference.
  **Mitigation:** All mappings documented and reviewed. The secret key used matches what CI/CD was substituting.

- **Risk:** `REDIS_PASSWORD` empty value causes Redis connection issues.
  **Mitigation:** Verified live deployments already use empty REDIS_PASSWORD. No behavior change.
