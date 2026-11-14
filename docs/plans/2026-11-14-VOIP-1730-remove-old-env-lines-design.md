# VOIP-1730: Remove the old vendor key environment lines from compose and k8s

## Goal

The vendor keys of `bin-ai-manager`, `bin-email-manager`, `bin-message-manager`, `bin-number-manager`, `bin-tts-manager` and `bin-transcribe-manager` are read from the database only (VOIP-1721, VOIP-1722). The container environment entries that still pass the old variables are removed from `komodo/docker-compose.yml` and `k8s/deployment.yml` of these six services. No code change.

## Entries removed (matched by name, not by line number)

- ai: `ENGINE_KEY_CHATGPT`, `GOOGLE_API_KEY`
- email: `SENDGRID_API_KEY`, `MAILGUN_API_KEY`, `MAILGUN_WEBHOOK_SIGNING_KEY` (compose only; k8s has no such entry)
- message: `AUTHTOKEN_MESSAGEBIRD`, `AUTHTOKEN_TELNYX`
- number: `TWILIO_SID`, `TWILIO_TOKEN`, `TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID`, `TELNYX_TOKEN`
- tts: `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`, `ELEVENLABS_API_KEY`
- transcribe: `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`

In compose one list item `- NAME=...`; in k8s the whole `- name: NAME` block (name, valueFrom, secretKeyRef with name and key).

## Kept

Config encryption keys, `DATABASE_DSN`, `POD_IP`, ai `LOCAL_LLM_MODELS`, `MCP_OAUTH_*`, `MCP_SECRET_ENCRYPTION_KEYS`, `ENGINE_BASE_KEY_ENCRYPTION_KEYS`, tts and transcribe `GCP_SA_JSON` and `GOOGLE_APPLICATION_CREDENTIALS` with their mounts, and every line of `bin-pipecat-manager`. `bin-pipecat-manager` still reads `GOOGLE_API_KEY` and `ELEVENLABS_API_KEY`, so the shared Komodo variables stay. Komodo variables and cluster secret keys are not deleted here.

## Verification

1. Before the change, `git grep` of the names in the six services with pathspec excludes `:!*_test.go`, `:!*.md`, `:!*/vendor/*`, `:!*/komodo/docker-compose.yml`, `:!*/k8s/deployment.yml` returns nothing (no code reads them).
2. A script removes by name and asserts: each entry is removed exactly once per file (and zero times where the design says none), the set of removed names equals the list above, the environment name sets before and after differ only by the removed names; the other lines and the kept names (`DATABASE_DSN`, `POD_IP`, `GCP_SA_JSON`, `GOOGLE_APPLICATION_CREDENTIALS`, `*_CONFIG_ENCRYPTION_KEYS`, number `NUMBER_SECRET_ENCRYPTION_KEYS`, ai `LOCAL_LLM_MODELS`, `MCP_OAUTH_*`) are still present and `bin-pipecat-manager` has no diff (checked with `git diff`); both files still parse as YAML.
3. `go test` of the six services (including the compose definition tests of message and transcribe).
4. After the change, `git grep` of the names in `*/komodo/docker-compose.yml` and `*/k8s/deployment.yml` of the six services returns nothing.
5. `git diff --numstat` shows only deletions in the twelve manifest files and no change in `bin-pipecat-manager`; the script writes nothing unless every check passes. After the document change, `git grep -nE "in step|still injects" -- 'bin-*-manager/docs/operations.md'` finds none of the old sentences, and `git grep -n "environment pair" -- bin-transcribe-manager/docs/operations.md` shows the restored-lines wording.

## Documents

These sentences tell the operator to keep the deployment variables in step during the rollback window (or say the old image reads them "as long as the deployment still injects them"). That is no longer possible, so each is changed to: the deployment does not inject the variables any more, and a rollback to an image from before VOIP-1721 or VOIP-1722 first restores these lines (revert of this change). The Komodo variables and cluster secret keys are not deleted by this change, so a restored line supplies the value again.

- `bin-ai-manager/docs/operations.md:120`, `bin-tts-manager/docs/operations.md:159`, `bin-email-manager/docs/operations.md:102`, `bin-message-manager/docs/operations.md:142`, `bin-number-manager/docs/operations.md:154` (different wording), `bin-transcribe-manager/docs/operations.md:128` and `:135`.

Old design documents under `docs/plans` stay as history.

## Deployment and rollback

The six services are deployed from main with the newest image, which does not read these variables. A rolling deploy that still has an old image running next to a new one is not a case here, because the compose and the image change in the same deploy of main. An image from before VOIP-1721 or VOIP-1722 needs the variables again, so a rollback to such an image restores the lines first (`git revert` of this change); the stored keys are not affected.
