# Fix Pipecat Tool Cache Staleness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent stale tool cache in pipecat-manager by periodically refreshing tools from ai-manager.

**Architecture:** Add a background goroutine in `main.go` that calls the existing `FetchTools()` on a 5-minute ticker. No new methods, no interface changes — just wiring in `main.go`.

**Tech Stack:** Go, existing `toolhandler.FetchTools()`

---

### Task 1: Add periodic tool refresh goroutine

**Files:**
- Modify: `bin-pipecat-manager/cmd/pipecat-manager/main.go:122-126`

**Step 1: Add the `time` import and refresh goroutine**

In `main.go`, add `"time"` to the import block, then add the goroutine after the initial `FetchTools` call (after line 126):

```go
// Create tool handler and fetch tools from ai-manager
toolHandler := toolhandler.NewToolHandler(requestHandler)
if err := toolHandler.FetchTools(context.Background()); err != nil {
    log.Warnf("Could not fetch tools from ai-manager: %v. Continuing with empty tool set.", err)
}

// Periodically refresh tools to pick up changes after rolling deployments.
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        if err := toolHandler.FetchTools(context.Background()); err != nil {
            log.Warnf("Could not refresh tools from ai-manager: %v. Keeping current cache.", err)
        }
    }
}()
```

**Step 2: Run verification**

```bash
cd bin-pipecat-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All pass. No new dependencies, no interface changes, no mock regen needed.

**Step 3: Commit**

```bash
git add bin-pipecat-manager/cmd/pipecat-manager/main.go
git commit -m "NOJIRA-Fix-pipecat-tool-cache-staleness

- bin-pipecat-manager: Add periodic background refresh of tool cache from ai-manager
- bin-pipecat-manager: Refresh runs every 5 minutes to pick up changes after rolling deployments
- bin-pipecat-manager: On refresh failure, keeps current cache and logs warning"
```
