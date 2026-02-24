# Fix webhook-manager bugs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 4 bugs in webhook-manager: retry body reuse, response body close, UUID cache key format, and unstarted subscribehandler.

**Architecture:** All fixes are in `bin-webhook-manager`. Changes touch `message.go` (HTTP delivery), `webhook.go` (callers), `cachehandler/handler.go` (Redis keys), and `cmd/webhook-manager/main.go` (wiring). No interface changes, no new dependencies.

**Tech Stack:** Go, RabbitMQ, Redis, gomock

---

### Task 1: Fix sendMessage retry body reuse and response body handling

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/message.go`

**Step 1: Rewrite sendMessage**

Replace the entire `sendMessage` function. Key changes:
- Move `http.NewRequest` inside the retry loop so each attempt gets a fresh body
- Drain and close response body inside sendMessage (no more returning `*http.Response`)
- Change return type from `(*http.Response, error)` to `error`
- Log response status inside sendMessage (callers relied on this for debug logs)

```go
package webhookhandler

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// sendMessage sends the message to the given uri with the given method and data.
func (h *webhookHandler) sendMessage(uri string, method string, dataType string, data []byte) error {

	log := logrus.WithFields(
		logrus.Fields{
			"func":   "sendMessage",
			"uri":    uri,
			"method": method,
		},
	)
	log.Debugf("Sending a message. data: %v", data)

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
		if err != nil {
			log.Errorf("Could not create request. err: %v", err)
			return err
		}

		if data != nil && dataType != "" {
			req.Header.Set("Content-Type", dataType)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Could not send the request correctly. Making a retrying: %d, err: %v", i, err)
			lastErr = err
			time.Sleep(time.Second * 1)
			continue
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		log.WithField("response_status", resp.StatusCode).Debugf("Sent the event correctly.")
		return nil
	}

	log.Errorf("Could not send the request. err: %v", lastErr)
	return lastErr
}
```

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs/bin-webhook-manager && go build ./...`
Expected: Compilation error because `webhook.go` still expects `(*http.Response, error)` return type. That's expected — we fix callers in Task 2.

---

### Task 2: Update sendMessage callers in webhook.go

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/webhook.go`

**Step 1: Update SendWebhookToCustomer goroutine**

In `webhook.go`, replace lines 41-48 (the goroutine inside `SendWebhookToCustomer`):

Old:
```go
		go func() {
			res, err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(dataType), data)
			if err != nil {
				log.Errorf("Could not send a request. err: %v", err)
				return
			}
			log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", m.WebhookMethod, m.WebhookURI, res.StatusCode)
		}()
```

New:
```go
		go func() {
			if err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(dataType), data); err != nil {
				log.Errorf("Could not send a request. err: %v", err)
			}
		}()
```

**Step 2: Update SendWebhookToURI goroutine**

In `webhook.go`, replace lines 74-81 (the goroutine inside `SendWebhookToURI`):

Old:
```go
	go func() {
		res, err := h.sendMessage(uri, string(method), string(dataType), data)
		if err != nil {
			log.Errorf("Could not send a request. err: %v", err)
			return
		}
		log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", method, uri, res.StatusCode)
	}()
```

New:
```go
	go func() {
		if err := h.sendMessage(uri, string(method), string(dataType), data); err != nil {
			log.Errorf("Could not send a request. err: %v", err)
		}
	}()
```

**Step 3: Verify build and tests pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs/bin-webhook-manager && go build ./... && go test ./pkg/webhookhandler/...`
Expected: Build succeeds. Tests pass (existing tests don't mock `sendMessage` — they use real `webhookHandler` but the goroutine calls `sendMessage` against `"test.com"` which will fail silently in background goroutines, same as before).

---

### Task 3: Fix Redis cache key UUID format

**Files:**
- Modify: `bin-webhook-manager/pkg/cachehandler/handler.go`

**Step 1: Fix format verb in AccountSet (line 42)**

Old:
```go
	key := fmt.Sprintf("webhook.account:%d", u.ID)
```

New:
```go
	key := fmt.Sprintf("webhook.account:%s", u.ID)
```

**Step 2: Fix format verb in AccountGet (line 53)**

Old:
```go
	key := fmt.Sprintf("webhook.account:%d", id)
```

New:
```go
	key := fmt.Sprintf("webhook.account:%s", id)
```

**Step 3: Verify build and tests pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs/bin-webhook-manager && go build ./... && go test ./pkg/cachehandler/... ./pkg/dbhandler/... ./pkg/accounthandler/...`
Expected: PASS. No cachehandler tests exist but dbhandler and accounthandler tests use mocked cache, so they're unaffected.

---

### Task 4: Wire subscribehandler in main.go

**Files:**
- Modify: `bin-webhook-manager/cmd/webhook-manager/main.go`

**Step 1: Add subscribehandler import**

Add to the imports block:
```go
	"monorepo/bin-webhook-manager/pkg/subscribehandler"
```

**Step 2: Refactor run() to share dependencies between runListen and runSubscribe**

The current `runListen` creates `sockHandler`, `accountHandler`, etc. internally. We need these shared with `runSubscribe`. Refactor `run()` to create shared dependencies and pass them to both functions.

Replace the `run()` function (lines 122-135):

```go
// run runs the webhook-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebhookEvent, serviceName, "")
	accountHandler := accounthandler.NewAccountHandler(dbHandler, reqHandler)

	// run listen
	if err := runListen(sockHandler, notifyHandler, accountHandler, dbHandler); err != nil {
		return errors.Wrapf(err, "could not run listen handler")
	}

	// run subscribe
	if err := runSubscribe(sockHandler, accountHandler); err != nil {
		return errors.Wrapf(err, "could not run subscribe handler")
	}

	log.Debug("All handlers started successfully")
	return nil
}
```

**Step 3: Simplify runListen to accept pre-built handlers**

Replace the `runListen` function (lines 137-165):

```go
// runListen runs the listen handler
func runListen(sockHandler sockhandler.SockHandler, notifyHandler notifyhandler.NotifyHandler, accountHandler accounthandler.AccountHandler, db dbhandler.DBHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})
	log.Debugf("Running listen handler")

	whHandler := webhookhandler.NewWebhookHandler(db, notifyHandler, accountHandler)
	listenHandler := listenhandler.NewListenHandler(sockHandler, whHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameWebhookRequest), string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run the listen handler correctly")
	}

	return nil
}
```

**Step 4: Add runSubscribe function**

Add after `runListen`:

```go
// runSubscribe runs the subscribe handler
func runSubscribe(sockHandler sockhandler.SockHandler, accountHandler accounthandler.AccountHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})
	log.Debugf("Running subscribe handler")

	subscribeTargets := string(commonoutline.QueueNameCustomerEvent)
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameWebhookSubscribe),
		subscribeTargets,
		accountHandler,
	)

	if err := subHandler.Run(); err != nil {
		return errors.Wrapf(err, "could not run the subscribe handler correctly")
	}

	return nil
}
```

**Step 5: Verify build passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs/bin-webhook-manager && go build ./...`
Expected: Build succeeds.

---

### Task 5: Run full verification workflow and commit

**Step 1: Run full verification**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs/bin-webhook-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 2: Check for conflicts with main**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs && \
git fetch origin main && \
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```
Expected: No conflicts (we're only changing webhook-manager files).

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs && \
git add bin-webhook-manager/pkg/webhookhandler/message.go \
        bin-webhook-manager/pkg/webhookhandler/webhook.go \
        bin-webhook-manager/pkg/cachehandler/handler.go \
        bin-webhook-manager/cmd/webhook-manager/main.go \
        docs/plans/2026-02-24-fix-webhook-manager-bugs-design.md \
        docs/plans/2026-02-24-fix-webhook-manager-bugs-plan.md
```

Then commit with message:
```
NOJIRA-fix-webhook-manager-bugs

Fix 4 bugs in webhook-manager affecting webhook delivery reliability and cache correctness.

- bin-webhook-manager: Fix HTTP request body consumed on retry in sendMessage
- bin-webhook-manager: Drain and close response body inside sendMessage instead of deferring after return
- bin-webhook-manager: Fix Redis cache key format from %d to %s for UUID fields
- bin-webhook-manager: Wire subscribehandler to keep webhook config cache warm on customer updates
```

**Step 4: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-webhook-manager-bugs && \
git push -u origin NOJIRA-fix-webhook-manager-bugs
```

Then create PR with `gh pr create`.
