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

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Mocks co-located with handler packages. See `pkg/aicallhandler/start_test.go` for reference.
