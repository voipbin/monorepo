# Migrate Deployment Env Vars to secretKeyRef — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert all 29 services' K8s deployment env vars from `value: ${TEMPLATE}` to `secretKeyRef` referencing `bin-manager-secrets`.

**Architecture:** YAML-only changes to `bin-*/k8s/deployment.yml` files. No Go code changes. Each `value: ${TEMPLATE_VAR}` entry becomes a `valueFrom.secretKeyRef` block. Hardcoded values and `fieldRef` entries are left untouched.

**Tech Stack:** Kubernetes YAML, `bin-manager-secrets` K8s Secret

**Design doc:** `docs/plans/2026-03-18-migrate-deployment-envvars-to-secretkeyref-design.md`

---

## Reference: Conversion Rules

**Standard conversion** (env name matches secret key):
```yaml
# BEFORE
- name: DATABASE_DSN
  value: ${DATABASE_DSN}

# AFTER
- name: DATABASE_DSN
  valueFrom:
    secretKeyRef:
      name: bin-manager-secrets
      key: DATABASE_DSN
```

**Name-mapped conversion** (env name differs from secret key):
```yaml
# BEFORE
- name: ENGINE_KEY_CHATGPT
  value: "${AUTHTOKEN_OPENAI}"

# AFTER
- name: ENGINE_KEY_CHATGPT
  valueFrom:
    secretKeyRef:
      name: bin-manager-secrets
      key: OPENAI_API_KEY
```

**DO NOT convert:**
- Hardcoded values: `value: "/metrics"`, `value: ":2112"`, `value: "1"`, `value: "clickhouse.infrastructure..."`, `value: "voipbin.net"`, etc.
- `fieldRef` entries: `POD_NAME`, `POD_IP`, `NODE_IP`, `POD_NAMESPACE`
- Anything in `bin-rag-manager` (already converted)

**Name mapping table** (env name → secretKeyRef.key):

| Service | Env Name | Secret Key |
|---------|----------|------------|
| bin-ai-manager | ENGINE_KEY_CHATGPT | OPENAI_API_KEY |
| bin-api-manager | SSL_PRIVKEY_BASE64 | SSL_PRIVKEY_API_BASE64 |
| bin-api-manager | SSL_CERT_BASE64 | SSL_CERT_API_BASE64 |
| bin-api-manager | GCP_BUCKET_NAME | GCP_BUCKET_NAME_TMP |
| bin-hook-manager | SSL_PRIVKEY_BASE64 | SSL_PRIVKEY_HOOK_BASE64 |
| bin-hook-manager | SSL_CERT_BASE64 | SSL_CERT_HOOK_BASE64 |
| bin-message-manager | AUTHTOKEN_TELNYX | TELNYX_TOKEN |
| bin-pipecat-manager (both containers) | OPENAI_API_KEY | OPENAI_API_KEY |
| bin-registrar-manager | DATABASE_DSN_BIN | DATABASE_DSN |
| bin-timeline-manager | GCS_BUCKET_NAME | GCP_BUCKET_NAME_MEDIA |

All other template vars: `secretKeyRef.key` = env var name (e.g., `DATABASE_DSN` → key: `DATABASE_DSN`).

---

## Task 1: Batch 1 — agent-manager, ai-manager, api-manager, billing-manager, call-manager, campaign-manager, conference-manager

**Files:**
- Modify: `bin-agent-manager/k8s/deployment.yml`
- Modify: `bin-ai-manager/k8s/deployment.yml`
- Modify: `bin-api-manager/k8s/deployment.yml`
- Modify: `bin-billing-manager/k8s/deployment.yml`
- Modify: `bin-call-manager/k8s/deployment.yml`
- Modify: `bin-campaign-manager/k8s/deployment.yml`
- Modify: `bin-conference-manager/k8s/deployment.yml`

**Step 1: Convert bin-agent-manager**

Convert these template vars to secretKeyRef (standard mapping):
- `DATABASE_DSN` → key: `DATABASE_DSN`
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `REDIS_ADDRESS` → key: `REDIS_ADDRESS`
- `REDIS_PASSWORD` → key: `REDIS_PASSWORD`

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS` (all hardcoded).

**Step 2: Convert bin-ai-manager**

Convert these template vars to secretKeyRef:
- `DATABASE_DSN` → key: `DATABASE_DSN`
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `REDIS_ADDRESS` → key: `REDIS_ADDRESS`
- `REDIS_PASSWORD` → key: `REDIS_PASSWORD`
- `ENGINE_KEY_CHATGPT` → key: `OPENAI_API_KEY` (**name-mapped**)

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 3: Convert bin-api-manager**

Convert these template vars to secretKeyRef:
- `DATABASE_DSN` → key: `DATABASE_DSN`
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `REDIS_ADDRESS` → key: `REDIS_ADDRESS`
- `REDIS_PASSWORD` → key: `REDIS_PASSWORD`
- `SSL_PRIVKEY_BASE64` → key: `SSL_PRIVKEY_API_BASE64` (**name-mapped**)
- `SSL_CERT_BASE64` → key: `SSL_CERT_API_BASE64` (**name-mapped**)
- `GCP_PROJECT_ID` → key: `GCP_PROJECT_ID`
- `GCP_BUCKET_NAME` → key: `GCP_BUCKET_NAME_TMP` (**name-mapped**)
- `JWT_KEY` → key: `JWT_KEY`

Leave untouched: `POD_NAME`, `POD_NAMESPACE`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`.

**Step 4: Convert bin-billing-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 5: Convert bin-call-manager**

Convert these template vars to secretKeyRef:
- `DATABASE_DSN` → key: `DATABASE_DSN`
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `REDIS_ADDRESS` → key: `REDIS_ADDRESS`
- `REDIS_PASSWORD` → key: `REDIS_PASSWORD`
- `HOMER_API_ADDRESS` → key: `HOMER_API_ADDRESS`
- `HOMER_AUTH_TOKEN` → key: `HOMER_AUTH_TOKEN`
- `HOMER_WHITELIST` → key: `HOMER_WHITELIST`

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `PROJECT_BASE_DOMAIN`, `PROJECT_BUCKET_NAME`, `CLICKHOUSE_ADDRESS` (all hardcoded).

**Step 6: Convert bin-campaign-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 7: Convert bin-conference-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 8: Validate YAML syntax for all 7 files**

Run:
```bash
for svc in agent-manager ai-manager api-manager billing-manager call-manager campaign-manager conference-manager; do
  echo "--- bin-$svc ---"
  python3 -c "import yaml; yaml.safe_load(open('bin-$svc/k8s/deployment.yml'))" && echo "OK" || echo "FAIL"
done
```
Expected: All 7 print "OK".

**Step 9: Commit Batch 1**

```bash
git add bin-agent-manager/k8s/deployment.yml bin-ai-manager/k8s/deployment.yml bin-api-manager/k8s/deployment.yml bin-billing-manager/k8s/deployment.yml bin-call-manager/k8s/deployment.yml bin-campaign-manager/k8s/deployment.yml bin-conference-manager/k8s/deployment.yml
git commit -m "NOJIRA-Migrate-deployment-envvars-to-secretKeyRef

Batch 1: Convert env vars from template substitution to secretKeyRef.

- bin-agent-manager: Convert 4 template vars to secretKeyRef
- bin-ai-manager: Convert 5 template vars to secretKeyRef (ENGINE_KEY_CHATGPT -> OPENAI_API_KEY)
- bin-api-manager: Convert 9 template vars to secretKeyRef (SSL/GCP name-mapped)
- bin-billing-manager: Convert 4 template vars to secretKeyRef
- bin-call-manager: Convert 7 template vars to secretKeyRef
- bin-campaign-manager: Convert 4 template vars to secretKeyRef
- bin-conference-manager: Convert 4 template vars to secretKeyRef"
```

---

## Task 2: Batch 2 — contact-manager, conversation-manager, customer-manager, email-manager, flow-manager, hook-manager, message-manager

**Files:**
- Modify: `bin-contact-manager/k8s/deployment.yml`
- Modify: `bin-conversation-manager/k8s/deployment.yml`
- Modify: `bin-customer-manager/k8s/deployment.yml`
- Modify: `bin-email-manager/k8s/deployment.yml`
- Modify: `bin-flow-manager/k8s/deployment.yml`
- Modify: `bin-hook-manager/k8s/deployment.yml`
- Modify: `bin-message-manager/k8s/deployment.yml`

**Step 1: Convert bin-contact-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 2: Convert bin-conversation-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 3: Convert bin-customer-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 4: Convert bin-email-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `MAILGUN_API_KEY` → key: `MAILGUN_API_KEY`
- `SENDGRID_API_KEY` → key: `SENDGRID_API_KEY`

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 5: Convert bin-flow-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 6: Convert bin-hook-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `SSL_PRIVKEY_BASE64` → key: `SSL_PRIVKEY_HOOK_BASE64` (**name-mapped**)
- `SSL_CERT_BASE64` → key: `SSL_CERT_HOOK_BASE64` (**name-mapped**)

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`.

**Step 7: Convert bin-message-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `AUTHTOKEN_MESSAGEBIRD` → key: `AUTHTOKEN_MESSAGEBIRD`
- `AUTHTOKEN_TELNYX` → key: `TELNYX_TOKEN` (**name-mapped**)

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 8: Validate YAML syntax for all 7 files**

Run:
```bash
for svc in contact-manager conversation-manager customer-manager email-manager flow-manager hook-manager message-manager; do
  echo "--- bin-$svc ---"
  python3 -c "import yaml; yaml.safe_load(open('bin-$svc/k8s/deployment.yml'))" && echo "OK" || echo "FAIL"
done
```
Expected: All 7 print "OK".

**Step 9: Commit Batch 2**

```bash
git add bin-contact-manager/k8s/deployment.yml bin-conversation-manager/k8s/deployment.yml bin-customer-manager/k8s/deployment.yml bin-email-manager/k8s/deployment.yml bin-flow-manager/k8s/deployment.yml bin-hook-manager/k8s/deployment.yml bin-message-manager/k8s/deployment.yml
git commit -m "NOJIRA-Migrate-deployment-envvars-to-secretKeyRef

Batch 2: Convert env vars from template substitution to secretKeyRef.

- bin-contact-manager: Convert 4 template vars to secretKeyRef
- bin-conversation-manager: Convert 4 template vars to secretKeyRef
- bin-customer-manager: Convert 4 template vars to secretKeyRef
- bin-email-manager: Convert 6 template vars to secretKeyRef
- bin-flow-manager: Convert 4 template vars to secretKeyRef
- bin-hook-manager: Convert 6 template vars to secretKeyRef (SSL name-mapped)
- bin-message-manager: Convert 6 template vars to secretKeyRef (AUTHTOKEN_TELNYX -> TELNYX_TOKEN)"
```

---

## Task 3: Batch 3 — number-manager, outdial-manager, pipecat-manager, queue-manager, registrar-manager, route-manager, sentinel-manager

**Files:**
- Modify: `bin-number-manager/k8s/deployment.yml`
- Modify: `bin-outdial-manager/k8s/deployment.yml`
- Modify: `bin-pipecat-manager/k8s/deployment.yml`
- Modify: `bin-queue-manager/k8s/deployment.yml`
- Modify: `bin-registrar-manager/k8s/deployment.yml`
- Modify: `bin-route-manager/k8s/deployment.yml`
- Modify: `bin-sentinel-manager/k8s/deployment.yml`

**Step 1: Convert bin-number-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `TWILIO_SID` → key: `TWILIO_SID`
- `TWILIO_TOKEN` → key: `TWILIO_TOKEN`
- `TELNYX_CONNECTION_ID` → key: `TELNYX_CONNECTION_ID`
- `TELNYX_PROFILE_ID` → key: `TELNYX_PROFILE_ID`
- `TELNYX_TOKEN` → key: `TELNYX_TOKEN`

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 2: Convert bin-outdial-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 3: Convert bin-pipecat-manager (2 containers)**

Container `pipecat-manager` — convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `CARTESIA_API_KEY` → key: `CARTESIA_API_KEY`
- `ELEVENLABS_API_KEY` → key: `ELEVENLABS_API_KEY`
- `OPENAI_API_KEY` → key: `OPENAI_API_KEY` (template was `${AUTHTOKEN_OPENAI}`, use secret key `OPENAI_API_KEY`)
- `DEEPGRAM_API_KEY` → key: `DEEPGRAM_API_KEY`
- `XAI_API_KEY` → key: `XAI_API_KEY`
- `GOOGLE_API_KEY` → key: `GOOGLE_API_KEY`

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

Container `pipecat-script-runner` — convert:
- `CARTESIA_API_KEY` → key: `CARTESIA_API_KEY`
- `ELEVENLABS_API_KEY` → key: `ELEVENLABS_API_KEY`
- `OPENAI_API_KEY` → key: `OPENAI_API_KEY` (template was `${AUTHTOKEN_OPENAI}`, use secret key `OPENAI_API_KEY`)
- `DEEPGRAM_API_KEY` → key: `DEEPGRAM_API_KEY`
- `XAI_API_KEY` → key: `XAI_API_KEY`
- `GOOGLE_API_KEY` → key: `GOOGLE_API_KEY`

**Step 4: Convert bin-queue-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 5: Convert bin-registrar-manager**

Convert:
- `DATABASE_DSN_ASTERISK` → key: `DATABASE_DSN_ASTERISK`
- `DATABASE_DSN_BIN` → key: `DATABASE_DSN` (**name-mapped**)
- `DOMAIN_NAME_EXTENSION` → key: `DOMAIN_NAME_EXTENSION`
- `DOMAIN_NAME_TRUNK` → key: `DOMAIN_NAME_TRUNK`
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `REDIS_ADDRESS` → key: `REDIS_ADDRESS`
- `REDIS_PASSWORD` → key: `REDIS_PASSWORD`

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 6: Convert bin-route-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 7: Convert bin-sentinel-manager**

Convert: `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS` (only 1 template var).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `CLICKHOUSE_ADDRESS`.

**Step 8: Validate YAML syntax for all 7 files**

Run:
```bash
for svc in number-manager outdial-manager pipecat-manager queue-manager registrar-manager route-manager sentinel-manager; do
  echo "--- bin-$svc ---"
  python3 -c "import yaml; yaml.safe_load(open('bin-$svc/k8s/deployment.yml'))" && echo "OK" || echo "FAIL"
done
```
Expected: All 7 print "OK".

**Step 9: Commit Batch 3**

```bash
git add bin-number-manager/k8s/deployment.yml bin-outdial-manager/k8s/deployment.yml bin-pipecat-manager/k8s/deployment.yml bin-queue-manager/k8s/deployment.yml bin-registrar-manager/k8s/deployment.yml bin-route-manager/k8s/deployment.yml bin-sentinel-manager/k8s/deployment.yml
git commit -m "NOJIRA-Migrate-deployment-envvars-to-secretKeyRef

Batch 3: Convert env vars from template substitution to secretKeyRef.

- bin-number-manager: Convert 9 template vars to secretKeyRef
- bin-outdial-manager: Convert 4 template vars to secretKeyRef
- bin-pipecat-manager: Convert 16 template vars to secretKeyRef across 2 containers (OPENAI_API_KEY mapped)
- bin-queue-manager: Convert 4 template vars to secretKeyRef
- bin-registrar-manager: Convert 7 template vars to secretKeyRef (DATABASE_DSN_BIN -> DATABASE_DSN)
- bin-route-manager: Convert 4 template vars to secretKeyRef
- bin-sentinel-manager: Convert 1 template var to secretKeyRef"
```

---

## Task 4: Batch 4 — storage-manager, tag-manager, talk-manager, timeline-manager, transcribe-manager, transfer-manager, tts-manager, webhook-manager

**Files:**
- Modify: `bin-storage-manager/k8s/deployment.yml`
- Modify: `bin-tag-manager/k8s/deployment.yml`
- Modify: `bin-talk-manager/k8s/deployment.yml`
- Modify: `bin-timeline-manager/k8s/deployment.yml`
- Modify: `bin-transcribe-manager/k8s/deployment.yml`
- Modify: `bin-transfer-manager/k8s/deployment.yml`
- Modify: `bin-tts-manager/k8s/deployment.yml`
- Modify: `bin-webhook-manager/k8s/deployment.yml`

**Step 1: Convert bin-storage-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `GCP_PROJECT_ID` → key: `GCP_PROJECT_ID`
- `GCP_BUCKET_NAME_TMP` → key: `GCP_BUCKET_NAME_TMP`
- `GCP_BUCKET_NAME_MEDIA` → key: `GCP_BUCKET_NAME_MEDIA`
- `JWT_KEY` → key: `JWT_KEY`

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 2: Convert bin-tag-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 3: Convert bin-talk-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 4: Convert bin-timeline-manager**

Convert:
- `RABBITMQ_ADDRESS` → key: `RABBITMQ_ADDRESS`
- `HOMER_API_ADDRESS` → key: `HOMER_API_ADDRESS`
- `HOMER_AUTH_TOKEN` → key: `HOMER_AUTH_TOKEN`
- `GCS_BUCKET_NAME` → key: `GCP_BUCKET_NAME_MEDIA` (**name-mapped**)

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `CLICKHOUSE_ADDRESS`, `CLICKHOUSE_DATABASE`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`.

**Step 5: Convert bin-transcribe-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `AWS_ACCESS_KEY` → key: `AWS_ACCESS_KEY`
- `AWS_SECRET_KEY` → key: `AWS_SECRET_KEY`

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`, `STREAMING_LISTEN_PORT`, `STT_PROVIDER_PRIORITY`.

**Step 6: Convert bin-transfer-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `NODE_IP`, `POD_IP` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 7: Convert bin-tts-manager**

Convert:
- `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (standard)
- `AWS_ACCESS_KEY` → key: `AWS_ACCESS_KEY`
- `AWS_SECRET_KEY` → key: `AWS_SECRET_KEY`
- `ELEVENLABS_API_KEY` → key: `ELEVENLABS_API_KEY`

Leave untouched: `POD_IP`, `POD_NAMESPACE`, `POD_NAME` (fieldRef), `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 8: Convert bin-webhook-manager**

Convert: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD` (all standard mapping).

Leave untouched: `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `REDIS_DATABASE`, `CLICKHOUSE_ADDRESS`.

**Step 9: Validate YAML syntax for all 8 files**

Run:
```bash
for svc in storage-manager tag-manager talk-manager timeline-manager transcribe-manager transfer-manager tts-manager webhook-manager; do
  echo "--- bin-$svc ---"
  python3 -c "import yaml; yaml.safe_load(open('bin-$svc/k8s/deployment.yml'))" && echo "OK" || echo "FAIL"
done
```
Expected: All 8 print "OK".

**Step 10: Commit Batch 4**

```bash
git add bin-storage-manager/k8s/deployment.yml bin-tag-manager/k8s/deployment.yml bin-talk-manager/k8s/deployment.yml bin-timeline-manager/k8s/deployment.yml bin-transcribe-manager/k8s/deployment.yml bin-transfer-manager/k8s/deployment.yml bin-tts-manager/k8s/deployment.yml bin-webhook-manager/k8s/deployment.yml
git commit -m "NOJIRA-Migrate-deployment-envvars-to-secretKeyRef

Batch 4: Convert env vars from template substitution to secretKeyRef.

- bin-storage-manager: Convert 8 template vars to secretKeyRef
- bin-tag-manager: Convert 4 template vars to secretKeyRef
- bin-talk-manager: Convert 4 template vars to secretKeyRef
- bin-timeline-manager: Convert 4 template vars to secretKeyRef (GCS_BUCKET_NAME -> GCP_BUCKET_NAME_MEDIA)
- bin-transcribe-manager: Convert 6 template vars to secretKeyRef
- bin-transfer-manager: Convert 4 template vars to secretKeyRef
- bin-tts-manager: Convert 7 template vars to secretKeyRef
- bin-webhook-manager: Convert 4 template vars to secretKeyRef"
```

---

## Task 5: Final validation and PR

**Step 1: Verify all 29 deployment files have no remaining `${` template vars**

Run:
```bash
grep -r '\${' bin-*/k8s/deployment.yml
```
Expected: No output (zero matches). If any matches remain, a file was missed.

**Step 2: Verify all secretKeyRef entries reference `bin-manager-secrets`**

Run:
```bash
grep -A1 'secretKeyRef' bin-*/k8s/deployment.yml | grep 'name:' | sort -u
```
Expected: Only `name: bin-manager-secrets`.

**Step 3: Count total conversions**

Run:
```bash
grep -c 'secretKeyRef' bin-*/k8s/deployment.yml | grep -v ':0'
```
Expected: 29 files with secretKeyRef entries (all services except rag-manager which was already done).

**Step 4: Commit design docs**

```bash
git add docs/plans/2026-03-18-migrate-deployment-envvars-to-secretkeyref-design.md docs/plans/2026-03-18-migrate-deployment-envvars-to-secretkeyref.md
git commit -m "NOJIRA-Migrate-deployment-envvars-to-secretKeyRef

- docs: Add design document and implementation plan for secretKeyRef migration"
```

**Step 5: Push and create PR**

```bash
git push -u origin NOJIRA-Migrate-deployment-envvars-to-secretKeyRef
```

Create PR with title `NOJIRA-Migrate-deployment-envvars-to-secretKeyRef` and body describing all 4 batches.
