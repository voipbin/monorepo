# bin-ai-manager

AI conversation orchestration service. Manages AI configurations, active call sessions (AIcalls), message history, and LLM tool execution. Delegates real-time audio to `bin-pipecat-manager`.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, request routing table, event subscriptions
- [docs/domain.md](docs/domain.md) — AI/AIcall/Message/Summary models, LLM engines, tool definitions
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, queue names
- [docs/operations.md](docs/operations.md) — config flags, Prometheus metrics, CLI tool, common commands

## Key concepts

- **AI** — per-customer LLM configuration (engine, model, init prompt, tool list, TTS/STT settings)
- **AIcall** — active session linking an AI config to a reference (call / conversation / task); lifecycle: `initiating → progressing → terminating → terminated`
- **Tool** — LLM function-call capability (`connect_call`, `send_email`, `set_variables`, etc.); definitions in `pkg/toolhandler/definitions.go`
- **Per-pod routing** — follow-up RPCs to pipecat-manager target `bin-manager.pipecat-manager.request.<POD_IP>` using `pipecatcall.HostID`; see [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md)

## CRITICAL: Platform LLM key names

The stored key is `llm.openai_compat.api_key` (set with `ai-control config set --key llm.openai_compat.api_key`). It is used for OpenAI and other OpenAI-compatible providers (e.g. Grok). The service does not read the former `engine_key_chatgpt` flag or the `ENGINE_KEY_CHATGPT` variable any more: the key is read from the database only (VOIP-1721).

**Gemini audit uses a separate key.** The `geminiaudithandler` requires a Google API key (`AIza...`), stored as `llm.gemini.api_key` (database only; `GOOGLE_API_KEY` is read by bin-pipecat-manager from its own environment, not by this service). This is distinct from `llm.openai_compat.api_key` which holds an OpenAI-style key (`sk-...`). Both keys are read for each call through `internal/aiconfig` (never cache them; they have no environment source); see docs/operations.md, "Platform LLM keys". They can also be written through the admin API (`/service_configs/ai`, served as `/v1/configs` by the shared `configrpc`, history source `api`); that route is handled before the request log entry exists and must never log the request, because the body of a secret write is the secret.

## Common commands

```bash
# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Build
go build -o ./bin/ai-manager ./cmd/ai-manager/

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)

# Regenerate mocks
go generate ./...
```

## Database migrations

The schema of `ai_ai_audits`, `ai_ai_prompt_histories`, `ai_ai_prompt_proposals`, `ai_aicall_participants`, `ai_aicalls`, `ai_ais`, `ai_mcp_oauth_states`, `ai_mcp_servers`, `ai_messages`, `ai_summaries` and `ai_teams` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, down is the no-op `SELECT 1`). Table options are `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci` on four tables (`ai_ai_audits`, `ai_ai_prompt_histories`, `ai_ai_prompt_proposals`, `ai_aicall_participants`) and the column charset of `ai_messages.content`; JSON columns are declared `json`. Version table: `ai_schema_migrations`.
- `cmd/ai-manager` runs `migrationhandler.Run` in `runDaemon`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `ai-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do).
- New database: the service creates its tables at startup; the baseline's table options apply only to a database that has none of the tables. `utf8mb4_0900_ai_ci` needs an engine that provides it (tested: MySQL 8.0 and MariaDB 12.3; other versions are untested).
- Update `scripts/database_scripts_test/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/` (file structure and table options only; the dynamic check below checks the migrated database). Git relative check: `scripts/check-service-migrations.sh bin-ai-manager` after `git fetch origin main`. Dynamic check on scratch containers: `scripts/verify-service-migrations.sh bin-ai-manager` (repo root).

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Mocks co-located with handler packages. See `pkg/aicallhandler/start_test.go` for reference.

## Cache invariants

Any code path that writes to `ai_ais` — transactional or not — MUST call `h.aiUpdateToCache(ctx, aiID)` after the write succeeds. See `pkg/dbhandler/ai.go` (`AICreate`, `AIDelete`, `AIUpdate`) for the convention and `pkg/dbhandler/aipromptproposal.go` (`AIAcceptProposal`) for the transactional case. For transactional writers, the cache refresh must happen AFTER `tx.Commit()` (use a `refreshCache` flag + deferred closure so the call-site is single and consistent across exit paths).

To audit: `grep -nE 'UPDATE ai_ais|INSERT INTO ai_ais' pkg/dbhandler/*.go` and confirm every match is followed by a path that ultimately calls `aiUpdateToCache`.

The same convention does NOT apply to `ai_ai_prompt_proposals` or `ai_ai_prompt_histories` (those are not cached today).
