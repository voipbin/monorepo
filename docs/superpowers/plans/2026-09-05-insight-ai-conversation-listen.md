# Insight AI Conversation Listen (VOIP-1470) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Plan status:** APPROVED, rev 5 (plan review round 1: REQUEST_CHANGES, 14 findings; round 2: REQUEST_CHANGES, 6 mechanical; round 3: APPROVE with 3 cosmetic LOWs; round 4: APPROVE with 1 cosmetic LOW (two residual `253-256` anchors, corrected to `252-256` in rev 5). Two consecutive approvals; plan review loop CLOSED. See the matrices at the end).

**Goal:** Let a Case Insight Assistant AIcall follow a messaging conversation (SMS/MMS, LINE, WhatsApp, webchat) in real time and push `notify_agent` notes, by extending the call listen pipeline with a conversation-manager intake.

**Architecture:** The approved design is `docs/plans/2026-09-05-insight-ai-conversation-listen-design.md` (rev 6). Section numbers below (`§x.y`) refer to it. The existing trigger API, Redis buffers, `RunListenTurn`, `notify_agent` tool and proactive rows are reused; only the input source (a `conversation-manager.conversation.*.message_created` binding resolved through `ai:listen:conversation:<conversation_id>`) and the lifecycle (deferred flush, turn-time Case status check) are new. No DB migration, no API change, no frontend change.

**Tech Stack:** Go 1.25 (toolchain in go.mod says 1.27.1), gomock (`go.uber.org/mock`), go-redis v8, Prometheus client, viper/pflag config. Monorepo local replaces (`../bin-*`), vendor regenerated with `go mod vendor` (never committed).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/VOIP-1470-Insight-AI-conversation-listen` (branch `VOIP-1470-Insight-AI-conversation-listen`). Every command below is relative to that directory. Never touch `~/gitvoipbin/monorepo` itself.

**Verification workflow (mandatory per service before each commit that touches it):**

```bash
cd <service-dir> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Commit convention:** title is the branch name `VOIP-1470-Insight-AI-conversation-listen`; body is `- bin-<service>: ...` bullets; no AI attribution. Because the title must match the branch, every commit in this plan uses the same title and a body describing that commit's slice.

---

## File structure

| File | Responsibility |
|---|---|
| `bin-contact-manager/models/kase/kase.go`, `kase_test.go` | `ReferenceTypeConversationMessage` constant + value pin |
| `bin-flow-manager/pkg/activeflowhandler/actionhandle.go` | replace the two bare `"call"` / `"conversation_message"` literals in `case_create` with the `kmkase` constants |
| `bin-conversation-manager/models/conversation/metadata.go` | comment fix only (who writes `ContactCaseID`) |
| `bin-ai-manager/internal/config/main.go`, `main_test.go` | three `aicall_listen_conversation_*` flags, env map, defaults, validation, `SetXxxForTest` |
| `bin-ai-manager/models/aicall/main.go` | `MetaKeyListenConversationID` |
| `bin-ai-manager/pkg/cachehandler/listen.go`, `listen_test.go`, `main.go` (+ regenerated `mock_main.go`) | `ai:listen:conversation:<id>` resolver quartet, `ListenPendingLen` |
| `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`, `metrics_listen_test.go` | `kind` label on start/turn/notify, two new conversation vecs |
| `bin-ai-manager/pkg/aicallhandler/listen.go`, `listen_test.go` | `listenKind`, conversation-aware `RunListenTurn`, prompt/header selection, `clearListenState` SREM |
| `bin-ai-manager/pkg/aicallhandler/listen_trigger.go`, `listen_trigger_test.go` | step-5 branch, `kind` labels |
| `bin-ai-manager/pkg/aicallhandler/listen_conversation.go`, `listen_conversation_test.go` (new) | `checkListenEligibleConversation`, `startListenConversation`, `EventCVMessageCreated`, line builder, deferred flush |
| `bin-ai-manager/pkg/aicallhandler/main.go` (+ regenerated `mock_main.go`) | interface method, `afterFunc` seam, `flushScheduled`, `ListenTurnConversationSystemPrompt` |
| `bin-ai-manager/pkg/aicallhandler/process.go`, `start.go`, `tool_insight.go` | terminate gate, idle-expiry cleanup, notify metric label |
| `bin-ai-manager/pkg/subscribehandler/main.go`, `conversationmanager.go` (new), `binding_golden_test.go` | binding, publisher const, dispatch, golden 13 |
| `bin-ai-manager/docs/operations.md`, `docs/architecture.md`, `docs/domain.md` | flag table, events section, metadata key |

---

### Task 1: `kase.ReferenceTypeConversationMessage` and the flow-manager literals

**Files:**
- Modify: `bin-contact-manager/models/kase/kase.go:101-109`
- Modify: `bin-contact-manager/models/kase/kase_test.go:171-181`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:3-34` (imports), `:1332`, `:1341`

- [ ] **Step 1: Write the failing value-pin test**

Append to `bin-contact-manager/models/kase/kase_test.go`:

```go
// TestReferenceTypeConversationMessageValue pins the stored value of a Case
// created from a messaging conversation. flow-manager's case_create action wrote
// this as a bare literal before VOIP-1470; the constant is now the single named
// spelling. Changing it silently orphans every stored conversation Case row.
func TestReferenceTypeConversationMessageValue(t *testing.T) {
	if ReferenceTypeConversationMessage != "conversation_message" {
		t.Errorf("ReferenceTypeConversationMessage mismatch. expected: %q, got: %q", "conversation_message", ReferenceTypeConversationMessage)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd bin-contact-manager && go test ./models/kase/ -run TestReferenceTypeConversationMessageValue -v`
Expected: FAIL to compile with `undefined: ReferenceTypeConversationMessage`

- [ ] **Step 3: Add the constant**

In `bin-contact-manager/models/kase/kase.go`, directly after `const ReferenceTypeCall = "call"` (line 109), add:

```go
// ReferenceTypeConversationMessage is the stored ReferenceType value for a Case
// created from a messaging conversation (SMS/MMS, LINE, WhatsApp, webchat,
// email). Case.ReferenceID is then the conversation id. Same untyped-string
// rationale as ReferenceTypeCall above.
//
// Introduced by docs/plans/2026-09-05-insight-ai-conversation-listen-design.md
// §5.1, which branches on it from bin-ai-manager; bin-flow-manager's case_create
// action is the producer.
const ReferenceTypeConversationMessage = "conversation_message"
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd bin-contact-manager && go test ./models/kase/ -v`
Expected: PASS (all tests in the package)

- [ ] **Step 5: Replace the flow-manager literals**

In `bin-flow-manager/pkg/activeflowhandler/actionhandle.go` add the import (keep the file's existing one-import-per-group style; place it after the `conversationmedia` import group):

```go
	kmkase "monorepo/bin-contact-manager/models/kase"
```

Change line 1332 `referenceType = "call"` to:

```go
		referenceType = kmkase.ReferenceTypeCall
```

Change line 1341 `referenceType = "conversation_message"` to:

```go
		referenceType = kmkase.ReferenceTypeConversationMessage
```

`bin-flow-manager/go.mod` already requires and replaces `monorepo/bin-contact-manager`; no go.mod edit is expected. If `go mod tidy` in step 6 changes go.mod/go.sum anyway, commit those changes.

- [ ] **Step 6: Run the verification workflow for both services**

Run: `cd bin-contact-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: all green.

Run: `cd bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: all green. (`go mod vendor` copies the new constant into flow-manager's vendor tree; the vendor directory is gitignored.)

- [ ] **Step 7: Commit**

```bash
git add bin-contact-manager/models/kase/kase.go bin-contact-manager/models/kase/kase_test.go bin-flow-manager/pkg/activeflowhandler/actionhandle.go
git add bin-flow-manager/go.mod bin-flow-manager/go.sum bin-contact-manager/go.mod bin-contact-manager/go.sum 2>/dev/null || true
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-contact-manager: Add kase.ReferenceTypeConversationMessage constant with a value-pin test
- bin-flow-manager: Use the kase reference-type constants in the case_create action instead of bare literals"
```

---

### Task 2: Configuration flags

**Files:**
- Modify: `bin-ai-manager/internal/config/main.go` (struct after line 62, pflags after line 111, env map after line 146, viper after line 208, `validateListenConfig` positives after line 310, `SetListenDefaultsForTest`, new `SetXxxForTest` helpers)
- Modify: `bin-ai-manager/internal/config/main_test.go` (`Test_ListenConfigDefaults`)
- Modify: `bin-ai-manager/docs/operations.md` (flag table, near line 140)

- [ ] **Step 1: Write the failing defaults test**

In `bin-ai-manager/internal/config/main_test.go`, inside `Test_ListenConfigDefaults` after the `AIcallListenStartLockReleaseTimeoutSeconds != 3` check, add:

```go
	if cfg.AIcallListenConversationEnabled {
		t.Errorf("AIcallListenConversationEnabled must default to false -- the conversation variant ships dark independently of the master switch")
	}
	if cfg.AIcallListenConversationMaxMessageChars != 2000 {
		t.Errorf("AIcallListenConversationMaxMessageChars mismatch. expected: 2000, got: %d", cfg.AIcallListenConversationMaxMessageChars)
	}
	if cfg.AIcallListenConversationFlushJitterMs != 1000 {
		t.Errorf("AIcallListenConversationFlushJitterMs mismatch. expected: 1000, got: %d", cfg.AIcallListenConversationFlushJitterMs)
	}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd bin-ai-manager && go test ./internal/config/ -run Test_ListenConfigDefaults -v`
Expected: FAIL to compile with `cfg.AIcallListenConversationEnabled undefined`

- [ ] **Step 3: Add the fields, flags, env map, viper reads, defaults, validation, helpers**

In the `Config` struct, after `AIcallListenStartLockReleaseTimeoutSeconds` (line 62), add:

```go
	// Insight AI realtime listening on conversation (message) Cases
	// (docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.12).
	AIcallListenConversationEnabled         bool // Variant switch, evaluated after AIcallListenEnabled and only on the conversation branch. Off: the trigger returns skipped_disabled and a running conversation listen stops at its next turn. Never affects call listens.
	AIcallListenConversationMaxMessageChars int  // Per-message truncation applied before a conversation line is buffered; the window is line-counted, not byte-counted, so an email body must not be allowed to dominate it.
	AIcallListenConversationFlushJitterMs   int  // Upper bound of the random jitter added to the deferred flush delay, so two replicas' timers for one AIcall do not race the debounce lock at the same instant.
```

In `Bootstrap`'s pflag block, after the `aicall_listen_start_lock_release_timeout_seconds` line (111), add:

```go
	f.Bool("aicall_listen_conversation_enabled", false, "Variant switch for Insight AI realtime listening on conversation (message) Cases; requires aicall_listen_enabled too")
	f.Int("aicall_listen_conversation_max_message_chars", 2000, "Per-message character cap applied before a conversation line is buffered for listening")
	f.Int("aicall_listen_conversation_flush_jitter_ms", 1000, "Upper bound (milliseconds) of the random jitter added to the conversation listen deferred-flush delay")
```

In the env map, after `"aicall_listen_start_lock_release_timeout_seconds"` (line 146), add:

```go
		"aicall_listen_conversation_enabled":           "AICALL_LISTEN_CONVERSATION_ENABLED",
		"aicall_listen_conversation_max_message_chars": "AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS",
		"aicall_listen_conversation_flush_jitter_ms":   "AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS",
```

In the `globalConfig = Config{...}` literal, after `AIcallListenStartLockReleaseTimeoutSeconds: viper.GetInt(...)` (line 208), add:

```go
			AIcallListenConversationEnabled:         viper.GetBool("aicall_listen_conversation_enabled"),
			AIcallListenConversationMaxMessageChars: viper.GetInt("aicall_listen_conversation_max_message_chars"),
			AIcallListenConversationFlushJitterMs:   viper.GetInt("aicall_listen_conversation_flush_jitter_ms"),
```

In `validateListenConfig`, append to the `positives` slice (after the `aicall_listen_start_lock_release_timeout_seconds` row):

```go
		{"aicall_listen_conversation_max_message_chars", globalConfig.AIcallListenConversationMaxMessageChars},
```

and, directly after the `if len(invalid) > 0 { ... }` block, add the non-negative check (zero jitter is a legitimate "no jitter" setting, so it is not in `positives`):

```go
	if globalConfig.AIcallListenConversationFlushJitterMs < 0 {
		return errors.Errorf("invalid listen configuration: aicall_listen_conversation_flush_jitter_ms must be >= 0, got %d", globalConfig.AIcallListenConversationFlushJitterMs)
	}
```

In `SetListenDefaultsForTest`, append:

```go
	globalConfig.AIcallListenConversationEnabled = false
	globalConfig.AIcallListenConversationMaxMessageChars = 2000
	globalConfig.AIcallListenConversationFlushJitterMs = 1000
```

After `SetAIcallListenQAContextSizeForTest`, add:

```go
// SetAIcallListenConversationEnabledForTest overrides the conversation listen
// variant switch in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenConversationEnabledForTest(enabled bool) {
	globalConfig.AIcallListenConversationEnabled = enabled
}

// SetAIcallListenConversationMaxMessageCharsForTest overrides the per-message
// truncation cap in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenConversationMaxMessageCharsForTest(chars int) {
	globalConfig.AIcallListenConversationMaxMessageChars = chars
}

// SetAIcallListenConversationFlushJitterMsForTest overrides the deferred-flush
// jitter bound in tests (0 makes the delay deterministic).
// USE ONLY FROM TESTS.
func SetAIcallListenConversationFlushJitterMsForTest(ms int) {
	globalConfig.AIcallListenConversationFlushJitterMs = ms
}
```

- [ ] **Step 4: Add validation rows**

`main_test.go` `Test_Validate_ListenSizing` (line 400) is a table of `{name string; mutate func()}` whose loop calls `SetListenDefaultsForTest()`, `tt.mutate()`, then expects `Validate()` to error. Append two rows after the `"a negative QA context size is rejected"` row (line 438):

```go
		{
			name:   "a zero conversation message cap is rejected",
			mutate: func() { globalConfig.AIcallListenConversationMaxMessageChars = 0 },
		},
		{
			name:   "a negative conversation flush jitter is rejected",
			mutate: func() { globalConfig.AIcallListenConversationFlushJitterMs = -1 },
		},
```

Then add a standalone test after `Test_Validate_ListenSizing` that pins the zero-jitter case as VALID (zero jitter is a legitimate "no jitter" setting and must not be caught by the `> 0` positives list):

```go
// Test_Validate_ZeroConversationFlushJitterIsAllowed pins that jitter is a
// non-negative bound, not a positive one: 0 disables jitter and must validate.
func Test_Validate_ZeroConversationFlushJitterIsAllowed(t *testing.T) {
	SetListenDefaultsForTest()
	defer SetListenDefaultsForTest()

	globalConfig.AIcallListenConversationFlushJitterMs = 0

	if err := Validate(); err != nil {
		t.Fatalf("zero jitter must be valid. err: %v", err)
	}
}
```

- [ ] **Step 5: Run the config tests**

Run: `cd bin-ai-manager && go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 6: Document the flags**

In `bin-ai-manager/docs/operations.md`, in the config table that lists `aicall_listen_enabled` (near line 140), add three rows in the same format as the neighbouring listen rows:

```
`aicall_listen_conversation_enabled` / `AICALL_LISTEN_CONVERSATION_ENABLED`, default `false`. Variant switch for realtime listening on conversation (message) Cases; evaluated after `aicall_listen_enabled` and only on the conversation branch. Off: the trigger returns `skipped_disabled`; a running conversation listen stops at its next turn. Call listens are unaffected.
`aicall_listen_conversation_max_message_chars` / `AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS`, default `2000`. Per-message character cap before a conversation line is buffered (suffix ` [truncated]`).
`aicall_listen_conversation_flush_jitter_ms` / `AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS`, default `1000`. Upper bound of the random jitter added to the deferred flush delay (`aicall_listen_evaluate_interval_seconds` + jitter).
```

- [ ] **Step 7: Commit**

```bash
git add bin-ai-manager/internal/config/main.go bin-ai-manager/internal/config/main_test.go bin-ai-manager/docs/operations.md
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Add aicall_listen_conversation_enabled, _max_message_chars and _flush_jitter_ms config flags with defaults, validation and test helpers"
```

---

### Task 3: Metadata key and cache primitives

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:40`
- Modify: `bin-ai-manager/pkg/cachehandler/listen.go`, `bin-ai-manager/pkg/cachehandler/main.go:49-71`
- Modify: `bin-ai-manager/pkg/cachehandler/listen_test.go`
- Modify: `bin-ai-manager/docs/domain.md` (metadata table, near line 46)

- [ ] **Step 1: Write the failing key test**

`Test_listenKeys` (`bin-ai-manager/pkg/cachehandler/listen_test.go:21`) is a `{name, got, expect string}` table with fixtures `transcribeID = 11111111-2222-3333-4444-555555555555` and `aicallID = 66666666-7777-8888-9999-aaaaaaaaaaaa`. Add one row after the `"start lock"` row (line 40):

```go
		{"conversation resolver set", listenConversationKey(transcribeID), "ai:listen:conversation:11111111-2222-3333-4444-555555555555"},
```

(reusing the `transcribeID` fixture as a conversation id; the key format is all that is pinned here).

- [ ] **Step 2: Run it to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/cachehandler/ -run Test_listenKeys -v`
Expected: FAIL to compile with `undefined: listenConversationKey`

- [ ] **Step 3: Add the metadata key**

In `bin-ai-manager/models/aicall/main.go` after `MetaKeyListenOwnsTranscribe` (line 40):

```go
// MetaKeyListenConversationID is the Metadata map key (string, a UUID) holding
// the conversation this AIcall is listening to (design
// docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.2.1).
// Metadata rather than a column: unlike listen_call_id there is no event-driven
// sweep that needs to query by it; every reader already has the row in hand. An
// AIcall carries at most one of MetaKeyListenTranscribeID / this key.
const MetaKeyListenConversationID = "listen_conversation_id"
```

- [ ] **Step 4: Add the cache primitives**

In `bin-ai-manager/pkg/cachehandler/listen.go`, after `listenStartLockKey` (line 64):

```go
// listenConversationKey is the conversation -> listening AIcall ids resolver
// (design 2026-09-05 §5.2.2). Same shape and TTL discipline as
// listenTranscribeKey; keyed by conversation id because Message.CaseID is not
// populated on every write path, while Message.ConversationID always is.
func listenConversationKey(conversationID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:conversation:%s", conversationID)
}
```

After `ListenAIcallIDRemove` (ends line 179), add:

```go
// ListenConversationAIcallIDsGet returns every AIcall id currently listening
// to the given conversation. A SET for the same reason as ListenAIcallIDsGet.
func (h *handler) ListenConversationAIcallIDsGet(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	tmp, err := h.Cache.SMembers(ctx, listenConversationKey(conversationID)).Result()
	if err != nil {
		return nil, err
	}

	res := []uuid.UUID{}
	for _, m := range tmp {
		id := uuid.FromStringOrNil(m)
		if id == uuid.Nil {
			continue
		}
		res = append(res, id)
	}

	return res, nil
}

// ListenConversationAIcallIDAdd registers aicallID as a listener of the
// conversation and (re)arms the key's TTL in one atomic EVAL.
func (h *handler) ListenConversationAIcallIDAdd(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenSetAddExpireScript,
		[]string{listenConversationKey(conversationID)},
		aicallID.String(), listenTTLSeconds(ttl),
	).Err()
}

// ListenConversationAIcallIDRemove drops one listener from the conversation's
// resolver set.
func (h *handler) ListenConversationAIcallIDRemove(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) error {
	return h.Cache.SRem(ctx, listenConversationKey(conversationID), aicallID.String()).Err()
}

// ListenConversationAIcallIDIsMember reports whether aicallID is registered as
// a listener of the conversation. Used by the conversation start's idempotency
// check (design 2026-09-05 §5.1.1 step 0).
func (h *handler) ListenConversationAIcallIDIsMember(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) (bool, error) {
	return h.Cache.SIsMember(ctx, listenConversationKey(conversationID), aicallID.String()).Result()
}

// ListenPendingLen returns the number of buffered, not-yet-evaluated lines.
// The conversation kind's RunListenTurn short-circuits on 0 BEFORE the Case RPC
// and BEFORE the turn counter, so a deferred flush that finds nothing costs
// neither (design 2026-09-05 §5.4).
func (h *handler) ListenPendingLen(ctx context.Context, aicallID uuid.UUID) (int64, error) {
	return h.Cache.LLen(ctx, listenPendingKey(aicallID)).Result()
}
```

In `bin-ai-manager/pkg/cachehandler/main.go`, inside the `CacheHandler` interface after `ListenAIcallIDRemove` (line 51):

```go
	ListenConversationAIcallIDsGet(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	ListenConversationAIcallIDAdd(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error
	ListenConversationAIcallIDRemove(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) error
	ListenConversationAIcallIDIsMember(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) (bool, error)
```

and after `ListenPendingPopAll` (line 54):

```go
	ListenPendingLen(ctx context.Context, aicallID uuid.UUID) (int64, error)
```

- [ ] **Step 5: Add behaviour tests against miniredis, regenerate the mock, run the package tests**

`cachehandler/listen_test.go` already runs the Lua scripts against an in-process Redis (`setupListenTestHandler`, line 65; `Test_listenWritesAreAtomicWithTheirTTL`, line 87). Append a row to that test's table after the `"ListenTurnCountIncr"` row (line 144):

```go
		{
			// VOIP-1470: the conversation resolver reuses listenSetAddExpireScript,
			// so its add must carry its TTL atomically like the transcribe set.
			name: "ListenConversationAIcallIDAdd",
			write: func(h *handler) error {
				return h.ListenConversationAIcallIDAdd(context.Background(), transcribeID, aicallID, ttl)
			},
			key:       listenConversationKey(transcribeID),
			expectTTL: ttl,
		},
```

and add a new test at the end of the file:

```go
// Test_listenConversationResolverAndPendingLen exercises the VOIP-1470
// primitives end to end: membership add/is-member/get/remove and the pending
// list length the conversation turn short-circuits on.
func Test_listenConversationResolverAndPendingLen(t *testing.T) {
	conversationID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")
	otherID := uuid.FromStringOrNil("77777777-7777-8888-9999-aaaaaaaaaaaa")
	ctx := context.Background()

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	if member, err := h.ListenConversationAIcallIDIsMember(ctx, conversationID, aicallID); err != nil || member {
		t.Fatalf("fresh key must not have members. member: %v, err: %v", member, err)
	}
	if err := h.ListenConversationAIcallIDAdd(ctx, conversationID, aicallID, time.Hour); err != nil {
		t.Fatalf("add failed. err: %v", err)
	}
	if err := h.ListenConversationAIcallIDAdd(ctx, conversationID, otherID, time.Hour); err != nil {
		t.Fatalf("second add failed. err: %v", err)
	}
	if member, err := h.ListenConversationAIcallIDIsMember(ctx, conversationID, aicallID); err != nil || !member {
		t.Fatalf("added id must be a member. member: %v, err: %v", member, err)
	}
	got, err := h.ListenConversationAIcallIDsGet(ctx, conversationID)
	if err != nil || len(got) != 2 {
		t.Fatalf("get must return both members. got: %v, err: %v", got, err)
	}
	if err := h.ListenConversationAIcallIDRemove(ctx, conversationID, aicallID); err != nil {
		t.Fatalf("remove failed. err: %v", err)
	}
	got, err = h.ListenConversationAIcallIDsGet(ctx, conversationID)
	if err != nil || len(got) != 1 || got[0] != otherID {
		t.Fatalf("remove must leave only the other member. got: %v, err: %v", got, err)
	}

	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 0 {
		t.Fatalf("empty pending list must be 0. n: %d, err: %v", n, err)
	}
	for i := 0; i < 3; i++ {
		if err := h.ListenPendingPush(ctx, aicallID, fmt.Sprintf("[CUSTOMER] %d", i), time.Hour); err != nil {
			t.Fatalf("push failed. err: %v", err)
		}
	}
	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 3 {
		t.Fatalf("pending list must count pushes. n: %d, err: %v", n, err)
	}
	if _, err := h.ListenPendingPopAll(ctx, aicallID); err != nil {
		t.Fatalf("pop failed. err: %v", err)
	}
	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 0 {
		t.Fatalf("drained pending list must be 0. n: %d, err: %v", n, err)
	}
}
```

Run: `cd bin-ai-manager && go generate ./pkg/cachehandler/... && go test ./pkg/cachehandler/ -v`
Expected: PASS; `pkg/cachehandler/mock_main.go` now contains `ListenConversationAIcallIDsGet` etc.

- [ ] **Step 6: Document the metadata key**

In `bin-ai-manager/docs/domain.md`, in the AIcall metadata table that has the `listen_transcribe_id` row (line 46), add:

```
| `listen_conversation_id` | string (UUID) | The conversation this AIcall is listening to (conversation Cases only). Metadata rather than a column: no event-driven sweep queries by it. An AIcall carries at most one of `listen_transcribe_id` / `listen_conversation_id` |
```

- [ ] **Step 7: Commit**

```bash
git add bin-ai-manager/models/aicall/main.go bin-ai-manager/pkg/cachehandler/listen.go bin-ai-manager/pkg/cachehandler/listen_test.go bin-ai-manager/pkg/cachehandler/main.go bin-ai-manager/pkg/cachehandler/mock_main.go bin-ai-manager/docs/domain.md
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Add MetaKeyListenConversationID and the ai:listen:conversation resolver cache primitives plus ListenPendingLen"
```

---

### Task 4: `kind` label on the listen metrics

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen_test.go:22-27` and its `expected` map
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go` (new `listenKind` type; every `promListenTurnTotal` / `promListenStartTotal` site)
- Modify: `bin-ai-manager/pkg/aicallhandler/listen_trigger.go` (every `promListenStartTotal` site)
- Modify: `bin-ai-manager/pkg/aicallhandler/tool_insight.go:1179`
- Modify: `bin-ai-manager/pkg/aicallhandler/listen_test.go:316-321, 803-812`, `listen_trigger_test.go:1597-1610, 1734-1738`

- [ ] **Step 1: Write the failing metrics registration test**

Replace the body of `Test_listenMetricNames` in `metrics_listen_test.go` lines 22-27 (the `WithLabelValues`/`Add(0)` touches) with:

```go
	promListenStartTotal.WithLabelValues("call", "started")
	promListenSegmentTotal.WithLabelValues("buffered")
	promListenTurnTotal.WithLabelValues("call", "ran")
	promListenNotifyTotal.WithLabelValues("call")
	promListenStopFailedTotal.Add(0)
	promListenMembershipCheckFailedTotal.Add(0)
	promListenConversationSegmentTotal.WithLabelValues("buffered")
	promListenConversationFlushTotal.WithLabelValues("ran")
```

and add to the `expected` map:

```go
		"ai_manager_aicall_listen_conversation_segment_total": false,
		"ai_manager_aicall_listen_conversation_flush_total":   false,
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ -run Test_listenMetricNames -v`
Expected: FAIL to compile (`too many arguments` on `WithLabelValues`, `undefined: promListenConversationSegmentTotal`)

- [ ] **Step 3: Rewrite `metrics_listen.go`**

Replace the three `CounterVec`/`Counter` declarations and add two vecs so the `var (...)` block reads:

```go
var (
	promListenMembershipCheckFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_membership_check_failed_total",
			Help:      "Total number of listen-turn membership checks that errored and degraded to treating the tool call as a real Q&A turn. Near-zero expected; a sustained non-zero rate means Redis is unhealthy, not that anything listen-specific is wrong.",
		},
	)

	promListenStartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_start_total",
			Help:      "Total number of listen-start attempts by kind and outcome. kind: call, conversation, or unknown for the gates that run before the Case's reference type is known. result: started, reused, skipped_not_listenable, skipped_disabled, skipped_confbridge_not_ready, skipped_confbridge_error, skipped_start_locked, failed.",
		},
		[]string{"kind", "result"},
	)

	promListenTurnTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_turn_total",
			Help:      "Total listen evaluation turns by kind and outcome. skipped_locked measured against ran is the direct read on how much LLM spend the debounce is saving -- near-zero skipped_locked means the interval is too short for the traffic. skipped_case_closed is the conversation kind's stop signal.",
		},
		[]string{"kind", "result"},
	)

	promListenSegmentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_segment_total",
			Help:      "Total transcript segments seen by listen intake, by outcome. dropped_unknown dominates by design -- this handler sees every final STT result platform-wide.",
		},
		[]string{"result"},
	)

	promListenConversationSegmentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_conversation_segment_total",
			Help:      "Total conversation messages seen by listen intake, by outcome: buffered, dropped_deleted, dropped_empty, dropped_unknown, dropped_tenant_mismatch, failed. dropped_unknown dominates by design -- this handler sees every conversation message platform-wide; dropped_tenant_mismatch must stay at zero.",
		},
		[]string{"result"},
	)

	promListenConversationFlushTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_conversation_flush_total",
			Help:      "Deferred flush timers for conversation listening, by outcome: ran (won the lock and invoked a turn; read against turn_total skipped_empty), skipped_locked, skipped_scheduled (a timer was already armed for this AIcall on this replica).",
		},
		[]string{"result"},
	)

	promListenStopFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_stop_failed_total",
			Help:      "Total listen transcribe-stop RPCs that failed and fell back to the call-hangup-ends-the-audio-transport backstop.",
		},
	)

	promListenNotifyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_notify_total",
			Help:      "Total number of proactive notifications actually delivered to an agent's Insight panel, by listen kind.",
		},
		[]string{"kind"},
	)
)
```

and register the two new vecs in `init()`:

```go
		promListenConversationSegmentTotal,
		promListenConversationFlushTotal,
```

- [ ] **Step 4: Add `listenKind` to `listen.go`**

After `listenOwnsTranscribeFromMetadata` (ends line 58) add:

```go
// listenKind discriminates which kind of listen session an AIcall holds
// (design 2026-09-05 §5.2.1). An AIcall carries at most one listen pointer.
type listenKind string

const (
	listenKindNone         listenKind = ""
	listenKindCall         listenKind = "call"
	listenKindConversation listenKind = "conversation"
)

// listenKindLabelUnknown is the metric `kind` label value for sites that fire
// before a listen pointer exists or before the Case's reference type is known
// (design 2026-09-05 §5.13).
const listenKindLabelUnknown = "unknown"

// listenConversationIDFromMetadata reads the conversation this AIcall listens
// to, or uuid.Nil when the key is absent or malformed.
func listenConversationIDFromMetadata(c *aicall.AIcall) uuid.UUID {
	if c == nil || c.Metadata == nil {
		return uuid.Nil
	}

	tmp, ok := c.Metadata[aicall.MetaKeyListenConversationID].(string)
	if !ok {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(tmp)
}

// listenKindOf resolves the AIcall's listen kind from its metadata pointers.
func listenKindOf(c *aicall.AIcall) listenKind {
	if listenTranscribeIDFromMetadata(c) != uuid.Nil {
		return listenKindCall
	}
	if listenConversationIDFromMetadata(c) != uuid.Nil {
		return listenKindConversation
	}
	return listenKindNone
}

// label renders the kind as its metric label value; none maps to unknown.
func (k listenKind) label() string {
	if k == listenKindNone {
		return listenKindLabelUnknown
	}
	return string(k)
}
```

Note `listenTranscribeIDFromMetadata` currently dereferences `c.Metadata` without a nil-`c` guard; it is only ever called with a non-nil `c`, leave it.

- [ ] **Step 5: Update every metric call site (mechanical)**

Run: `cd bin-ai-manager && grep -n "promListenStartTotal.WithLabelValues\|promListenTurnTotal.WithLabelValues\|promListenNotifyTotal" pkg/aicallhandler/*.go | grep -v _test`
Expected: 29 lines (listen.go 11, listen_trigger.go 15, tool_insight.go 1, plus the 2 `promListenNotifyTotal` declaration/registration lines in metrics_listen.go, which Step 3 already rewrote). Edit each call site:

`listen_trigger.go`:
- `checkListenEligible` step 1 (`failed` after `aiHandler.Get`), step 2 (`skipped_not_listenable`), step 4 (`skipped_not_listenable` on `ReferenceType != ContactCase`, `failed` on `ContactV1CaseGet`, `skipped_not_listenable` on tenant mismatch): first arg `listenKindLabelUnknown`.
- step 3 `reused` (line 217): first arg `string(listenKindCall)`.
- step 5 `kase.ReferenceType != kmkase.ReferenceTypeCall` (line 253-256) and everything after it in this file (call id parse, `CallV1CallGet` failure, cross-customer call, call status, and every site in `runListenStart`, `startListenTranscribe`, `rollbackListenState`, `waitForConfbridgeReady`): first arg `string(listenKindCall)`. (Task 6 replaces lines 252-256 with the reference-type switch; the label decision here still holds for the `call` arm.)

`listen.go`:
- `RunListenTurn` line 253 (`Get` failed, `skipped_invalid`): `listenKindLabelUnknown`.
- Every other `promListenTurnTotal` site in `RunListenTurn` and `runListenTurnWithLines`: introduce `kindLabel := listenKindOf(c).label()` right after `c` is obtained and use it as the first arg. (Task 5 restructures `RunListenTurn`; do the label pass now so the file compiles.)
- `EventTMTranscriptCreated` line 720 (`skipped_locked`): `string(listenKindCall)`.
- Any `promListenTurnTotal` site in `stopListenByCallID` (final flush): `string(listenKindCall)`.

`tool_insight.go:1179`: `promListenNotifyTotal.WithLabelValues(listenKindOf(c).label()).Inc()`.

- [ ] **Step 6: Update the test sites**

`listen_test.go`:
- `metricDelta` (lines 316-321) gains a `kind` parameter:

```go
// metricDelta reports how much a promListenTurnTotal (kind, label) cell moved across fn.
func metricDelta(t *testing.T, kind string, label string, fn func()) float64 {
	t.Helper()
	before := testutil.ToFloat64(promListenTurnTotal.WithLabelValues(kind, label))
	fn()
	return testutil.ToFloat64(promListenTurnTotal.WithLabelValues(kind, label)) - before
}
```

- `Test_RunListenTurn` (line 328): add a column `expectKind string` to the table struct (after `expectResult`). The row `"missing listen_transcribe_id metadata stops listening"` (lines 379-387, `metadata: map[string]any{}`) gets `expectKind: "unknown"`, because with no listen pointer `listenKindOf(c) == listenKindNone` and the production site emits `unknown` (see the deviation note in the self-review). Every other row leaves it empty. At the call site (line 515) replace `metricDelta(t, tt.expectResult, func() {` with:

```go
			kind := tt.expectKind
			if kind == "" {
				kind = "call"
			}
			delta := metricDelta(t, kind, tt.expectResult, func() {
```

- lines 626 and 1028: `metricDelta(t, "call", "ran", func() {`.
- lines 803-812: `promListenTurnTotal.WithLabelValues("call", "skipped_locked")` (both reads); `promListenSegmentTotal` lines are unchanged.
- Any other `promListenTurnTotal.WithLabelValues(` / `promListenStartTotal.WithLabelValues(` in test files: add `"call"` as the first argument. Run `grep -n -F "promListen" pkg/aicallhandler/*_test.go | grep -F "WithLabelValues("` and expect matches only in `listen_test.go`, `listen_trigger_test.go`, `metrics_listen_test.go`. (`helpers_test.go:350,357` match a plain `WithLabelValues(` grep but belong to `promAIcallInterruptAttemptedTotal` and must NOT be touched.)

`listen_trigger_test.go` lines 1597, 1609, 1610, 1734, 1738: `promListenStartTotal.WithLabelValues("call", ...)`.

- [ ] **Step 7: Build and run the package tests**

Run: `cd bin-ai-manager && go build ./... && go test ./pkg/aicallhandler/ 2>&1 | tail -30`
Expected: PASS. If a test fails on a metric delta, the corresponding production site is still passing one label; fix the site, not the test.

- [ ] **Step 8: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Add the kind label to the listen start, turn and notify metrics, introduce listenKind, and add the conversation segment and flush metric vecs"
```

---

### Task 5: Conversation-aware `RunListenTurn`, prompt, cleanup, terminate, idle expiry

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (prompt constant after `ListenTurnSystemPrompt`, line 329)
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go` (`RunListenTurn`, `buildListenTurnMessages`, `buildListenTranscriptBlock`, `clearListenState`)
- Modify: `bin-ai-manager/pkg/aicallhandler/process.go:63`
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:492-494`
- Test: `bin-ai-manager/pkg/aicallhandler/listen_test.go`, `start_test.go` (or a new `listen_conversation_test.go` for the conversation rows)

- [ ] **Step 1: Write the failing turn tests**

Create `bin-ai-manager/pkg/aicallhandler/listen_conversation_test.go` with the fixtures every later task in this file extends:

```go
package aicallhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	kmkase "monorepo/bin-contact-manager/models/kase"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

var (
	lcConversationID = uuid.FromStringOrNil("44440000-0000-4000-8000-000000000001")
)

// listeningConversationAIcall is an AIcall that passes every conversation-kind
// RunListenTurn precondition. ltAIcallID/ltCustomerID/ltCaseID come from
// listen_trigger_test.go.
func listeningConversationAIcall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: ltAIcallID, CustomerID: ltCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   ltAIID,
		ReferenceType:  aicall.ReferenceTypeContactCase,
		ReferenceID:    ltCaseID,
		Status:         aicall.StatusProgressing,
		PipecatcallID:  uuid.FromStringOrNil("44440000-0000-4000-8000-0000000000aa"),
		Metadata: map[string]any{
			aicall.MetaKeyListenConversationID: lcConversationID.String(),
		},
	}
}

func openConversationCase() *kmkase.Case {
	return &kmkase.Case{
		ID:            ltCaseID,
		CustomerID:    ltCustomerID,
		ReferenceType: kmkase.ReferenceTypeConversationMessage,
		ReferenceID:   lcConversationID.String(),
		Status:        kmkase.StatusOpen,
	}
}

func Test_listenKindOf(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect listenKind
		label  string
	}{
		{"nil metadata", &aicall.AIcall{}, listenKindNone, "unknown"},
		{"transcribe pointer", listeningAIcall(), listenKindCall, "call"},
		{"conversation pointer", listeningConversationAIcall(), listenKindConversation, "conversation"},
		{"malformed conversation pointer", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenConversationID: "nope"}}, listenKindNone, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listenKindOf(tt.c)
			if got != tt.expect {
				t.Errorf("kind mismatch. expected: %q, got: %q", tt.expect, got)
			}
			if got.label() != tt.label {
				t.Errorf("label mismatch. expected: %q, got: %q", tt.label, got.label())
			}
		})
	}
}

// Test_RunListenTurn_Conversation covers the conversation-only gates that sit
// between the predicate and the turn counter (design 2026-09-05 §5.5.1, §5.5.2).
// Not parallel: metric deltas.
func Test_RunListenTurn_Conversation(t *testing.T) {
	tests := []struct {
		name               string
		callKind           bool
		conversationFlag   bool
		pendingLen         int64
		pendingLenErr      error
		responseCase       *kmkase.Case
		responseCaseErr    error
		expectStop         bool
		expectCaseGet      bool
		expectCounter      bool
		expectResultLabel  string
	}{
		{
			name:              "conversation sub-flag off stops the session without a Case RPC",
			conversationFlag:  false,
			expectStop:        true,
			expectResultLabel: "skipped_disabled",
		},
		{
			name:              "empty pending buffer short-circuits before the Case RPC and the counter",
			conversationFlag:  true,
			pendingLen:        0,
			expectResultLabel: "skipped_empty",
		},
		{
			name:              "LLEN error is tolerated and the turn proceeds to the Case check",
			conversationFlag:  true,
			pendingLenErr:     fmt.Errorf("redis down"),
			responseCase:      openConversationCase(),
			expectCaseGet:     true,
			expectCounter:     true,
			expectResultLabel: "skipped_empty", // ListenPendingPopAll returns nothing in this harness, so the turn ends as skipped_empty after the counter
		},
		{
			name:              "closed Case stops listening",
			conversationFlag:  true,
			pendingLen:        2,
			responseCase:      &kmkase.Case{ID: ltCaseID, CustomerID: ltCustomerID, ReferenceType: kmkase.ReferenceTypeConversationMessage, ReferenceID: lcConversationID.String(), Status: kmkase.StatusClosed},
			expectCaseGet:     true,
			expectStop:        true,
			expectResultLabel: "skipped_case_closed",
		},
		{
			name:              "Case RPC failure is metered as failed and nothing is popped or counted",
			conversationFlag:  true,
			pendingLen:        2,
			responseCaseErr:   fmt.Errorf("rpc timeout"),
			expectCaseGet:     true,
			expectResultLabel: "failed",
		},
		{
			// Design §7 item 4: a CALL-kind AIcall in the same table must be
			// untouched by the conversation gates -- no LLEN, no Case RPC, and
			// the conversation sub-flag being OFF must not stop it.
			name:              "call kind ignores the conversation sub-flag and never calls LLEN or the Case RPC",
			callKind:          true,
			conversationFlag:  false,
			expectCounter:     true,
			expectResultLabel: "skipped_empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			config.SetAIcallListenEnabledForTest(true)
			config.SetAIcallListenConversationEnabledForTest(tt.conversationFlag)

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newListenTurnHarness(mc)
			ctx := context.Background()

			c := listeningConversationAIcall()
			kindLabel := "conversation"
			if tt.callKind {
				c = listeningAIcall()
				kindLabel = "call"
			}
			m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)

			if tt.conversationFlag && !tt.callKind {
				m.cache.EXPECT().ListenPendingLen(ctx, ltAIcallID).Return(tt.pendingLen, tt.pendingLenErr)
			} else {
				m.cache.EXPECT().ListenPendingLen(gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectCaseGet {
				m.req.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(tt.responseCase, tt.responseCaseErr)
			} else {
				m.req.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectCounter {
				m.cache.EXPECT().ListenTurnCountIncr(ctx, ltAIcallID, gomock.Any()).Return(int64(1), nil)
				m.cache.EXPECT().ListenPendingPopAll(ctx, ltAIcallID).Return([]string{}, nil)
			} else {
				m.cache.EXPECT().ListenTurnCountIncr(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectStop {
				// stopListening -> clearListenState for the conversation kind:
				// SREM the resolver, clear the keys, strip the metadata key.
				m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, ltAIcallID).Return(nil)
				m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
						meta, _ := fields[aicall.FieldMetadata].(map[string]any)
						if _, still := meta[aicall.MetaKeyListenConversationID]; still {
							t.Errorf("clearListenState must strip listen_conversation_id")
						}
						return nil
					})
			} else {
				m.cache.EXPECT().ListenStateClear(gomock.Any(), gomock.Any()).Times(0)
			}

			got := metricDelta(t, kindLabel, tt.expectResultLabel, func() { m.h.RunListenTurn(ctx, ltAIcallID) })
			if got != 1 {
				t.Errorf("result %q must be metered exactly once. got: %v", tt.expectResultLabel, got)
			}
		})
	}
}

func Test_buildListenTurnMessages_ConversationKind(t *testing.T) {
	config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningConversationAIcall()
	m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return([]*message.Message{}, nil)

	res, err := m.h.buildListenTurnMessages(ctx, c, []string{"[CUSTOMER] hi"}, []string{"[CUSTOMER] hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sawConversationPrompt, sawCallPrompt bool
	for _, row := range res {
		content, _ := row["content"].(string)
		if content == ListenTurnConversationSystemPrompt {
			sawConversationPrompt = true
		}
		if content == ListenTurnSystemPrompt {
			sawCallPrompt = true
		}
	}
	if !sawConversationPrompt || sawCallPrompt {
		t.Errorf("conversation kind must use ListenTurnConversationSystemPrompt only. conversation: %v, call: %v", sawConversationPrompt, sawCallPrompt)
	}

	last, _ := res[len(res)-1]["content"].(string)
	if !strings.HasPrefix(last, "Conversation so far:\n") {
		t.Errorf("conversation transcript block must start with the conversation header. got: %q", last)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ -run 'Test_listenKindOf|Test_RunListenTurn_Conversation|Test_buildListenTurnMessages_ConversationKind' -v 2>&1 | tail -20`
Expected: `Test_listenKindOf` passes (Task 4 added the helper); the other two FAIL (`undefined: ListenTurnConversationSystemPrompt`, and/or unexpected mock calls).

- [ ] **Step 3: Add the conversation prompt**

In `bin-ai-manager/pkg/aicallhandler/main.go`, after the `ListenTurnSystemPrompt` constant (line 329, inside the same `const (...)` block), add:

```go
	// ListenTurnConversationSystemPrompt is the conversation-kind counterpart of
	// ListenTurnSystemPrompt (design 2026-09-05 §5.5.3): same mechanics, same
	// rules, only the framing differs. Business conditions still come from the
	// customer's own init_prompt.
	ListenTurnConversationSystemPrompt = `You are silently monitoring a live messaging conversation between a human agent and a customer (SMS, chat, or similar). You are NOT talking to anyone right now.

Below you will see a rolling window of the messages exchanged so far, tagged by sender. Lines after the "--- NEW SINCE YOUR LAST CHECK ---" marker are what you have not evaluated yet; everything before it you have already considered on a previous check. Lines tagged [AGENT] were written by the agent you are assisting (or an automated reply on their behalf); never alert the agent about their own messages.

Your task on each check:
1. Read the new lines in the context of the conversation so far.
2. Decide whether the instructions in your configured prompt warrant alerting the human agent RIGHT NOW.
3. If and only if they do, call the notify_agent tool with one or two sentences written for a busy human handling several conversations.

CRITICAL RULES:
- Saying nothing is the correct and expected outcome for most checks. Do not manufacture something to say.
- notify_agent is the ONLY way to reach the agent. Any text you produce instead of a tool call is discarded and nobody will ever see it.
- Never repeat a notification you already sent on this conversation. Check the conversation above before notifying.
- Do not summarize the conversation, do not narrate what is happening, and do not greet anyone. You are not a participant.
- Do not use other tools unless answering the alert genuinely requires information the messages do not contain.`
```

- [ ] **Step 4: Make `buildListenTurnMessages` and the transcript header kind-aware**

In `listen.go` `buildListenTurnMessages`, replace the block that appends `ListenTurnSystemPrompt` with:

```go
	turnPrompt := ListenTurnSystemPrompt
	if listenKindOf(c) == listenKindConversation {
		turnPrompt = ListenTurnConversationSystemPrompt
	}
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": turnPrompt,
	})
```

and replace the final `buildListenTranscriptBlock(window, newLines)` call with:

```go
	res = append(res, map[string]any{
		"role":    string(message.RoleUser),
		"content": buildListenTranscriptBlock(listenTranscriptHeader(listenKindOf(c)), window, newLines),
	})
```

Then replace `buildListenTranscriptBlock` with:

```go
const (
	listenTranscriptHeaderCall         = "Live call transcript so far:"
	listenTranscriptHeaderConversation = "Conversation so far:"
)

// listenTranscriptHeader picks the transcript block's first line by kind.
func listenTranscriptHeader(kind listenKind) string {
	if kind == listenKindConversation {
		return listenTranscriptHeaderConversation
	}
	return listenTranscriptHeaderCall
}

// buildListenTranscriptBlock renders the rolling window plus the new lines
// under a kind-specific header. The header is a parameter (rather than a
// second wrapper function) because the only caller is buildListenTurnMessages
// and an unused wrapper would fail golangci-lint's unused check.
func buildListenTranscriptBlock(header string, window []string, newLines []string) string {
	seen := window
	if len(newLines) > 0 && len(window) >= len(newLines) {
		seen = window[:len(window)-len(newLines)]
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	for _, line := range seen {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(listenTranscriptNewMarker)
	sb.WriteString("\n")
	for _, line := range newLines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}
```

- [ ] **Step 5: Restructure `RunListenTurn`**

Replace `RunListenTurn` (the function spans lines 244-299; line 300 is blank and `runListenTurnWithLines`'s doc comment starts at 301 and must be kept) with:

```go
func (h *aicallHandler) RunListenTurn(ctx context.Context, aicallID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RunListenTurn",
		"aicall_id": aicallID,
	})

	c, err := h.Get(ctx, aicallID)
	if err != nil {
		promListenTurnTotal.WithLabelValues(listenKindLabelUnknown, "skipped_invalid").Inc()
		return
	}
	kind := listenKindOf(c)
	kindLabel := kind.label()

	if !config.Get().AIcallListenEnabled {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_disabled").Inc()
		return
	}

	// Scoped to the conversation kind so a disabled conversation variant can
	// never stop a call listen (design 2026-09-05 §5.5.1).
	if kind == listenKindConversation && !config.Get().AIcallListenConversationEnabled {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_disabled").Inc()
		return
	}

	if c.Status != aicall.StatusProgressing ||
		c.ReferenceType != aicall.ReferenceTypeContactCase ||
		kind == listenKindNone {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_invalid").Inc()
		return
	}

	if kind == listenKindConversation {
		// Empty short-circuit BEFORE the Case RPC and BEFORE the turn counter:
		// a deferred flush that finds nothing must cost neither (§5.4). The
		// check is non-atomic with the pop below; a line landing in between only
		// means one turn is counted normally.
		pending, errLen := h.cache.ListenPendingLen(ctx, aicallID)
		if errLen != nil {
			log.Warnf("Could not read the pending buffer length; proceeding as non-empty. err: %v", errLen)
		} else if pending == 0 {
			promListenTurnTotal.WithLabelValues(kindLabel, "skipped_empty").Inc()
			return
		}

		// Turn-time Case status check: contact-manager publishes nothing on
		// Close, so this is the conversation kind's primary stop signal (§5.5.2).
		kase, errCase := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
		if errCase != nil {
			log.Errorf("Could not get the case for the listen turn. err: %v", errCase)
			promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
			return
		}
		if kase.Status == kmkase.StatusClosed {
			h.stopListening(ctx, c)
			promListenTurnTotal.WithLabelValues(kindLabel, "skipped_case_closed").Inc()
			return
		}
	}

	turns, errCount := h.cache.ListenTurnCountIncr(ctx, aicallID, listenBufferTTL())
	if errCount != nil {
		log.Warnf("Could not increment the listen turn counter. err: %v", errCount)
	} else if turns > int64(config.Get().AIcallListenMaxTurnsPerAIcall) {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_cap").Inc()
		return
	}

	lines, err := h.cache.ListenPendingPopAll(ctx, aicallID)
	if err != nil {
		log.Errorf("Could not drain the pending buffer. err: %v", err)
		promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
		return
	}
	if len(lines) == 0 {
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_empty").Inc()
		return
	}

	h.runListenTurnWithLines(ctx, c, lines)
}
```

Add `kmkase "monorepo/bin-contact-manager/models/kase"` to `listen.go`'s imports. Keep the existing doc comment above the function; append one sentence: "Conversation-kind AIcalls additionally pass an empty-buffer short-circuit and a turn-time Case status check before anything is counted or popped."

- [ ] **Step 6: Extend `clearListenState`**

In `clearListenState`, after the transcribe `ListenAIcallIDRemove` block and before `ListenStateClear`, add:

```go
	if conversationID := listenConversationIDFromMetadata(c); conversationID != uuid.Nil {
		if errRem := h.cache.ListenConversationAIcallIDRemove(ctx, conversationID, c.ID); errRem != nil {
			log.Warnf("Could not remove the conversation listen resolver membership. err: %v", errRem)
		}
	}
```

and extend the metadata strip loop's condition to:

```go
		if k == aicall.MetaKeyListenTranscribeID || k == aicall.MetaKeyListenOwnsTranscribe || k == aicall.MetaKeyListenConversationID {
			continue
		}
```

`stopListening` needs no change: with no `listen_owns_transcribe` it already falls straight through to `clearListenState`.

- [ ] **Step 7: Widen the terminate gate and the idle-expiry cleanup**

`process.go:63`: replace the condition with a pure helper so the gate is unit-testable without driving the whole `ProcessTerminate` flow:

```go
	if listenTerminateNeedsStop(tmp) {
		h.stopListening(ctx, tmp)
	}
```

and add to `listen.go` (after `listenKind.label()`):

```go
// listenTerminateNeedsStop is ProcessTerminate's gate: only a contact_case
// AIcall can be listening, and it is listening if either the call-kind column
// or either metadata pointer says so (design 2026-09-05 §5.7; the ListenCallID
// clause is kept as belt-and-braces for call rows).
func listenTerminateNeedsStop(c *aicall.AIcall) bool {
	if c == nil || c.ReferenceType != aicall.ReferenceTypeContactCase {
		return false
	}
	return c.ListenCallID != uuid.Nil || listenKindOf(c) != listenKindNone
}
```

`start.go:492-494` (inside `startReferenceTypeContactCase`'s idle-expired branch; do NOT touch the similar block at `:282`, which is the chatbot path and can never hold a listen AIcall):

```go
			if _, errEnd := h.UpdateStatus(ctx, existing.ID, aicall.StatusTerminated); errEnd != nil {
				log.Warnf("Could not terminate idle AIcall: %v", errEnd)
			}
			// UpdateStatus never clears listen state (design 2026-09-05 §5.7);
			// release the resolver membership and buffers of an idle-expired
			// listener here so they do not linger until their TTLs.
			if listenKindOf(existing) != listenKindNone {
				h.stopListening(ctx, existing)
			}
```

- [ ] **Step 8: Add the terminate/idle tests**

Append to `listen_conversation_test.go`:

```go
// Test_ProcessTerminate_ConversationListenGate pins process.go's widened gate:
// a conversation listener (ListenCallID == Nil, metadata pointer set) must
// still have its state cleared on terminate.
func Test_clearListenState_ConversationKind(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningConversationAIcall()

	rem := m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, ltAIcallID).Return(nil)
	clear := m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
	update := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
	gomock.InOrder(rem, clear, update)
	m.cache.EXPECT().ListenAIcallIDRemove(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.req.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), gomock.Any()).Times(0)

	m.h.stopListening(ctx, c)
}
```

For `process.go`, the gate is now the pure `listenTerminateNeedsStop`; pin it with a table in `listen_conversation_test.go` (the existing `Test_ProcessTerminate` at `process_test.go:96` drives the whole terminate flow and has no listen rows; it stays green because none of its AIcalls carry listen state):

```go
func Test_listenTerminateNeedsStop(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect bool
	}{
		{"nil", nil, false},
		{"task reference is never listening", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeTask, ListenCallID: ltCallID}, false},
		{"contact_case with no listen state", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase}, false},
		{"contact_case with listen_call_id column", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase, ListenCallID: ltCallID}, true},
		{"contact_case with transcribe pointer", listeningAIcall(), true},
		{"contact_case with conversation pointer", listeningConversationAIcall(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenTerminateNeedsStop(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}
```

For `start.go`, extend `Test_startReferenceTypeContactCase` (`start_test.go:3771`):

1. Add `cache *cachehandler.MockCacheHandler` to its local `mocks` struct (lines 3782-3788), construct it in the loop (`cache: cachehandler.NewMockCacheHandler(mc),` at lines 4627-4633) and wire it into the handler literal (`cache: m.cache,` at lines 4635-4641). Add `"monorepo/bin-ai-manager/pkg/cachehandler"` to the file's imports if absent.
2. Copy the row `"duplicate key — existing idle-expired (not yet terminated) — terminates and retries without rate limit"` (lines 4443-4526) as a new row named `"duplicate key — existing idle-expired conversation listener — terminates, clears listen state, retries"`, with these differences only: give `existingIdle` the metadata `Metadata: map[string]any{aicall.MetaKeyListenConversationID: lcConversationID.String()}` and `ReferenceType: aicall.ReferenceTypeContactCase`, and directly after the `m.notify.EXPECT().PublishWebhookEvent(ctx, terminatedAIcall.CustomerID, aicall.EventTypeStatusTerminated, terminatedAIcall)` line add:

```go
				// VOIP-1470: idle-expiry must also release the listener's state
				// (design 2026-09-05 §5.7). stopListening -> clearListenState.
				m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, existingIdleID).Return(nil)
				m.cache.EXPECT().ListenStateClear(ctx, existingIdleID).Return(nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, existingIdleID, gomock.Any()).Return(nil)
```

Use fresh UUIDs for the row's ids: replace the `55000000-` prefix with `56000000-` in EVERY id the cloned row carries, in `ai`/`assistanceID`/`activeflowID`/`referenceID`, inside `mockSetup`, AND in the row's `expectRes` block (the original's `expectRes` at lines 4517-4523 references `55000000-0008-...` and `55000000-0002-...`; a clone that forgets it fails its result assertion). The ORIGINAL row (no metadata) is the required "non-listening row does not call stopListening" proof: with `cache` now a strict gomock mock, any unexpected `ListenStateClear` call there fails the test.

- [ ] **Step 9: Run the package tests**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ 2>&1 | tail -30`
Expected: PASS, including the pre-existing `Test_RunListenTurn` rows (call kind is unchanged: no `ListenPendingLen`, no `ContactV1CaseGet`).

- [ ] **Step 10: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Make RunListenTurn conversation-aware (scoped sub-flag, empty short-circuit, turn-time Case status check), add the conversation listen prompt and transcript header, clear the conversation resolver on stop, widen the terminate gate and clean up idle-expired listeners"
```

---

### Task 6: Trigger branch (`checkListenEligibleConversation`, `startListenConversation`)

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/listen_conversation.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/listen_trigger.go:252-256`
- Modify: `bin-ai-manager/pkg/aicallhandler/listen_trigger_test.go:140-150` (existing row), plus new rows
- Test: `bin-ai-manager/pkg/aicallhandler/listen_conversation_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `listen_conversation_test.go`:

```go
// Test_startListenConversation covers design 2026-09-05 §5.1.1 exit by exit.
func Test_startListenConversation(t *testing.T) {
	tests := []struct {
		name         string
		metadata     map[string]any
		isMember     bool
		isMemberErr  error
		addErr       error
		updateErr    error
		expectResult string
		expectAdd    bool
		expectUpdate bool
		expectRemove bool
	}{
		{
			name:         "pointer set and member present is reused with zero writes",
			metadata:     map[string]any{aicall.MetaKeyListenConversationID: lcConversationID.String()},
			isMember:     true,
			expectResult: "reused",
		},
		{
			name:         "pointer set but membership missing re-adds and rewrites",
			metadata:     map[string]any{aicall.MetaKeyListenConversationID: lcConversationID.String()},
			isMember:     false,
			expectResult: "started",
			expectAdd:    true,
			expectUpdate: true,
		},
		{
			name:         "SISMEMBER error degrades to a fresh start",
			metadata:     map[string]any{aicall.MetaKeyListenConversationID: lcConversationID.String()},
			isMemberErr:  fmt.Errorf("redis down"),
			expectResult: "started",
			expectAdd:    true,
			expectUpdate: true,
		},
		{
			name:         "pointer absent starts",
			expectResult: "started",
			expectAdd:    true,
			expectUpdate: true,
		},
		{
			name:         "SADD failure is failed with no DB write",
			addErr:       fmt.Errorf("redis down"),
			expectResult: "failed",
			expectAdd:    true,
		},
		{
			name:         "DB failure rolls the SADD back",
			updateErr:    fmt.Errorf("db down"),
			expectResult: "failed",
			expectAdd:    true,
			expectUpdate: true,
			expectRemove: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newListenTurnHarness(mc)
			ctx := context.Background()

			c := listeningConversationAIcall()
			c.Metadata = tt.metadata

			if tt.metadata != nil {
				m.cache.EXPECT().ListenConversationAIcallIDIsMember(ctx, lcConversationID, ltAIcallID).Return(tt.isMember, tt.isMemberErr)
			}
			if tt.expectAdd {
				m.cache.EXPECT().ListenConversationAIcallIDAdd(ctx, lcConversationID, ltAIcallID, listenResolverTTL).Return(tt.addErr)
			} else {
				m.cache.EXPECT().ListenConversationAIcallIDAdd(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectUpdate {
				m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
						meta, _ := fields[aicall.FieldMetadata].(map[string]any)
						if meta[aicall.MetaKeyListenConversationID] != lcConversationID.String() {
							t.Errorf("metadata must carry listen_conversation_id. got: %v", meta)
						}
						if _, has := fields[aicall.FieldListenCallID]; has {
							t.Errorf("a conversation start must never write listen_call_id")
						}
						return tt.updateErr
					})
			} else {
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectRemove {
				m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, ltAIcallID).Return(nil)
			} else {
				m.cache.EXPECT().ListenConversationAIcallIDRemove(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			before := testutil.ToFloat64(promListenStartTotal.WithLabelValues("conversation", tt.expectResult))
			res := m.h.startListenConversation(ctx, c, lcConversationID)
			if res != tt.expectResult {
				t.Errorf("result mismatch. expected: %s, got: %s", tt.expectResult, res)
			}
			if got := testutil.ToFloat64(promListenStartTotal.WithLabelValues("conversation", tt.expectResult)) - before; got != 1 {
				t.Errorf("result %q must be metered exactly once. got: %v", tt.expectResult, got)
			}
		})
	}
}

// Test_checkListenEligible_ConversationBranch pins the step-5 switch: a
// conversation Case runs the inline start and returns proceed=false so
// ProcessListen never spawns the call-only runListenStart.
func Test_checkListenEligible_ConversationBranch(t *testing.T) {
	tests := []struct {
		name             string
		conversationFlag bool
		referenceID      string
		expectStart      bool
		expectLabel      string
	}{
		{"sub-flag off is skipped_disabled", false, lcConversationID.String(), false, "skipped_disabled"},
		{"empty reference id is skipped_not_listenable", true, "", false, "skipped_not_listenable"},
		{"garbage reference id is skipped_not_listenable", true, "not-a-uuid", false, "skipped_not_listenable"},
		{"valid conversation id starts inline", true, lcConversationID.String(), true, "started"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			config.SetAIcallListenEnabledForTest(true)
			config.SetAIcallListenConversationEnabledForTest(tt.conversationFlag)

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newListenTurnHarness(mc)
			mockAI := aihandlerMockForTrigger(mc)
			m.h.aiHandler = mockAI
			ctx := context.Background()

			c := listenEligibleAIcall()
			mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{Identity: commonidentity.Identity{ID: ltAIID}, Type: ai.TypeInsight}, nil)
			kase := openConversationCase()
			kase.ReferenceID = tt.referenceID
			m.req.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(kase, nil)
			m.req.EXPECT().CallV1CallGet(gomock.Any(), gomock.Any()).Times(0)
			m.req.EXPECT().TranscribeV1TranscribeStart(
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			).Times(0)

			if tt.expectStart {
				m.cache.EXPECT().ListenConversationAIcallIDAdd(ctx, lcConversationID, ltAIcallID, listenResolverTTL).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
			} else {
				m.cache.EXPECT().ListenConversationAIcallIDAdd(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			before := testutil.ToFloat64(promListenStartTotal.WithLabelValues("conversation", tt.expectLabel))
			a, k, callID, call, proceed, err := m.h.checkListenEligible(ctx, c)
			if err != nil {
				t.Fatalf("checkListenEligible must never return an error. got: %v", err)
			}
			if proceed || a != nil || k != nil || call != nil || callID != uuid.Nil {
				t.Errorf("the conversation branch must return proceed=false with zero values. proceed: %v", proceed)
			}
			if got := testutil.ToFloat64(promListenStartTotal.WithLabelValues("conversation", tt.expectLabel)) - before; got != 1 {
				t.Errorf("label %q must be metered exactly once. got: %v", tt.expectLabel, got)
			}
		})
	}
}
```

Also add the `ProcessListen`-level test design §7 item 5c asks for (the call-only `runListenStart` goroutine must never be spawned on the conversation branch; mirrors `listen_trigger_test.go:380-418`):

```go
// Test_ProcessListen_ConversationBranchNeverSpawnsRunListenStart pins that the
// conversation branch completes inline and ProcessListen never spawns the
// call-only async stage.
func Test_ProcessListen_ConversationBranchNeverSpawnsRunListenStart(t *testing.T) {
	config.SetListenDefaultsForTest()
	config.SetAIcallListenEnabledForTest(true)
	config.SetAIcallListenConversationEnabledForTest(true)

	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	mockAI := aihandlerMockForTrigger(mc)
	m.h.aiHandler = mockAI
	ctx := context.Background()

	c := listenEligibleAIcall()
	m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil).Times(2) // ProcessListen's Get, then startListenConversation's re-read
	mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{Identity: commonidentity.Identity{ID: ltAIID}, Type: ai.TypeInsight}, nil)
	m.req.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(openConversationCase(), nil)
	m.cache.EXPECT().ListenConversationAIcallIDAdd(ctx, lcConversationID, ltAIcallID, listenResolverTTL).Return(nil)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)

	var hookCalls int32
	var mu sync.Mutex
	m.h.runListenStartHook = func(context.Context, *ai.AI, *aicall.AIcall, *kmkase.Case, uuid.UUID, *cmcall.Call) {
		mu.Lock()
		hookCalls++
		mu.Unlock()
	}

	res, err := m.h.ProcessListen(ctx, ltAIcallID)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if res != c {
		t.Errorf("ProcessListen must return the AIcall it fetched")
	}

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if hookCalls != 0 {
		t.Errorf("runListenStart must never run on the conversation branch. calls: %d", hookCalls)
	}
}
```

This needs `"sync"`, `"time"` and `cmcall "monorepo/bin-call-manager/models/call"` in the file's imports (Task 7 adds `"time"` too; add it once).

Add the small helper the test above uses, at the bottom of `listen_conversation_test.go` (it keeps the file self-contained; `aihandler` import is needed):

```go
func aihandlerMockForTrigger(mc *gomock.Controller) *aihandler.MockAIHandler {
	return aihandler.NewMockAIHandler(mc)
}
```

and add `"monorepo/bin-ai-manager/models/ai"`, `"monorepo/bin-ai-manager/pkg/aihandler"` and `"github.com/prometheus/client_golang/prometheus/testutil"` to the file's imports (Task 5 deliberately did not import `testutil`; this task is its first user).

Also, in `listen_trigger_test.go` update the existing row at lines 140-150 (the row's closing `},` is line 150). Its Case uses `ReferenceType: "conversation_message"`, which now takes the conversation branch instead of the "not listenable" arm. Change that row to:

```go
		{
			name:         "case reference type is neither call nor conversation_message",
			flagEnabled:  true,
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: &kmkase.Case{
				ID: ltCaseID, CustomerID: ltCustomerID,
				ReferenceType: "email_thread", ReferenceID: ltCallID.String(),
			},
			expectCaseGet: true,
		},
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ -run 'Test_startListenConversation|Test_checkListenEligible' -v 2>&1 | tail -20`
Expected: FAIL to compile (`undefined: startListenConversation`).

- [ ] **Step 3: Create `listen_conversation.go` with the trigger half**

```go
package aicallhandler

import (
	"context"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	kmkase "monorepo/bin-contact-manager/models/kase"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// checkListenEligibleConversation is checkListenEligible's step-5 branch for a
// Case created from a messaging conversation (design 2026-09-05 §5.1, §5.1.1).
//
// It runs the whole start INLINE. There is no external session to create, so
// none of the call branch's goroutine, confbridge poll or start lock applies.
// The caller returns proceed=false regardless of the outcome; every outcome is
// metered here under kind="conversation".
func (h *aicallHandler) checkListenEligibleConversation(ctx context.Context, c *aicall.AIcall, kase *kmkase.Case) {
	if !config.Get().AIcallListenConversationEnabled {
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_disabled").Inc()
		return
	}

	conversationID := uuid.FromStringOrNil(kase.ReferenceID)
	if conversationID == uuid.Nil {
		// flow-manager's case_create may legitimately store an empty ReferenceID.
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_not_listenable").Inc()
		return
	}

	h.startListenConversation(ctx, c, conversationID)
}

// startListenConversation registers c as a listener of conversationID: an
// idempotency check, then two idempotent writes (resolver SADD, metadata
// pointer). Returns the metered result for tests.
func (h *aicallHandler) startListenConversation(ctx context.Context, c *aicall.AIcall, conversationID uuid.UUID) string {
	log := logrus.WithFields(logrus.Fields{
		"func":            "startListenConversation",
		"aicall_id":       c.ID,
		"conversation_id": conversationID,
	})
	kindLabel := string(listenKindConversation)

	// Step 0: pointer already set AND resolver membership present -> reused.
	// Pointer set but membership missing (Redis flush) falls through: the SADD
	// below re-registers and the metadata rewrite is a no-op.
	if listenConversationIDFromMetadata(c) == conversationID {
		member, errMember := h.cache.ListenConversationAIcallIDIsMember(ctx, conversationID, c.ID)
		if errMember != nil {
			log.Warnf("Could not check the conversation listen membership; starting fresh. err: %v", errMember)
		} else if member {
			promListenStartTotal.WithLabelValues(kindLabel, "reused").Inc()
			return "reused"
		}
	}

	// Step 1: resolver membership, before the DB pointer, so nothing can route a
	// message at an AIcall the resolver does not yet know.
	if errAdd := h.cache.ListenConversationAIcallIDAdd(ctx, conversationID, c.ID, listenResolverTTL); errAdd != nil {
		log.Errorf("Could not add the conversation listen resolver membership. err: %v", errAdd)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}

	// Step 2: persist the pointer. Re-read the row first so a concurrent
	// metadata write is merged rather than clobbered (same shape as
	// UpdateListenState for the call kind).
	cur, errGet := h.db.AIcallGet(ctx, c.ID)
	if errGet != nil {
		log.Errorf("Could not re-read the aicall before writing the listen pointer. err: %v", errGet)
		h.rollbackListenConversation(ctx, conversationID, c.ID)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}
	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyListenConversationID] = conversationID.String()

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, c.ID, map[aicall.Field]any{
		aicall.FieldMetadata: metadata,
	}); errUpdate != nil {
		log.Errorf("Could not write the conversation listen pointer. err: %v", errUpdate)
		h.rollbackListenConversation(ctx, conversationID, c.ID)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}

	promListenStartTotal.WithLabelValues(kindLabel, "started").Inc()
	return "started"
}

// rollbackListenConversation is the best-effort SREM after a failed start. If it
// fails too, the entry expires with listenResolverTTL and the first line that
// reaches RunListenTurn's predicate clears it.
func (h *aicallHandler) rollbackListenConversation(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) {
	if errRem := h.cache.ListenConversationAIcallIDRemove(ctx, conversationID, aicallID); errRem != nil {
		logrus.WithFields(logrus.Fields{"func": "rollbackListenConversation", "aicall_id": aicallID}).
			Warnf("Could not roll back the conversation listen resolver membership. err: %v", errRem)
	}
}
```

Note for the test table in Step 1: the "DB failure rolls the SADD back" row and the "pointer absent starts" rows expect `AIcallGet` only when `expectUpdate` is true; the "SADD failure" row must not call `AIcallGet`. That matches the code above (the re-read happens after the SADD). The `expectUpdate` rows return `c` itself from `AIcallGet`, so `cur.Metadata` is the test's metadata.

- [ ] **Step 4: Insert the step-5 switch in `listen_trigger.go`**

Replace lines 252-256 (the existing `// Step 5: reference typing.` comment at 252 plus the four code lines below it, as left by Task 4):

```go
	// Step 5: reference typing.
	if kase.ReferenceType != kmkase.ReferenceTypeCall {
		promListenStartTotal.WithLabelValues(string(listenKindCall), "skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
```

with:

```go
	// Step 5: reference typing. The conversation branch runs its whole start
	// inline and never proceeds to steps 6-8, which are call-only (design
	// 2026-09-05 §5.1).
	switch kase.ReferenceType {
	case kmkase.ReferenceTypeCall:
		// steps 6-8 below
	case kmkase.ReferenceTypeConversationMessage:
		h.checkListenEligibleConversation(ctx, c, kase)
		return nil, nil, uuid.Nil, nil, false, nil
	default:
		promListenStartTotal.WithLabelValues(listenKindLabelUnknown, "skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
```

- [ ] **Step 5: Run the package tests**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ 2>&1 | tail -30`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Branch the listen trigger on conversation Cases and add the inline, idempotent startListenConversation"
```

---

### Task 7: Intake, deferred flush, subscribe binding

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/listen_conversation.go` (intake half)
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (interface method after `EventTMTranscriptCreated`; struct fields; constructor)
- Modify: `bin-ai-manager/pkg/subscribehandler/main.go` (const, pattern, switch), create `conversationmanager.go`, modify `binding_golden_test.go`
- Modify: `bin-ai-manager/docs/architecture.md` (events section, near line 100)
- Test: `listen_conversation_test.go`, `subscribehandler/conversationmanager_test.go` (new, if the package has per-handler tests; otherwise the golden test is the coverage)

- [ ] **Step 1: Write the failing intake tests**

Append to `listen_conversation_test.go`:

```go
func Test_conversationMessageLine(t *testing.T) {
	config.SetListenDefaultsForTest()
	config.SetAIcallListenConversationMaxMessageCharsForTest(10)

	tests := []struct {
		name   string
		msg    *cvmessage.Message
		expect string
	}{
		{"incoming text", &cvmessage.Message{Direction: cvmessage.DirectionIncoming, Text: "  hello  "}, "[CUSTOMER] hello"},
		{"outgoing text", &cvmessage.Message{Direction: cvmessage.DirectionOutgoing, Text: "hi"}, "[AGENT] hi"},
		{"no direction", &cvmessage.Message{Direction: cvmessage.DirectionNond, Text: "x"}, "[SPEAKER] x"},
		{"subject prefixes the text", &cvmessage.Message{Direction: cvmessage.DirectionIncoming, Subject: "Re: bill", Text: "hi"}, "[CUSTOMER] Subject: Re: bill\nhi"},
		{"text over the cap is truncated", &cvmessage.Message{Direction: cvmessage.DirectionIncoming, Text: "0123456789ABCDEF"}, "[CUSTOMER] 0123456789 [truncated]"},
		{"media only", &cvmessage.Message{Direction: cvmessage.DirectionIncoming, Medias: []cvmedia.Media{{Type: cvmedia.TypeImage}}}, "[CUSTOMER] [media: image]"},
		{"text and two medias", &cvmessage.Message{Direction: cvmessage.DirectionIncoming, Text: "see", Medias: []cvmedia.Media{{Type: cvmedia.TypeImage}, {Type: cvmedia.TypeFile}}}, "[CUSTOMER] see [media: image] [media: file]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conversationMessageLine(tt.msg); got != tt.expect {
				t.Errorf("line mismatch.\nexpected: %q\ngot:      %q", tt.expect, got)
			}
		})
	}
}

// Test_EventCVMessageCreated covers design 2026-09-05 §5.3.2 exit by exit and
// the §5.4 debounce/flush decisions. Not parallel: metric deltas.
func Test_EventCVMessageCreated(t *testing.T) {
	otherAIcallID := uuid.FromStringOrNil("44440000-0000-4000-8000-000000000002")

	tests := []struct {
		name            string
		msg             *cvmessage.Message
		resolved        []uuid.UUID
		resolveErr      error
		getErr          error
		aicallCustomer  uuid.UUID
		lockAcquired    bool
		expectSegment   string
		expectBuffered  int
		expectLock      bool
		expectTurn      bool
		expectFlush     string
	}{
		{
			name:          "deleted message is dropped before the resolver",
			msg:           &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "x", TMDelete: ptrTimeNow()},
			expectSegment: "dropped_deleted",
		},
		{
			name:          "empty text without media is dropped before the resolver",
			msg:           &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "   "},
			expectSegment: "dropped_empty",
		},
		{
			name:          "unknown conversation is dropped",
			msg:           &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "x"},
			resolved:      []uuid.UUID{},
			expectSegment: "dropped_unknown",
		},
		{
			name:          "resolver error is dropped_unknown",
			msg:           &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "x"},
			resolveErr:    fmt.Errorf("redis down"),
			expectSegment: "dropped_unknown",
		},
		{
			name:           "aicall lookup failure is failed and nothing is buffered",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "x"},
			resolved:       []uuid.UUID{ltAIcallID},
			getErr:         fmt.Errorf("not found"),
			expectSegment:  "failed",
		},
		{
			name:           "tenant mismatch is dropped and nothing is buffered",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil("55550000-0000-4000-8000-000000000001")}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "x"},
			resolved:       []uuid.UUID{ltAIcallID},
			aicallCustomer: ltCustomerID,
			expectSegment:  "dropped_tenant_mismatch",
		},
		{
			name:           "outgoing line is buffered but never tries the lock",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionOutgoing, Text: "agent reply"},
			resolved:       []uuid.UUID{ltAIcallID},
			aicallCustomer: ltCustomerID,
			expectSegment:  "buffered",
			expectBuffered: 1,
		},
		{
			name:           "incoming line wins the lock and spawns a turn",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "customer"},
			resolved:       []uuid.UUID{ltAIcallID},
			aicallCustomer: ltCustomerID,
			lockAcquired:   true,
			expectSegment:  "buffered",
			expectBuffered: 1,
			expectLock:     true,
			expectTurn:     true,
		},
		{
			name:           "incoming line loses the lock and arms one deferred flush",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "customer"},
			resolved:       []uuid.UUID{ltAIcallID},
			aicallCustomer: ltCustomerID,
			lockAcquired:   false,
			expectSegment:  "buffered",
			expectBuffered: 1,
			expectLock:     true,
			expectFlush:    "armed",
		},
		{
			name:           "two listeners each get their own buffer and lock attempt",
			msg:            &cvmessage.Message{Identity: commonidentity.Identity{CustomerID: ltCustomerID}, ConversationID: lcConversationID, Direction: cvmessage.DirectionIncoming, Text: "customer"},
			resolved:       []uuid.UUID{ltAIcallID, otherAIcallID},
			aicallCustomer: ltCustomerID,
			lockAcquired:   true,
			expectSegment:  "buffered",
			expectBuffered: 2,
			expectLock:     true,
			expectTurn:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			config.SetAIcallListenEnabledForTest(true)
			config.SetAIcallListenConversationEnabledForTest(true)
			config.SetAIcallListenConversationFlushJitterMsForTest(0)

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newListenTurnHarness(mc)
			ctx := context.Background()

			armed := 0
			m.h.afterFunc = func(d time.Duration, fn func()) *time.Timer {
				armed++
				if d != 20*time.Second {
					t.Errorf("flush delay must be interval + 0 jitter. got: %v", d)
				}
				return time.NewTimer(time.Hour) // never fires in this test
			}
			turnsSpawned := make(chan uuid.UUID, 4)
			m.h.runListenTurnHook = func(_ context.Context, id uuid.UUID) { turnsSpawned <- id }

			if tt.msg.TMDelete == nil && strings.TrimSpace(tt.msg.Text) != "" || len(tt.msg.Medias) > 0 {
				m.cache.EXPECT().ListenConversationAIcallIDsGet(ctx, lcConversationID).Return(tt.resolved, tt.resolveErr).MaxTimes(1)
			}
			for _, id := range tt.resolved {
				id := id
				c := listeningConversationAIcall()
				c.ID = id
				c.CustomerID = tt.aicallCustomer
				if tt.getErr != nil {
					m.db.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
					continue
				}
				m.db.EXPECT().AIcallGet(ctx, id).Return(c, nil)
				if tt.expectBuffered > 0 {
					m.cache.EXPECT().ListenPendingPush(ctx, id, gomock.Any(), gomock.Any()).Return(nil)
					m.cache.EXPECT().ListenWindowPush(ctx, id, gomock.Any(), 40, gomock.Any()).Return(nil)
				}
				if tt.expectLock {
					m.cache.EXPECT().ListenTurnTryLock(ctx, id, 20*time.Second).Return(tt.lockAcquired, nil)
				} else {
					m.cache.EXPECT().ListenTurnTryLock(gomock.Any(), id, gomock.Any()).Times(0)
				}
			}

			segBefore := testutil.ToFloat64(promListenConversationSegmentTotal.WithLabelValues(tt.expectSegment))
			m.h.EventCVMessageCreated(ctx, tt.msg)
			segGot := testutil.ToFloat64(promListenConversationSegmentTotal.WithLabelValues(tt.expectSegment)) - segBefore
			if int(segGot) < 1 {
				t.Errorf("segment result %q must be metered. got: %v", tt.expectSegment, segGot)
			}

			if tt.expectTurn {
				for i := 0; i < len(tt.resolved); i++ {
					select {
					case <-turnsSpawned:
					case <-time.After(2 * time.Second):
						t.Fatalf("expected a turn to be spawned")
					}
				}
			} else if len(turnsSpawned) != 0 {
				t.Errorf("no turn must be spawned. got: %d", len(turnsSpawned))
			}
			switch tt.expectFlush {
			case "armed":
				if armed != 1 {
					t.Errorf("exactly one flush timer must be armed. got: %d", armed)
				}
			default:
				if armed != 0 {
					t.Errorf("no flush timer must be armed. got: %d", armed)
				}
			}
		})
	}
}

// Test_listenFlush pins the §5.4 invariants: one timer per AIcall per process
// while armed (skipped_scheduled), Delete-before-TryLock so a mid-flush
// arrival can re-arm, and lock-lost leaves the buffer alone.
func Test_listenFlush(t *testing.T) {
	config.SetListenDefaultsForTest()
	config.SetAIcallListenEnabledForTest(true)
	config.SetAIcallListenConversationEnabledForTest(true)
	config.SetAIcallListenConversationFlushJitterMsForTest(0)

	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	ctx := context.Background()

	var captured func()
	m.h.afterFunc = func(_ time.Duration, fn func()) *time.Timer {
		captured = fn
		return time.NewTimer(time.Hour)
	}
	turns := 0
	m.h.runListenTurnHook = func(_ context.Context, _ uuid.UUID) { turns++ }

	// First arm.
	scheduledBefore := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("skipped_scheduled"))
	m.h.scheduleListenFlush(ctx, ltAIcallID)
	if captured == nil {
		t.Fatalf("first call must arm a timer")
	}
	// Second arm while the first is pending is skipped_scheduled.
	first := captured
	m.h.scheduleListenFlush(ctx, ltAIcallID)
	if got := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("skipped_scheduled")) - scheduledBefore; got != 1 {
		t.Errorf("second arm must be skipped_scheduled. got: %v", got)
	}

	// Fire: lock lost -> skipped_locked, no turn, and the marker is already
	// cleared so a re-arm from inside the callback succeeds (Delete before
	// TryLock).
	m.cache.EXPECT().ListenTurnTryLock(gomock.Any(), ltAIcallID, 20*time.Second).Return(false, nil)
	lockedBefore := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("skipped_locked"))
	rearmed := false
	m.h.afterFunc = func(_ time.Duration, fn func()) *time.Timer { rearmed = true; return time.NewTimer(time.Hour) }
	first()
	if got := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("skipped_locked")) - lockedBefore; got != 1 {
		t.Errorf("lock lost must be skipped_locked. got: %v", got)
	}
	if turns != 0 {
		t.Errorf("no turn on a lost lock. got: %d", turns)
	}
	m.h.scheduleListenFlush(ctx, ltAIcallID)
	if !rearmed {
		t.Errorf("the marker must be cleared before TryLock so a new arm succeeds")
	}

	// Fire: lock won -> ran and a turn.
	m.cache.EXPECT().ListenTurnTryLock(gomock.Any(), ltAIcallID, 20*time.Second).Return(true, nil)
	ranBefore := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("ran"))
	m.h.listenFlushFire(ltAIcallID)
	if got := testutil.ToFloat64(promListenConversationFlushTotal.WithLabelValues("ran")) - ranBefore; got != 1 {
		t.Errorf("lock won must be ran. got: %v", got)
	}
	if turns != 1 {
		t.Errorf("exactly one turn on a won lock. got: %d", turns)
	}
}

func ptrTimeNow() *time.Time {
	now := time.Now()
	return &now
}
```

Add `"time"`, `cvmessage "monorepo/bin-conversation-manager/models/message"` and `cvmedia "monorepo/bin-conversation-manager/models/media"` to the file's imports (Task 5 deliberately did not import `cvmessage`; this task is its first user. `"time"` may already be present from Task 6's `ProcessListen` test; declare each import once).

- [ ] **Step 2: Run them to verify they fail**

Run: `cd bin-ai-manager && go test ./pkg/aicallhandler/ -run 'Test_conversationMessageLine|Test_EventCVMessageCreated|Test_listenFlush' -v 2>&1 | tail -20`
Expected: FAIL to compile (`undefined: conversationMessageLine`, `m.h.afterFunc undefined`, ...).

- [ ] **Step 3: Add the seams to `aicallHandler` and the interface method**

In `main.go`, in the `AIcallHandler` interface after `EventTMTranscriptCreated`:

```go
	EventCVMessageCreated(ctx context.Context, evt *cvmessage.Message)
```

with import `cvmessage "monorepo/bin-conversation-manager/models/message"` (the package already vendors it via `tool_insight.go`).

In the `aicallHandler` struct after `runListenStartHook`:

```go
	// afterFunc is the deferred-flush timer seam (design 2026-09-05 §5.4). nil
	// means time.AfterFunc; tests inject a capturing stub.
	afterFunc func(d time.Duration, fn func()) *time.Timer

	// runListenTurnHook, when set, replaces the detached RunListenTurn goroutine
	// spawned by the conversation intake and flush so tests can observe it.
	runListenTurnHook func(ctx context.Context, aicallID uuid.UUID)

	// flushScheduled marks AIcall ids with a deferred flush timer armed in THIS
	// process. Advisory only -- the correctness bounds come from the Redis lock
	// and pending list; this merely stops a burst from arming N timers.
	flushScheduled sync.Map
```

Add `"sync"` and `"time"` to `main.go`'s imports. Leave `NewAIcallHandler` unchanged (nil `afterFunc` falls back to `time.AfterFunc`).

- [ ] **Step 4: Add the intake half to `listen_conversation.go`**

Append:

```go
// speakerTagForDirection maps a conversation message's direction to the
// structural speaker tag the listen turn prompt expects.
func speakerTagForDirection(direction cvmessage.Direction) string {
	switch direction {
	case cvmessage.DirectionIncoming:
		return "[CUSTOMER]"
	case cvmessage.DirectionOutgoing:
		return "[AGENT]"
	default:
		return "[SPEAKER]"
	}
}

const listenConversationTruncatedSuffix = " [truncated]"

// conversationMessageLine renders one buffered line (design 2026-09-05 §5.3.3):
// tag, optional "Subject: ..." line, whitespace-trimmed text truncated to
// aicall_listen_conversation_max_message_chars, and one "[media: <type>]"
// token per attachment (raw media.Type, no allowlist, no URL or payload).
func conversationMessageLine(evt *cvmessage.Message) string {
	var sb strings.Builder
	sb.WriteString(speakerTagForDirection(evt.Direction))

	if subject := strings.TrimSpace(evt.Subject); subject != "" {
		sb.WriteString(" Subject: ")
		sb.WriteString(subject)
		sb.WriteString("\n")
	} else {
		sb.WriteString(" ")
	}

	text := strings.TrimSpace(evt.Text)
	if maxChars := config.Get().AIcallListenConversationMaxMessageChars; maxChars > 0 {
		if runes := []rune(text); len(runes) > maxChars {
			text = string(runes[:maxChars]) + listenConversationTruncatedSuffix
		}
	}
	sb.WriteString(text)

	for _, m := range evt.Medias {
		// One separator before each token. When text is empty the builder
		// already ends in the tag's trailing space (or the subject's newline),
		// so only add a space after non-empty content.
		if s := sb.String(); !strings.HasSuffix(s, " ") && !strings.HasSuffix(s, "\n") {
			sb.WriteString(" ")
		}
		sb.WriteString("[media: ")
		sb.WriteString(string(m.Type))
		sb.WriteString("]")
	}

	return strings.TrimRight(sb.String(), " \n")
}
```

Trace against the Step 1 table: media only -> `"[CUSTOMER] "` then `"[media: image]"` (suffix is a space, no extra) -> `"[CUSTOMER] [media: image]"`; text + two medias -> `"[CUSTOMER] see"` + `" [media: image]"` + `" [media: file]"`; subject + text -> `"[CUSTOMER] Subject: Re: bill\nhi"`.

Then the handler:

```go
// EventCVMessageCreated is the conversation listen intake (design 2026-09-05
// §5.3.2). It runs for EVERY conversation message platform-wide, so everything
// before the resolver lookup must stay trivially cheap, and the resolver
// lookup itself is the only Redis round trip for the >99% that are not ours.
func (h *aicallHandler) EventCVMessageCreated(ctx context.Context, evt *cvmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "EventCVMessageCreated",
		"conversation_id": evt.ConversationID,
		"message_id":      evt.ID,
	})

	if evt.TMDelete != nil {
		promListenConversationSegmentTotal.WithLabelValues("dropped_deleted").Inc()
		return
	}
	if strings.TrimSpace(evt.Text) == "" && len(evt.Medias) == 0 {
		promListenConversationSegmentTotal.WithLabelValues("dropped_empty").Inc()
		return
	}

	aicallIDs, err := h.cache.ListenConversationAIcallIDsGet(ctx, evt.ConversationID)
	if err != nil {
		log.Warnf("Could not resolve the listening aicalls. err: %v", err)
		promListenConversationSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}
	if len(aicallIDs) == 0 {
		promListenConversationSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}

	line := conversationMessageLine(evt)
	ttl := listenBufferTTL()
	windowSize := config.Get().AIcallListenWindowSize
	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second

	for _, aicallID := range aicallIDs {
		// h.Get -> h.db.AIcallGet (the dbhandler is cache-first with a DB
		// fallback); acceptable because this runs only for the resolved
		// fraction. The tenant assertion needs the row.
		c, errGet := h.Get(ctx, aicallID)
		if errGet != nil {
			log.Warnf("Could not get the listening aicall. aicall_id: %s, err: %v", aicallID, errGet)
			promListenConversationSegmentTotal.WithLabelValues("failed").Inc()
			continue
		}
		if c.CustomerID == uuid.Nil || evt.CustomerID == uuid.Nil || c.CustomerID != evt.CustomerID {
			log.Warnf("Cross-customer conversation message blocked. aicall_id: %s, aicall_customer_id: %s, message_customer_id: %s", aicallID, c.CustomerID, evt.CustomerID)
			promListenConversationSegmentTotal.WithLabelValues("dropped_tenant_mismatch").Inc()
			continue
		}

		if errPending := h.cache.ListenPendingPush(ctx, aicallID, line, ttl); errPending != nil {
			log.Warnf("Could not buffer the pending line. aicall_id: %s, err: %v", aicallID, errPending)
			continue
		}
		if errWindow := h.cache.ListenWindowPush(ctx, aicallID, line, windowSize, ttl); errWindow != nil {
			log.Warnf("Could not buffer the window line. aicall_id: %s, err: %v", aicallID, errWindow)
		}
		promListenConversationSegmentTotal.WithLabelValues("buffered").Inc()

		// Only a customer message starts a turn; agent/bot output is context.
		if evt.Direction != cvmessage.DirectionIncoming {
			continue
		}

		acquired, errLock := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
		if errLock != nil {
			log.Warnf("Could not take the listen turn lock. aicall_id: %s, err: %v", aicallID, errLock)
			continue
		}
		if !acquired {
			promListenTurnTotal.WithLabelValues(string(listenKindConversation), "skipped_locked").Inc()
			h.scheduleListenFlush(ctx, aicallID)
			continue
		}

		h.spawnListenTurn(aicallID)
	}
}

// spawnListenTurn runs RunListenTurn detached, or hands the id to the test hook.
func (h *aicallHandler) spawnListenTurn(aicallID uuid.UUID) {
	if h.runListenTurnHook != nil {
		h.runListenTurnHook(context.Background(), aicallID)
		return
	}
	go func(id uuid.UUID) {
		turnCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		h.RunListenTurn(turnCtx, id)
	}(aicallID)
}

// scheduleListenFlush arms at most one deferred flush per AIcall per process
// (design 2026-09-05 §5.4). The delay is the debounce interval plus a random
// jitter so the two replicas' timers do not race the lock at the same instant.
func (h *aicallHandler) scheduleListenFlush(ctx context.Context, aicallID uuid.UUID) {
	if _, loaded := h.flushScheduled.LoadOrStore(aicallID, struct{}{}); loaded {
		promListenConversationFlushTotal.WithLabelValues("skipped_scheduled").Inc()
		return
	}

	delay := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second
	if jitter := config.Get().AIcallListenConversationFlushJitterMs; jitter > 0 {
		delay += time.Duration(rand.IntN(jitter+1)) * time.Millisecond
	}

	after := h.afterFunc
	if after == nil {
		after = time.AfterFunc
	}
	after(delay, func() { h.listenFlushFire(aicallID) })
}

// listenFlushFire is the timer body. INVARIANT: the marker is deleted BEFORE
// TryLock, never after the turn, so a message arriving mid-flush can arm a
// fresh timer instead of being stranded behind a marker nobody will clear.
func (h *aicallHandler) listenFlushFire(aicallID uuid.UUID) {
	h.flushScheduled.Delete(aicallID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second
	acquired, err := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
	if err != nil {
		logrus.WithFields(logrus.Fields{"func": "listenFlushFire", "aicall_id": aicallID}).
			Warnf("Could not take the listen turn lock for the flush. err: %v", err)
		promListenConversationFlushTotal.WithLabelValues("skipped_locked").Inc()
		return
	}
	if !acquired {
		promListenConversationFlushTotal.WithLabelValues("skipped_locked").Inc()
		return
	}

	promListenConversationFlushTotal.WithLabelValues("ran").Inc()
	if h.runListenTurnHook != nil {
		h.runListenTurnHook(ctx, aicallID)
		return
	}
	h.RunListenTurn(ctx, aicallID)
}
```

Add to `listen_conversation.go`'s imports: `"math/rand/v2"` (as `rand`), `"strings"`, `"time"`, `cvmessage "monorepo/bin-conversation-manager/models/message"`.

Note on `conversationMessageLine`: the media loop's separator condition is deliberately simple; the trailing `TrimRight` keeps "[CUSTOMER] [media: image]" from ending in a space when text is empty. Verify against the table in Step 1 and adjust the separator logic (not the expectations) if a case differs.

- [ ] **Step 5: Regenerate the aicallhandler mock and run the tests**

Run: `cd bin-ai-manager && go generate ./pkg/aicallhandler/... && go test ./pkg/aicallhandler/ 2>&1 | tail -30`
Expected: PASS. `mock_main.go` now has `EventCVMessageCreated`.

- [ ] **Step 6: Bind the subscribe handler**

`bin-ai-manager/pkg/subscribehandler/main.go`:

Add to the publisher const block:

```go
	publisherConversationManager = string(commonoutline.ServiceNameConversationManager)
```

Append to `topicPatterns` (LAST; the golden test is position-sensitive):

```go
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated),
```

Add the import `cvmessage "monorepo/bin-conversation-manager/models/message"`.

In `processEvent`'s switch, after the `publisherTranscribeManager` case:

```go
	case m.Publisher == publisherConversationManager && m.Type == cvmessage.EventTypeMessageCreated:
		err = h.processEventCVMessageCreated(ctx, m)
```

Create `bin-ai-manager/pkg/subscribehandler/conversationmanager.go`:

```go
package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/sirupsen/logrus"
)

// processEventCVMessageCreated handles conversation-manager's
// conversation_message_created event.
//
// It fires for EVERY conversation message on the platform (all channels, both
// directions) and processEventRun spawns a goroutine per event, so it does the
// minimum: unmarshal, hand off, return. The ownership filter (one Redis
// SMEMBERS) lives in aicallHandler.EventCVMessageCreated with the listen state.
func (h *subscribeHandler) processEventCVMessageCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCVMessageCreated",
		"event": m,
	})

	var evt cvmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventCVMessageCreated(ctx, &evt)

	return nil
}
```

Update `binding_golden_test.go`: append `"conversation-manager.conversation.*.message_created",` to `expected`, change all three occurrences of `12` to `13` (the comment at line 30, the `!= 12` check at line 31, and the `expected: 12` inside the `Fatalf` format string at line 32), and the comment to `// design §5 + VOIP-1422 + NOJIRA Insight AI realtime listen + VOIP-1470: ai-manager binds exactly 13 patterns.`

Add a routing-key cross-check test in the same file (design §7 item 9):

```go
// Test_conversationBindingMatchesProducer pins that the bound pattern is what
// conversation-manager actually publishes for a message, so a renamed event
// type fails here rather than in production.
func Test_conversationBindingMatchesProducer(t *testing.T) {
	want := eventtopic.PatternForEventType(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated)
	if want != "conversation-manager.conversation.*.message_created" {
		t.Fatalf("pattern mismatch. got: %q", want)
	}
	msg := &cvmessage.Message{ConversationID: uuid.FromStringOrNil("66660000-0000-4000-8000-000000000001")}
	key := eventtopic.RoutingKey(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated, msg.EventSubscriptionID())
	if key != "conversation-manager.conversation.66660000-0000-4000-8000-000000000001.message_created" {
		t.Errorf("routing key mismatch. got: %q", key)
	}
}
```

with imports `"monorepo/bin-common-handler/models/eventtopic"`, `commonoutline "monorepo/bin-common-handler/models/outline"`, `cvmessage "monorepo/bin-conversation-manager/models/message"`, `"github.com/gofrs/uuid"`.

- [ ] **Step 7: Run the subscribehandler tests**

Run: `cd bin-ai-manager && go test ./pkg/subscribehandler/ -v 2>&1 | tail -20`
Expected: PASS (`Test_topicPatterns_golden` with 13, `Test_conversationBindingMatchesProducer`).

- [ ] **Step 8: Document the event**

In `bin-ai-manager/docs/architecture.md`, in the events/subscriptions section that lists `transcribe-manager.transcript.*.created` (near line 100), add a row/bullet:

```
- `conversation-manager.conversation.*.message_created` -> `aicallHandler.EventCVMessageCreated` (VOIP-1470 conversation listen intake). Fires for every conversation message platform-wide; the per-event floor is one unmarshal + one Redis `SMEMBERS` on `ai:listen:conversation:<conversation_id>`; >99% end there as `dropped_unknown`.
```

- [ ] **Step 9: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/ bin-ai-manager/pkg/subscribehandler/ bin-ai-manager/docs/architecture.md
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-ai-manager: Add the conversation_message_created intake with tenant assertion, line builder, incoming-only debounce and the deferred flush, and bind conversation-manager in the subscribe handler"
```

---

### Task 8: Comment fix, full verification, PR

**Files:**
- Modify: `bin-conversation-manager/models/conversation/metadata.go:11-15`

- [ ] **Step 1: Fix the stale comment**

Replace lines 11-15 of `metadata.go` with:

```go
	// ContactCaseID claims this Conversation for a Case. Its only writer today
	// is bin-api-manager's Case message-reply path
	// (pkg/servicehandler/case_message.go), via ConversationV1ConversationUpdateMetadata;
	// flow-manager's case_create action does NOT set it, so consumers must not
	// rely on it being present for every conversation-origin Case (see
	// docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.2).
	// Read-only from conversation-manager's own perspective: never read by
	// getExecuteMode or any flow/agent-routing dispatch decision.
```

- [ ] **Step 2: Run the full verification workflow for every touched service**

```bash
cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-contact-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-conversation-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all four green. Also run `cd bin-ai-manager && go test -race ./pkg/aicallhandler/ ./pkg/subscribehandler/` and expect PASS (the flush marker and hooks are exercised under the race detector).

- [ ] **Step 3: Commit**

```bash
git add bin-conversation-manager/models/conversation/metadata.go
git add -u
git commit -m "VOIP-1470-Insight-AI-conversation-listen

- bin-conversation-manager: Correct the Metadata.ContactCaseID comment to name its only writer"
```

- [ ] **Step 4: Conflict check and push**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main
git push -u origin VOIP-1470-Insight-AI-conversation-listen
```

If conflicts appear, merge `origin/main` into the branch, resolve, re-run Step 2, then push.

- [ ] **Step 5: Open the PR (do NOT merge)**

Title: `VOIP-1470-Insight-AI-conversation-listen`. Body:

```
Let the Case Insight Assistant follow a messaging conversation (SMS/MMS, LINE, WhatsApp, webchat) in real time and proactively push notify_agent notes, by extending the call listen pipeline with a conversation-manager message_created intake. Design: docs/plans/2026-09-05-insight-ai-conversation-listen-design.md (rev 6, approved). Ships dark: aicall_listen_conversation_enabled defaults to false and is independent of aicall_listen_enabled. No DB migration, no API change, no frontend change.

- docs: Add the approved conversation listen design and the implementation plan
- bin-contact-manager: Add kase.ReferenceTypeConversationMessage with a value-pin test
- bin-flow-manager: Use the kase reference-type constants in the case_create action
- bin-ai-manager: Add aicall_listen_conversation_enabled, _max_message_chars and _flush_jitter_ms config flags
- bin-ai-manager: Add MetaKeyListenConversationID and the ai:listen:conversation resolver cache primitives plus ListenPendingLen
- bin-ai-manager: Add the kind label to the listen start, turn and notify metrics and the conversation segment and flush vecs
- bin-ai-manager: Make RunListenTurn conversation-aware (scoped sub-flag, empty short-circuit, turn-time Case status check), add the conversation listen prompt, clear the conversation resolver on stop, widen the terminate gate and clean up idle-expired listeners
- bin-ai-manager: Branch the listen trigger on conversation Cases with the inline, idempotent startListenConversation
- bin-ai-manager: Add the conversation_message_created intake with tenant assertion, incoming-only debounce and deferred flush, and bind conversation-manager in the subscribe handler
- bin-conversation-manager: Correct the Metadata.ContactCaseID comment
```

Then close design-only PR #1268 with a comment pointing at the new PR (its commit is the first commit of this branch, so nothing is lost).

---

## Self-review against the design

- §5.1 trigger branch and step 5b: Task 6. §5.1.1 idempotency + two writes + rollback: Task 6. §5.2 metadata key + resolver quartet + `ListenPendingLen`: Task 3. §5.3.1 binding, publisher const, golden 13: Task 7 step 6. §5.3.2 intake exits incl. `failed` and tenant: Task 7. §5.3.3 line format, direction rule, media, truncation: Task 7. §5.4 debounce, deferred flush, Delete-before-TryLock, empty short-circuit: Tasks 5 and 7. §5.5 turn order, scoped sub-flag, Case check, prompt, header: Task 5. §5.7 terminate gate, idle expiry at `start.go:492` only, `clearListenState` SREM: Task 5. §5.8 kase constant + flow literal + comment fix: Tasks 1 and 8. §5.12 flags: Task 2. §5.13 `kind` label incl. `unknown`, new vecs, notify CounterVec: Task 4. §7 tests: each task's step 1; §7 item 9 routing-key cross-check: Task 7 step 6. Docs (operations, architecture, domain): Tasks 2, 7, 3.
- Type consistency: `listenKind`/`listenKindOf`/`label()` (Task 4) used in Tasks 5-7; `startListenConversation(ctx, c, conversationID) string` (Task 6) matches its tests; `scheduleListenFlush(ctx, aicallID)` and `listenFlushFire(aicallID)` (Task 7) match `Test_listenFlush`; `afterFunc func(time.Duration, func()) *time.Timer` matches `time.AfterFunc`'s signature; `runListenTurnHook func(ctx, uuid.UUID)` is used by both intake and flush.
- Known deviations from the design text, recorded (plan review round 1):
  1. `startListenConversation` re-reads the row with `db.AIcallGet` before merging metadata (mirrors `UpdateListenState`) instead of using `c.Metadata` directly; behaviour is the same, clobber-safety is better.
  2. Design §5.13 says `RunListenTurn`'s predicate-failed `skipped_invalid` (`listen.go:272`) emits `kind="unknown"` "because it means `listenKind == none`". That premise is incomplete: the predicate is a three-way OR, so a terminated *call* listener reaches it with `kind == call`. The plan emits `listenKindOf(c).label()` there, which yields `unknown` only when no pointer exists and the true kind otherwise. Strictly more accurate; `Test_RunListenTurn`'s empty-metadata row pins the `unknown` case.
  3. Design §5.7 replaces `process.go:63`'s second clause; the plan ORs the old `ListenCallID != uuid.Nil` clause with `listenKindOf(tmp) != listenKindNone` inside a pure `listenTerminateNeedsStop` helper. Wider and safe (a call row always has both), and the helper makes the gate unit-testable.
  4. Design §9 lists only `actionhandle.go:1341`; Task 1 also converts the sibling `"call"` literal at `:1332` to `kmkase.ReferenceTypeCall` for consistency. No behaviour change.
  5. The `process.go` terminate gate is tested through `listenTerminateNeedsStop`'s table rather than by extending `Test_ProcessTerminate`, whose rows drive the full terminate flow and carry no listen state.

## Review-response matrix (plan review round 1 -> rev 2)

| # | Severity | Finding (abridged) | Resolution in rev 2 |
|---|---|---|---|
| 1 | BLOCKING | `metricDelta("call", ...)` breaks the empty-metadata `Test_RunListenTurn` row (kind is `unknown`) | `metricDelta(t, kind, label, fn)`; `expectKind` column, `"unknown"` on that row; call sites 515/626/1028 updated (Task 4 step 6) |
| 2 | BLOCKING | media separator always emitted a space; media-only row failed | separator only when the builder does not already end in a space/newline; trace added (Task 7 step 4) |
| 3 | HIGH | predicate-failed label deviates from §5.13 unrecorded | deviation 2 above |
| 4 | HIGH | `Test_validateListenConfig` does not exist | exact rows in `Test_Validate_ListenSizing` + `Test_Validate_ZeroConversationFlushJitterIsAllowed` (Task 2 step 4) |
| 5 | HIGH | `RunListenTurn` anchor 244-308 would delete `runListenTurnWithLines`'s doc comment | 244-299 (Task 5 step 5) |
| 6 | HIGH | §7 items uncovered: call-kind row, `runListenStartHook` never invoked, idle non-listening row | call-kind row in `Test_RunListenTurn_Conversation`; `Test_ProcessListen_ConversationBranchNeverSpawnsRunListenStart`; original idle row with a strict `cache` mock is the non-listening proof (Tasks 5, 6) |
| 7 | MEDIUM | grep count "about 61" | 33 with the per-file breakdown (Task 4 step 5) |
| 8 | MEDIUM | search-and-improvise steps | `Test_startReferenceTypeContactCase` row cloned from lines 4444-4526 with exact expectations; `listenTerminateNeedsStop` table (Task 5 step 8) |
| 9 | MEDIUM | cache-key test row was a placeholder | literal expected string with the `transcribeID` fixture; miniredis behaviour tests added (Task 3 steps 1, 5) |
| 10 | MEDIUM | "cache-first, DB on miss" misattributed to `h.Get` | comment corrected to `h.Get -> h.db.AIcallGet` (Task 7 step 4) |
| 11 | MEDIUM | `process.go` OR deviation unrecorded | deviation 3 above |
| 12 | LOW | six anchors off by one | `cachehandler/main.go:50`, `listen.go:185`, `listenStartLockKey :64`, `listenOwnsTranscribeFromMetadata ends :59` (re-corrected to :58 in rev 3), trigger test row `140-150` (re-corrected in rev 4), `metrics_listen_test.go:22-27` |
| 13 | LOW | `cap` shadows the builtin | `maxChars` (Task 7 step 4) |
| 14 | LOW | Task 1 exceeds §9 (`:1332`) | deviation 4 above |

## Review-response matrix (plan review round 2 -> rev 3)

| # | Severity | Finding (abridged) | Resolution in rev 3 |
|---|---|---|---|
| 1 | MEDIUM | Task 5's test-file import block declared `cvmessage` and `testutil` before any use (compile error) | removed from Task 5; Task 6 adds `testutil`, Task 7 adds `cvmessage`, each at first use |
| 2 | MEDIUM | keeping a `buildListenTranscriptBlock(window, newLines)` wrapper leaves it unused and fails golangci-lint | wrapper deleted; single `buildListenTranscriptBlock(header, window, newLines)`; call site and rationale updated (Task 5 step 4) |
| 3 | LOW | grep count 33 overshot | 29 with metrics_listen.go = 2 (Task 4 step 5) |
| 4 | LOW | test-file grep also matches `helpers_test.go` (`promAIcallInterruptAttemptedTotal`) | grep scoped to `promListen*`; helpers_test.go named as untouched (Task 4 step 6) |
| 5 | LOW | residual anchors | `cachehandler/listen.go:179`, `cachehandler/main.go:51`, `listen.go:58`, `start_test.go:4443-4526` |
| 6 | LOW | UUID re-prefix instruction missed `expectRes` | instruction now covers every id incl. `expectRes` (lines 4517-4523) (Task 5 step 8) |

## Review-response matrix (plan review round 3 -> rev 4, APPROVE with cosmetic LOWs)

| # | Severity | Finding (abridged) | Resolution in rev 4 |
|---|---|---|---|
| 1 | LOW | Task 6 step 4 replaced 253-256 but the `// Step 5` comment sits at 252, leaving two comments | range is 252-256 and the old block shown includes the comment |
| 2 | LOW | "change both `12` to `13`" undercounts (three occurrences: comment, check, Fatalf string) | wording lists all three with line numbers (Task 7 step 6) |
| 3 | LOW | `140-149` vs `140-150` inconsistency for the trigger test row | `140-150` everywhere (Task 6 step 1, matrix row 12) |
