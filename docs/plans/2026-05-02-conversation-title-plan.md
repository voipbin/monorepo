# Conversation Title Auto-Generation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the broken `"conversation with " + peer.TargetName` title generation with a channel-aware `"SMS · Alice (+14155551234)"` / `"LINE · Alice"` format.

**Architecture:** A new `internal/convtitle/` package contains the pure `Build(convType, peer)` function. Both `conversationhandler.GetOrCreateBySelfAndPeer` and `linehandler.hookEventTypeFollow` call it. `Create()` is not modified. Existing conversations are not migrated.

**Tech Stack:** Go 1.21+, gomock (go.uber.org/mock), golangci-lint. All work in worktree `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/`.

**Design doc:** `docs/plans/2026-05-02-conversation-title-design.md`

---

## Task 1: Create `internal/convtitle` package (TDD)

**Files:**
- Create: `bin-conversation-manager/internal/convtitle/build.go`
- Create: `bin-conversation-manager/internal/convtitle/build_test.go`

All commands run from `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/bin-conversation-manager/`.

---

**Step 1: Create the test file**

Create `internal/convtitle/build_test.go`:

```go
package convtitle

import (
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
)

func Test_Build(t *testing.T) {
	tests := []struct {
		name       string
		convType   conversation.Type
		peer       commonaddress.Address
		wantName   string
		wantDetail string
	}{
		{
			name:     "sms with name and target",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155551234",
				TargetName: "Alice",
			},
			wantName:   "SMS · Alice (+14155551234)",
			wantDetail: "SMS conversation",
		},
		{
			name:     "sms with target only",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+14155551234",
			},
			wantName:   "SMS · +14155551234",
			wantDetail: "SMS conversation",
		},
		{
			name:     "sms with neither",
			convType: conversation.TypeMessage,
			peer:     commonaddress.Address{Type: commonaddress.TypeTel},
			wantName:   "SMS · Unknown",
			wantDetail: "SMS conversation",
		},
		{
			name:     "line with name and opaque target",
			convType: conversation.TypeLine,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "Uabcdef1234567890",
				TargetName: "Alice",
			},
			// Target is a LINE user ID — opaque, must NOT appear in name
			wantName:   "LINE · Alice",
			wantDetail: "LINE conversation",
		},
		{
			name:     "line with opaque target only",
			convType: conversation.TypeLine,
			peer: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "Uabcdef1234567890",
			},
			// Target is opaque but it's the only info available, so it shows as fallback
			wantName:   "LINE · Uabcdef1234567890",
			wantDetail: "LINE conversation",
		},
		{
			name:     "line with neither",
			convType: conversation.TypeLine,
			peer:     commonaddress.Address{Type: commonaddress.TypeLine},
			wantName:   "LINE · Unknown",
			wantDetail: "LINE conversation",
		},
		{
			name:     "email with name and target",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeEmail,
				Target:     "alice@example.com",
				TargetName: "Alice",
			},
			wantName:   "SMS · Alice (alice@example.com)",
			wantDetail: "SMS conversation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDetail := Build(tt.convType, tt.peer)
			if gotName != tt.wantName {
				t.Errorf("Build() name = %q, want %q", gotName, tt.wantName)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("Build() detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func Test_humanReadableTarget(t *testing.T) {
	tests := []struct {
		addrType commonaddress.Type
		want     bool
	}{
		{commonaddress.TypeTel, true},
		{commonaddress.TypeEmail, true},
		{commonaddress.TypeSIP, true},
		{commonaddress.TypeExtension, true},
		{commonaddress.TypeLine, false},
		{commonaddress.TypeAgent, false},
		{commonaddress.TypeAI, false},
		{commonaddress.TypeConference, false},
		{commonaddress.TypeNone, false},
		{"unknown_future_type", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.addrType), func(t *testing.T) {
			if got := humanReadableTarget(tt.addrType); got != tt.want {
				t.Errorf("humanReadableTarget(%q) = %v, want %v", tt.addrType, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run the test to confirm it fails (package doesn't exist yet)**

```bash
go test ./internal/convtitle/...
```

Expected: `cannot find package "monorepo/bin-conversation-manager/internal/convtitle"` or similar build error.

**Step 3: Create the implementation**

Create `internal/convtitle/build.go`:

```go
package convtitle

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
)

const titleSep = " · " // U+00B7 MIDDLE DOT

// Build returns the auto-generated name and detail for a new conversation.
func Build(convType conversation.Type, peer commonaddress.Address) (name, detail string) {
	label := channelLabel(convType)
	name = label + titleSep + peerName(peer)
	detail = label + " conversation"
	return
}

// channelLabel returns the human-readable channel name for a conversation type.
// When adding a new conversation.Type, add a case here — do not rely on the fallback.
func channelLabel(t conversation.Type) string {
	switch t {
	case conversation.TypeLine:
		return "LINE"
	case conversation.TypeMessage:
		return "SMS"
	default:
		return string(t)
	}
}

// peerName returns the best available display name for a peer address.
// For human-readable address types (tel, email, sip, extension), the raw
// Target is appended in parentheses when a TargetName is also present.
// For opaque types (line user IDs, UUIDs), the raw Target is suppressed.
func peerName(peer commonaddress.Address) string {
	if peer.TargetName != "" {
		if humanReadableTarget(peer.Type) && peer.Target != "" {
			return peer.TargetName + " (" + peer.Target + ")"
		}
		return peer.TargetName
	}
	if peer.Target != "" {
		return peer.Target
	}
	return "Unknown"
}

// humanReadableTarget returns true when the address Target field contains
// a human-readable identifier (phone number, email, SIP URI, extension).
// New address types with human-readable targets must be added here explicitly.
// Unknown types default to false (opaque) for safety.
func humanReadableTarget(t commonaddress.Type) bool {
	switch t {
	case commonaddress.TypeTel, commonaddress.TypeEmail,
		commonaddress.TypeSIP, commonaddress.TypeExtension:
		return true
	default:
		return false
	}
}
```

**Step 4: Run the tests to confirm they pass**

```bash
go test ./internal/convtitle/... -v
```

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/convtitle/build.go internal/convtitle/build_test.go
git commit -m "NOJIRA-Conversation-title-improvement

- bin-conversation-manager: Add internal/convtitle package for channel-aware title generation"
```

---

## Task 2: Update `db.go` — fix TypeMessage bug and call convtitle.Build

**Files:**
- Modify: `pkg/conversationhandler/db.go` (lines 56–76)
- Modify: `pkg/conversationhandler/db_test.go` (lines 696–697)

All commands run from `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/bin-conversation-manager/`.

---

**Step 1: Update the test expectation first (TDD — make the new expectation exist before changing the code)**

In `pkg/conversationhandler/db_test.go`, find `Test_GetOrCreateBySelfAndPeer_Create` (around line 659).

Change lines 696–697 from:
```go
Name:     "conversation with Peer Name",
Detail:   "conversation with Peer Name",
```
to:
```go
Name:   "SMS · Peer Name (+0987654321)",
Detail: "SMS conversation",
```

**Step 2: Run the test to confirm it now fails (code not updated yet)**

```bash
go test ./pkg/conversationhandler/... -run Test_GetOrCreateBySelfAndPeer_Create -v
```

Expected: FAIL — gomock "unexpected call" or arg mismatch on `ConversationCreate`.

**Step 3: Update `db.go`**

In `pkg/conversationhandler/db.go`, add the import:
```go
convtitle "monorepo/bin-conversation-manager/internal/convtitle"
```

Find `GetOrCreateBySelfAndPeer` (around line 60–69). Replace:
```go
res, err = h.Create(
    ctx,
    customerID,
    "conversation with "+peer.TargetName,
    "conversation with "+peer.TargetName,
    conversation.TypeMessage,
    dialogID,
    self,
    peer,
)
```
with:
```go
name, detail := convtitle.Build(conversationType, peer)
res, err = h.Create(
    ctx,
    customerID,
    name,
    detail,
    conversationType,
    dialogID,
    self,
    peer,
)
```

**Step 4: Run the test to confirm it passes**

```bash
go test ./pkg/conversationhandler/... -run Test_GetOrCreateBySelfAndPeer_Create -v
```

Expected: PASS.

**Step 5: Run all conversationhandler tests**

```bash
go test ./pkg/conversationhandler/... -v
```

Expected: All PASS.

**Step 6: Commit**

```bash
git add pkg/conversationhandler/db.go pkg/conversationhandler/db_test.go
git commit -m "NOJIRA-Conversation-title-improvement

- bin-conversation-manager: Use convtitle.Build in GetOrCreateBySelfAndPeer; fix hardcoded TypeMessage bug"
```

---

## Task 3: Update `hook.go` — call convtitle.Build in LINE follow event

**Files:**
- Modify: `pkg/linehandler/hook.go` (lines 110–119)

All commands run from `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/bin-conversation-manager/`.

Note: `pkg/linehandler/hook_test.go` tests are currently fully commented out — no test expectations to update.

---

**Step 1: Update `hook.go`**

Add the import:
```go
convtitle "monorepo/bin-conversation-manager/internal/convtitle"
```

Find `hookEventTypeFollow` (around line 110). Replace:
```go
res, err := h.reqHandler.ConversationV1ConversationCreate(
    ctx,
    ac.CustomerID,
    "Conversation with "+peer.TargetName,
    "Auto generated conversation",
    conversation.TypeLine,
    dialogID,
    self,
    *peer,
)
```
with:
```go
name, detail := convtitle.Build(conversation.TypeLine, *peer)
res, err := h.reqHandler.ConversationV1ConversationCreate(
    ctx,
    ac.CustomerID,
    name,
    detail,
    conversation.TypeLine,
    dialogID,
    self,
    *peer,
)
```

**Step 2: Build to confirm no compile errors**

```bash
go build ./...
```

Expected: SUCCESS with no errors.

**Step 3: Run all tests**

```bash
go test ./...
```

Expected: All PASS.

**Step 4: Commit**

```bash
git add pkg/linehandler/hook.go
git commit -m "NOJIRA-Conversation-title-improvement

- bin-conversation-manager: Use convtitle.Build in LINE follow event handler"
```

---

## Task 4: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/conversation_overview.rst`
- Rebuild: `bin-api-manager/docsdev/build/` (force-add, tracked in git)

All commands run from `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/`.

---

**Step 1: Add note to conversation_overview.rst**

Open `bin-api-manager/docsdev/source/conversation_overview.rst`. Find a suitable section (e.g., after the fields table or in the Overview section). Add:

```rst
Auto-Generated Titles
+++++++++++++++++++++

When a conversation is created automatically (incoming SMS or LINE follow event), the platform
generates a title in the format ``{channel} · {peer}``:

* ``SMS · Alice (+14155551234)`` — SMS from a known contact with phone number
* ``SMS · +14155551234`` — SMS from an unknown number
* ``LINE · Alice`` — LINE message from a known display name
* ``SMS · Unknown`` — peer information unavailable

The ``name`` field can be updated at any time via ``PUT /v1/conversations/{id}``.
```

**Step 2: Rebuild HTML (clean build)**

```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

Expected: `Build succeeded.` with no errors or warnings that break output.

**Step 3: Force-add build output and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement
git add bin-api-manager/docsdev/source/conversation_overview.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Conversation-title-improvement

- bin-api-manager: Document auto-generated conversation title format in RST; rebuild HTML"
```

---

## Task 5: Full verification

All commands run from `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-title-improvement/bin-conversation-manager/`.

---

**Step 1: Run the full verification workflow**

```bash
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps succeed with zero errors. If any step fails, fix the issue before proceeding.

**Step 2: Verify the new title format is visible in test output**

```bash
go test ./internal/convtitle/... -v
go test ./pkg/conversationhandler/... -v -run Test_GetOrCreateBySelfAndPeer
```

Expected: `PASS` on all relevant tests. Confirm `"SMS · Peer Name (+0987654321)"` appears in the output.

**Step 3: Final commit (if any go mod tidy/vendor changes)**

If `go mod tidy` or `go mod vendor` produced changes (unlikely since no new external deps):
```bash
git add go.mod go.sum
git commit -m "NOJIRA-Conversation-title-improvement

- bin-conversation-manager: go mod tidy after adding internal/convtitle package"
```

---

## Checklist

- [ ] Task 1: `internal/convtitle/build.go` + `build_test.go` — all tests pass
- [ ] Task 2: `db.go` updated + `db_test.go` expectation fixed — `Test_GetOrCreateBySelfAndPeer_Create` passes
- [ ] Task 3: `hook.go` updated — builds cleanly, all tests pass
- [ ] Task 4: RST updated + HTML rebuilt + both committed
- [ ] Task 5: Full verification (go mod tidy + vendor + generate + test + lint) all green
