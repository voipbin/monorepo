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

## CRITICAL: engine_key_chatgpt

The config flag is `engine_key_chatgpt` / env `ENGINE_KEY_CHATGPT` (not `openai_api_key`). This key is used for OpenAI and other OpenAI-compatible providers (e.g. Grok). Do not rename it.

**Gemini audit uses a separate key.** The `geminiaudithandler` requires a Google API key (`AIza...`), configured via `google_api_key` / env `GOOGLE_API_KEY`. This is distinct from `ENGINE_KEY_CHATGPT` which holds an OpenAI-style key (`sk-...`).

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
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head (apart from the collation of the four tables above, see `docs/e1-allowed-diff.txt`).
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'ai_ais' already exists`) and is unsupported while the Alembic stream still creates the ai tables. After the Alembic stream the baseline changes nothing; its table options apply only to a database that has none of the tables. `utf8mb4_0900_ai_ci` needs an engine that provides it (tested: MySQL 8.0 and MariaDB 12.3; other versions are untested).
- Update `scripts/database_scripts_test/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/` (file structure and table options only; it does not compare columns with Alembic, the dynamic check below does). Git relative check: `scripts/check-service-migrations.sh bin-ai-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> E1_ALLOWED_DIFF=bin-ai-manager/docs/e1-allowed-diff.txt scripts/verify-service-migrations.sh bin-ai-manager` (repo root; the file lists the intended collation differences of the four tables from the Alembic head, which show on MariaDB only).

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Mocks co-located with handler packages. See `pkg/aicallhandler/start_test.go` for reference.

## Cache invariants

Any code path that writes to `ai_ais` — transactional or not — MUST call `h.aiUpdateToCache(ctx, aiID)` after the write succeeds. See `pkg/dbhandler/ai.go` (`AICreate`, `AIDelete`, `AIUpdate`) for the convention and `pkg/dbhandler/aipromptproposal.go` (`AIAcceptProposal`) for the transactional case. For transactional writers, the cache refresh must happen AFTER `tx.Commit()` (use a `refreshCache` flag + deferred closure so the call-site is single and consistent across exit paths).

To audit: `grep -nE 'UPDATE ai_ais|INSERT INTO ai_ais' pkg/dbhandler/*.go` and confirm every match is followed by a path that ultimately calls `aiUpdateToCache`.

The same convention does NOT apply to `ai_ai_prompt_proposals` or `ai_ai_prompt_histories` (those are not cached today).
