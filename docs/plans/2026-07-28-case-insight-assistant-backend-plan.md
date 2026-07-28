# Case Insight Assistant — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `reference_type=contact_case` AIcalls behave correctly end-to-end through the existing `service_agents` API surface — no new endpoints — so square-admin's Case Insight Assistant panel (separate frontend plan) has a working backend to call.

**Architecture:** Six small, targeted fixes across two existing services in this monorepo: `bin-ai-manager` (`pkg/aicallhandler/start.go`, `send.go`) and `bin-api-manager` (`pkg/servicehandler/serviceagent_aicall.go`), plus one `bin-openapi-manager` spec edit and one RST doc update. No new files except tests.

**Tech Stack:** Go, gomock (`go.uber.org/mock`), table-driven tests, squirrel (`sq`) for SQL building, MySQL.

**Source design:** `square-admin/docs/plans/2026-07-28-case-insight-assistant-design.md` (APPROVED, Round 14). Read §3-§5 there for the "why" behind each fix below if anything here is unclear.

---

## File Structure

| File | Responsibility |
|---|---|
| `bin-ai-manager/pkg/aicallhandler/start.go` | `startReferenceTypeContactCase` — Tasks 1, 2 |
| `bin-ai-manager/pkg/aicallhandler/start_test.go` | Tests for the above |
| `bin-ai-manager/pkg/aicallhandler/send.go` | `Send` dispatcher — Task 3 |
| `bin-ai-manager/pkg/aicallhandler/send_test.go` | Tests for the above |
| `bin-ai-manager/internal/config/main.go` | New `aicall_send_cooldown_seconds` flag — Task 3 |
| `bin-openapi-manager/openapi/paths/service_agents/aicalls.yaml` | `assistance_id` conditionally-optional spec — Task 4 |
| `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go` | `ServiceAgentAIcallCreate` — Tasks 5, 6, 7 |
| `bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go` | Tests for the above |
| `bin-api-manager/docsdev/source/` | RST doc update — Task 8 |

Rollout order matches the design's §7: Tasks 1-3 (`bin-ai-manager`) and 4-7 (`bin-api-manager`/openapi) can land in either order relative to each other (different services, independently deployable), but Task 6 (AI resolution) has no dependency on Tasks 1-3 either. Task 8 (docs) lands with whichever PR ships last. This plan orders them 1→8 for a single linear PR; split into two PRs (one per service) if preferred — each task is self-contained enough to reorder.

---

## Task 1: Trigger the first AI turn on genuine creation only

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:411-419` (the `err == nil` return branch of `startReferenceTypeContactCase`)
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go` (add a case to `Test_startReferenceTypeContactCase`, `~line 3579`)

- [ ] **Step 1: Write the failing test**

Add this test case to the `tests` table in `Test_startReferenceTypeContactCase` (`pkg/aicallhandler/start_test.go`), right after the `"create succeeds on first attempt"` case (`~line 3663`):

```go
		{
			name: "create succeeds on first attempt — starts pipecatcall and advances to progressing",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("40000000-0001-11f0-dddd-000000000001"),
					CustomerID: uuid.FromStringOrNil("40000000-0002-11f0-dddd-000000000001"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("40000000-0001-11f0-dddd-000000000001"),
			activeflowID:   uuid.FromStringOrNil("40000000-0003-11f0-dddd-000000000001"),
			referenceID:    uuid.FromStringOrNil("40000000-0004-11f0-dddd-000000000001"),

			mockSetup: func(ctx context.Context, m *mocks) {
				pipecatcallID := uuid.FromStringOrNil("40000000-0005-11f0-dddd-000000000001")
				aicallID := uuid.FromStringOrNil("40000000-0006-11f0-dddd-000000000001")
				customerID := uuid.FromStringOrNil("40000000-0002-11f0-dddd-000000000001")
				created := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         aicallID,
						CustomerID: customerID,
					},
					ActiveflowID:  uuid.FromStringOrNil("40000000-0003-11f0-dddd-000000000001"),
					ReferenceType: aicall.ReferenceTypeContactCase,
					ReferenceID:   uuid.FromStringOrNil("40000000-0004-11f0-dddd-000000000001"),
					Status:        aicall.StatusInitiating,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("40000000-0007-11f0-dddd-000000000001")},
					HostID:   "host1",
				}
				progressing := &aicall.AIcall{
					Identity:      created.Identity,
					ActiveflowID:  created.ActiveflowID,
					ReferenceType: created.ReferenceType,
					ReferenceID:   created.ReferenceID,
					Status:        aicall.StatusProgressing,
				}

				m.util.EXPECT().UUIDCreate().Return(pipecatcallID)
				m.util.EXPECT().UUIDCreate().Return(aicallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, aicallID).Return(created, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, created.CustomerID, aicall.EventTypeStatusInitializing, created)
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)
				m.message.EXPECT().Create(ctx, uuid.Nil, created.CustomerID, created.ID, created.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// Task 1: the new first-turn trigger.
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx, created.PipecatcallID, created.CustomerID, created.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall, created.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
				m.db.EXPECT().AIcallUpdate(ctx, aicallID, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, aicallID).Return(progressing, nil)
			},

			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("40000000-0006-11f0-dddd-000000000001"),
					CustomerID: uuid.FromStringOrNil("40000000-0002-11f0-dddd-000000000001"),
				},
				ActiveflowID:  uuid.FromStringOrNil("40000000-0003-11f0-dddd-000000000001"),
				ReferenceType: aicall.ReferenceTypeContactCase,
				ReferenceID:   uuid.FromStringOrNil("40000000-0004-11f0-dddd-000000000001"),
				Status:        aicall.StatusProgressing,
			},
		},
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/aicallhandler/... -run Test_startReferenceTypeContactCase -v`
Expected: FAIL — the existing code returns right after `AIcallGet`, so the test's `PipecatV1PipecatcallStart`/`AIcallUpdate` expectations are never satisfied ("missing call(s)").

- [ ] **Step 3: Write minimal implementation**

In `pkg/aicallhandler/start.go`, replace the `err == nil` branch of `startReferenceTypeContactCase` (currently lines ~413-418):

```go
		res, err := h.startAIcallByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeContactCase, referenceID, false, teamParameter, currentMemberID)
		if err == nil {
			log.WithField("aicall", res).Debugf("Created aicall for contact_case. aicall_id: %s", res.ID)

			// Trigger the first AI turn exactly once, only here — this branch
			// runs only when the INSERT genuinely succeeded (no duplicate-key
			// retry involved), so there is no risk of double-triggering.
			pc, errStart := h.startPipecatcall(ctx, res)
			if errStart != nil {
				return nil, errors.Wrapf(errStart, "could not start pipecatcall for contact_case aicall. aicall_id: %s", res.ID)
			}
			if errTerm := h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultAITaskTimeout); errTerm != nil {
				return nil, errors.Wrapf(errTerm, "could not schedule pipecatcall termination. aicall_id: %s", res.ID)
			}
			if updated, errStatus := h.UpdateStatus(ctx, res.ID, aicall.StatusProgressing); errStatus != nil {
				log.Warnf("Could not update status to Progressing — continuing anyway. aicall_id: %s, err: %v", res.ID, errStatus)
			} else {
				res = updated
			}

			return res, nil
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/aicallhandler/... -run Test_startReferenceTypeContactCase -v`
Expected: PASS (all subtests, including the pre-existing ones — `startPipecatcall` internally calls `h.getPipecatcallMessages`, which for a brand-new AIcall with no prior messages calls `m.message.List` once, matching the mock above).

- [ ] **Step 5: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Trigger first AI turn on contact_case AIcall creation

- bin-ai-manager: startReferenceTypeContactCase now starts a pipecatcall and advances to Progressing on genuine creation, matching every other reference type's lifecycle"
```

---

## Task 2: Idle-expired contact_case AIcalls terminate instead of being reused forever

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:429-448` (the duplicate-key branch of `startReferenceTypeContactCase`)
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go`

- [ ] **Step 1: Write the failing test**

Add this case to `Test_startReferenceTypeContactCase`, and add `config.SetAIcallConversationIdleTimeoutHoursForTest(24)` to the top of the test function (next to the existing `SetAIcallContactCaseRecreateRateLimitMinutesForTest(5)` call) so the idle threshold is deterministic:

```go
		{
			name: "duplicate key — existing idle-expired (not yet terminated) — terminates and retries without rate limit",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50000000-0001-11f0-eeee-000000000001"),
					CustomerID: uuid.FromStringOrNil("50000000-0002-11f0-eeee-000000000001"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("50000000-0001-11f0-eeee-000000000001"),
			activeflowID:   uuid.FromStringOrNil("50000000-0003-11f0-eeee-000000000001"),
			referenceID:    uuid.FromStringOrNil("50000000-0004-11f0-eeee-000000000001"),

			mockSetup: func(ctx context.Context, m *mocks) {
				staleTM := time.Now().Add(-25 * time.Hour) // outside the 24h idle timeout
				existingIdleID := uuid.FromStringOrNil("50000000-0009-11f0-eeee-000000000001")
				existingIdle := &aicall.AIcall{
					Identity: commonidentity.Identity{ID: existingIdleID, CustomerID: uuid.FromStringOrNil("50000000-0002-11f0-eeee-000000000001")},
					Status:   aicall.StatusProgressing, // NOT Terminated/Terminating -- this is the bug case
					TMUpdate: &staleTM,
				}

				pipecatcallID1 := uuid.FromStringOrNil("50000000-0005-11f0-eeee-000000000001")
				aicallID1 := uuid.FromStringOrNil("50000000-0006-11f0-eeee-000000000001")
				pipecatcallID2 := uuid.FromStringOrNil("50000000-0007-11f0-eeee-000000000001")
				aicallID2 := uuid.FromStringOrNil("50000000-0008-11f0-eeee-000000000001")

				// attempt 0: duplicate key, existing is idle-expired (Progressing, stale TMUpdate)
				m.util.EXPECT().UUIDCreate().Return(pipecatcallID1)
				m.util.EXPECT().UUIDCreate().Return(aicallID1)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_aicall_active_reference_key'"))
				m.db.EXPECT().AIcallGetByReferenceID(ctx, uuid.FromStringOrNil("50000000-0004-11f0-eeee-000000000001")).Return(existingIdle, nil)

				// Task 2: terminate the idle row -- NO recreate-rate-limit check on this path.
				m.db.EXPECT().AIcallUpdate(ctx, existingIdleID, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusTerminated}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, existingIdleID).Return(&aicall.AIcall{
					Identity: existingIdle.Identity, Status: aicall.StatusTerminated,
				}, nil)

				// attempt 1: create succeeds (and starts pipecatcall per Task 1)
				created := &aicall.AIcall{
					Identity:      commonidentity.Identity{ID: aicallID2, CustomerID: uuid.FromStringOrNil("50000000-0002-11f0-eeee-000000000001")},
					ActiveflowID:  uuid.FromStringOrNil("50000000-0003-11f0-eeee-000000000001"),
					ReferenceType: aicall.ReferenceTypeContactCase,
					ReferenceID:   uuid.FromStringOrNil("50000000-0004-11f0-eeee-000000000001"),
					Status:        aicall.StatusInitiating,
				}
				m.util.EXPECT().UUIDCreate().Return(pipecatcallID2)
				m.util.EXPECT().UUIDCreate().Return(aicallID2)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, aicallID2).Return(created, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, created.CustomerID, aicall.EventTypeStatusInitializing, created)
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)
				m.message.EXPECT().Create(ctx, uuid.Nil, created.CustomerID, created.ID, created.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				responsePC := &pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("50000000-000a-11f0-eeee-000000000001")}, HostID: "host2"}
				m.req.EXPECT().PipecatV1PipecatcallStart(ctx, created.PipecatcallID, created.CustomerID, created.ActiveflowID, pmpipecatcall.ReferenceTypeAICall, created.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
				m.db.EXPECT().AIcallUpdate(ctx, aicallID2, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, aicallID2).Return(&aicall.AIcall{Identity: created.Identity, Status: aicall.StatusProgressing}, nil)
			},

			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50000000-0008-11f0-eeee-000000000001"),
					CustomerID: uuid.FromStringOrNil("50000000-0002-11f0-eeee-000000000001"),
				},
				Status: aicall.StatusProgressing,
			},
			expectRateLimitedInc: false, // the whole point of this test: NOT rate-limited
		},
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/aicallhandler/... -run Test_startReferenceTypeContactCase -v`
Expected: FAIL — current code takes the `Status == Terminated/Terminating` branch's `else` (reuse) path for a `Progressing` row regardless of `TMUpdate`, so it returns `existingIdle` directly instead of terminating and retrying.

- [ ] **Step 3: Write minimal implementation**

In `pkg/aicallhandler/start.go`, replace the tail of `startReferenceTypeContactCase` (currently lines ~429-448, from `if existing.Status == aicall.StatusTerminated...` through the final `return existing, nil`):

```go
		if existing.Status == aicall.StatusTerminated || existing.Status == aicall.StatusTerminating {
			// The row that won the race has since terminated (or was already
			// terminating). Its active_reference_key computes to NULL, so it no
			// longer occupies the unique slot — but before retrying the create,
			// check the recreate rate limit (VOIP-1234 design doc §2-1).
			if blocked, blockedErr := h.checkContactCaseRecreateRateLimit(referenceID, existing); blocked {
				return nil, blockedErr
			}

			log.Infof("Existing AIcall for contact_case is terminated/terminating — retrying create. aicall_id: %s, attempt: %d", existing.ID, attempt+1)
			lastErr = err
			continue
		}

		if h.isAIcallIdleExpired(existing) {
			// Still "live" by status, but idle past the configured timeout.
			// Terminate it explicitly and retry — deliberately WITHOUT the
			// recreate rate limit, which exists to bound someone else's very
			// recent explicit termination, not this AIcall's own long idle
			// gap (design doc §3, Round 4 correction).
			log.Infof("Existing AIcall for contact_case is idle-expired — terminating and retrying create. aicall_id: %s, attempt: %d", existing.ID, attempt+1)
			promAIcallIdleExpiredTotal.Inc()
			if _, errEnd := h.UpdateStatus(ctx, existing.ID, aicall.StatusTerminated); errEnd != nil {
				log.Warnf("Could not terminate idle AIcall: %v", errEnd)
			}
			lastErr = err
			continue
		}

		log.WithField("aicall", existing).Debugf("Reusing existing active aicall for contact_case. aicall_id: %s", existing.ID)
		return existing, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/aicallhandler/... -run Test_startReferenceTypeContactCase -v`
Expected: PASS.

- [ ] **Step 5: Run the full package test suite (make sure nothing else broke)**

Run: `go test ./pkg/aicallhandler/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Terminate idle-expired contact_case AIcalls instead of reusing forever

- bin-ai-manager: startReferenceTypeContactCase now checks isAIcallIdleExpired on the reuse path and explicitly terminates + retries, skipping the recreate rate limit (which does not apply to organic idle expiry)"
```

---

## Task 3: Per-AIcall send cooldown

**Files:**
- Modify: `bin-ai-manager/internal/config/main.go` (new config flag)
- Modify: `bin-ai-manager/pkg/aicallhandler/send.go:14-27` (`Send` dispatcher)
- Test: `bin-ai-manager/pkg/aicallhandler/send_test.go` (new `Test_Send` function — none exists yet)

- [ ] **Step 1: Add the config flag**

In `bin-ai-manager/internal/config/main.go`, add a new field to the config struct (next to `AIcallContactCaseRecreateRateLimitMinutes`, `~line 35`):

```go
	AIcallSendCooldownSeconds int // Minimum seconds between two Send() calls on the same AIcall, to bound LLM spend from rapid repeated sends.
```

Add the flag registration (next to the other `aicall_*` flags, `~line 70`):

```go
	f.Int("aicall_send_cooldown_seconds", 3, "Minimum seconds between two Send() calls on the same AIcall")
```

Add it to the env-binding map (`~line 90`, next to the other `aicall_*` entries) and the `viper.GetInt(...)` assignment (`~line 135`), following the exact same pattern as `aicall_contact_case_recreate_rate_limit_minutes` two lines above each.

Add a test-override helper next to `SetAIcallContactCaseRecreateRateLimitMinutesForTest` (`~line 164`):

```go
// SetAIcallSendCooldownSecondsForTest overrides the send cooldown in tests.
func SetAIcallSendCooldownSecondsForTest(seconds int) {
	globalConfig.AIcallSendCooldownSeconds = seconds
}
```

- [ ] **Step 2: Write the failing test**

Create `bin-ai-manager/pkg/aicallhandler/send_test.go`'s new `Test_Send` function (this file has `Test_SendReferenceTypeOthers`/`Test_SendReferenceTypeCall` but no `Test_Send` yet — add this as a new top-level function, anywhere in the file):

```go
func Test_Send(t *testing.T) {
	config.SetAIcallSendCooldownSecondsForTest(3)

	type mocks struct {
		util    *utilhandler.MockUtilHandler
		req     *requesthandler.MockRequestHandler
		db      *dbhandler.MockDBHandler
		message *messagehandler.MockMessageHandler
	}

	tests := []struct {
		name string

		aicallID uuid.UUID

		mockSetup func(ctx context.Context, m *mocks)

		expectErr          bool
		expectErrSubstring string
	}{
		{
			name:     "within cooldown -- rejected before dispatch",
			aicallID: uuid.FromStringOrNil("60000000-0001-11f0-ffff-000000000001"),

			mockSetup: func(ctx context.Context, m *mocks) {
				recentTM := time.Now().Add(-1 * time.Second) // inside the 3s cooldown
				m.db.EXPECT().AIcallGet(ctx, uuid.FromStringOrNil("60000000-0001-11f0-ffff-000000000001")).Return(&aicall.AIcall{
					Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("60000000-0001-11f0-ffff-000000000001")},
					ReferenceType: aicall.ReferenceTypeContactCase,
					TMUpdate:      &recentTM,
				}, nil)
			},
			expectErr:          true,
			expectErrSubstring: "cooldown",
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &mocks{
				util:    utilhandler.NewMockUtilHandler(mc),
				req:     requesthandler.NewMockRequestHandler(mc),
				db:      dbhandler.NewMockDBHandler(mc),
				message: messagehandler.NewMockMessageHandler(mc),
			}
			h := &aicallHandler{
				utilHandler:    m.util,
				reqHandler:     m.req,
				db:             m.db,
				messageHandler: m.message,
			}
			tt.mockSetup(ctx, m)

			_, err := h.Send(ctx, tt.aicallID, message.RoleUser, "hello", false, false)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Send() expected error, got nil")
				} else if tt.expectErrSubstring != "" && !strings.Contains(err.Error(), tt.expectErrSubstring) {
					t.Errorf("Send() error = %v, want substring %q", err, tt.expectErrSubstring)
				}
				return
			}
			if err != nil {
				t.Errorf("Send() unexpected error: %v", err)
			}
		})
	}
}
```

Add `"monorepo/bin-ai-manager/internal/config"` and `"time"` to this file's imports if not already present.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/aicallhandler/... -run Test_Send$ -v`
Expected: FAIL — `Send()` today has no cooldown check, so it proceeds into `SendReferenceTypeOthers` and the test's mock setup (which only sets up `AIcallGet`) is insufficient — the test fails with an unexpected-call panic from gomock, not the expected cooldown error.

- [ ] **Step 4: Write minimal implementation**

In `pkg/aicallhandler/send.go`, modify `Send`:

```go
func (h *aicallHandler) Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	c, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	if c.TMUpdate != nil {
		cooldown := time.Duration(config.Get().AIcallSendCooldownSeconds) * time.Second
		if elapsed := time.Since(*c.TMUpdate); elapsed < cooldown {
			return nil, errors.Errorf("send cooldown: aicall %s was updated %s ago, minimum interval is %s", id, elapsed.Round(time.Millisecond), cooldown)
		}
	}

	switch c.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.SendReferenceTypeCall(ctx, c, role, messageText, runImmediately, audioResponse)

	default:
		return h.SendReferenceTypeOthers(ctx, c, role, messageText)
	}
}
```

Add `"monorepo/bin-ai-manager/internal/config"` and `"time"` to `send.go`'s imports.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/aicallhandler/... -run Test_Send$ -v`
Expected: PASS.

- [ ] **Step 6: Run the full package test suite**

Run: `go test ./pkg/aicallhandler/... -v`
Expected: PASS. (`Test_SendReferenceTypeOthers`/`Test_SendReferenceTypeCall` call `SendReferenceTypeOthers`/`SendReferenceTypeCall` directly, bypassing `Send`'s new cooldown check, so they are unaffected.)

- [ ] **Step 7: Commit**

```bash
git add bin-ai-manager/internal/config/main.go bin-ai-manager/pkg/aicallhandler/send.go bin-ai-manager/pkg/aicallhandler/send_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Add per-AIcall send cooldown

- bin-ai-manager: Send() now rejects a new send within aicall_send_cooldown_seconds (default 3s) of the AIcall's last update, bounding LLM spend from rapid repeated sends"
```

---

## Task 4: Make `assistance_id` conditionally optional in the openapi spec

**Files:**
- Modify: `bin-openapi-manager/openapi/paths/service_agents/aicalls.yaml`

- [ ] **Step 1: Edit the spec**

In the `post:` operation's `requestBody.content.application/json.schema`, change the `required` list from:

```yaml
          required:
            - assistance_type
            - assistance_id
            - reference_type
            - reference_id
```

to:

```yaml
          required:
            - assistance_type
            - reference_type
            - reference_id
```

Add a note to the `assistance_id` property's `description` (find it just above the `required` block):

```yaml
            assistance_id:
              type: string
              format: uuid
              description: >-
                The unique identifier of the assistance entity (AI or Team).
                Returned from the `POST /ais`, `GET /ais`, `POST /teams`, or
                `GET /teams` response. Optional when `assistance_type=ai` and
                `reference_type=contact_case`: if omitted, the customer's own
                `type=insight` AI is resolved automatically. Required for
                every other combination.
              example: "550e8400-e29b-41d4-a716-446655440000"
```

- [ ] **Step 2: Regenerate the server stubs**

Run (from `bin-api-manager/`, per that service's codegen convention): `go generate ./...`
Expected: `gens/openapi_server/gen.go`'s `PostServiceAgentsAicallsJSONBody.AssistanceId` (or equivalent) changes from a required to an optional field. Diff it to confirm.

- [ ] **Step 3: Commit**

```bash
git add bin-openapi-manager/openapi/paths/service_agents/aicalls.yaml
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Make assistance_id conditionally optional for contact_case AI creates

- bin-openapi-manager: assistance_id is no longer unconditionally required on POST /service_agents/aicalls -- omit it for assistance_type=ai + reference_type=contact_case to get server-side Insight AI resolution"
```

(Regenerated `bin-api-manager/gens/openapi_server/gen.go` gets committed together with Task 5-7's `bin-api-manager` changes below, since that's where the generated code is consumed.)

---

## Task 5: Case-ownership check on `ServiceAgentAIcallCreate`

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go:129-153`
- Test: `bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go`

- [ ] **Step 1: Write the failing test**

Find (or create, if `ServiceAgentAIcallCreate` has no table-driven test yet — check the file first) the test function for `ServiceAgentAIcallCreate` in `serviceagent_aicall_test.go`, and add this case:

```go
		{
			name: "contact_case reference_id belongs to a different customer -- rejected",

			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("70000000-0001-11f0-1111-000000000001"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("70000000-0002-11f0-1111-000000000001"),

			responseCase: &cmcase.Case{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("70000000-0002-11f0-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("70000000-9999-11f0-1111-000000000001"), // NOT the caller's customer
				},
			},

			expectErr: serviceerrors.ErrPermissionDenied,
		},
```

Adjust the table's field names to match whatever the existing test already uses for mocking `h.reqHandler.ContactV1CaseGet` (a `responseCase`/`responseCaseErr`-shaped field feeding a `mockSetup`, following this file's existing convention — read the file first and match its style exactly rather than introducing a new pattern).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: FAIL — no ownership check exists yet, so the call proceeds to `aiGet`/activeflow creation instead of returning `ErrPermissionDenied` early.

- [ ] **Step 3: Write minimal implementation**

In `pkg/servicehandler/serviceagent_aicall.go`, insert this block into `ServiceAgentAIcallCreate` right after the `hasPermission` check (before the `// resolve the assistance entity's customer id...` comment, `~line 153`):

```go
	// Case-ownership check (contact_case only). Rejects before touching
	// anything else if the referenced Case doesn't belong to the caller's
	// own customer -- fail-fast; tool_insight.go already fails closed on
	// this independently, so this is a UX/hygiene guard, not the last line
	// of defense (design doc §4.5).
	if referenceType == amaicall.ReferenceTypeContactCase {
		kase, errCase := h.reqHandler.ContactV1CaseGet(ctx, a.CustomerID, referenceID)
		if errCase != nil || kase.CustomerID != a.CustomerID {
			log.Info("The referenced case does not belong to the agent's customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}
```

Add the `cmcase "monorepo/bin-contact-manager/models/case"` import if `ContactV1CaseGet`'s return type requires it and it's not already imported (check `pkg/servicehandler/case.go`'s imports for the exact alias already used elsewhere in this package, and reuse it — don't introduce a second alias for the same package).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: PASS (this new case; also re-run the full function's existing cases to confirm none needed a `ContactV1CaseGet` mock added — they shouldn't, since they use non-`contact_case` reference types unless the existing tests already use `contact_case`, in which case add the same-customer `responseCase` mock to each so they don't break).

- [ ] **Step 5: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/serviceagent_aicall.go bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Reject contact_case AIcall creation for a foreign customer's case

- bin-api-manager: ServiceAgentAIcallCreate now validates reference_id's owning customer for reference_type=contact_case before proceeding"
```

---

## Task 6: Server-side Insight AI resolution when `assistance_id` is omitted

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go` (same function, plus a new helper)
- Test: `bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go`

- [ ] **Step 1: Write the failing test**

Add these two cases to the same test table as Task 5:

```go
		{
			name: "assistance_id omitted, contact_case, exactly one insight AI -- resolved",

			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.Nil,
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("80000000-0002-11f0-2222-000000000001"),

			responseCase: &cmcase.Case{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("80000000-0002-11f0-2222-000000000001"),
					CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001"),
				},
			},
			responseAIList: []amai.AI{
				{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("80000000-0004-11f0-2222-000000000001"), CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001")}, Type: amai.TypeInsight},
			},
			responseAI: &amai.AI{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("80000000-0004-11f0-2222-000000000001"), CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001")}},
			// ... plus whatever this file's existing successful-create cases mock for
			// activeflow create + AIV1AIcallStart, following their exact pattern.
		},
		{
			name: "assistance_id omitted, contact_case, zero insight AIs -- not found",

			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.Nil,
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("80000000-0005-11f0-2222-000000000001"),

			responseCase: &cmcase.Case{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("80000000-0005-11f0-2222-000000000001"),
					CustomerID: uuid.FromStringOrNil("80000000-0006-11f0-2222-000000000001"),
				},
			},
			responseAIList: []amai.AI{},

			expectErr: serviceerrors.ErrNotFound,
		},
```

Wire `responseAIList` through `mockSetup` to `m.req.EXPECT().AIV1AIList(ctx, "", uint64(100), gomock.Any()).Return(tt.responseAIList, nil)`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: FAIL — `assistanceID=uuid.Nil` today falls into `h.aiGet(ctx, uuid.Nil)`, which errors out with a generic "not found"-shaped error rather than the resolution/empty-state behavior the test expects (and the zero-AI case doesn't return `serviceerrors.ErrNotFound` specifically).

- [ ] **Step 3: Write minimal implementation**

In `pkg/servicehandler/serviceagent_aicall.go`, insert this block right after Task 5's ownership check, before the `// resolve the assistance entity's customer id...` switch:

```go
	// Server-side Insight AI resolution: assistance_id may be omitted for
	// assistance_type=ai + reference_type=contact_case (design doc §4.1,
	// Round 9). This is the only combination the schema allows it to be
	// omitted for (see the openapi spec change in this same PR).
	if assistanceType == amaicall.AssistanceTypeAI && assistanceID == uuid.Nil {
		if referenceType != amaicall.ReferenceTypeContactCase {
			return nil, errors.Wrapf(serviceerrors.ErrInvalidArgument, "assistance_id is required for reference_type: %s", referenceType)
		}

		resolvedID, errResolve := h.resolveInsightAIID(ctx, a.CustomerID)
		if errResolve != nil {
			return nil, errResolve
		}
		assistanceID = resolvedID
		log.WithField("assistance_id", assistanceID).Debugf("Resolved the customer's insight AI.")
	}
```

Add this new function at the end of the same file:

```go
// resolveInsightAIID looks up the caller's own customer's single
// type=insight AI. Returns serviceerrors.ErrNotFound if none exists (the
// square-admin panel maps this to its empty state), or the
// most-recently-created one if 2+ exist (AIV1AIList orders tm_create desc).
func (h *serviceHandler) resolveInsightAIID(ctx context.Context, customerID uuid.UUID) (uuid.UUID, error) {
	filters, err := h.convertAIFilters(map[string]string{
		"deleted":     "false",
		"customer_id": customerID.String(),
		"type":        string(amai.TypeInsight),
	})
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "could not convert ai filters")
	}

	ais, err := h.reqHandler.AIV1AIList(ctx, "", 100, filters)
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "could not list ais")
	}
	if len(ais) == 0 {
		return uuid.Nil, serviceerrors.ErrNotFound
	}

	return ais[0].ID, nil
}
```

Add the `amai "monorepo/bin-ai-manager/models/ai"` import (this file currently only imports `amaicall "monorepo/bin-ai-manager/models/aicall"` — a separate package, both needed).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/serviceagent_aicall.go bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Resolve the customer's Insight AI server-side when assistance_id is omitted

- bin-api-manager: ServiceAgentAIcallCreate resolves assistance_id automatically for assistance_type=ai + reference_type=contact_case, so CustomerAgent-level callers never need the Admin/Manager-only AI list endpoint"
```

---

## Task 7: Delete the orphaned activeflow when `AIV1AIcallStart` returns a reused AIcall

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go:179-215` (end of `ServiceAgentAIcallCreate`)
- Test: `bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go`

- [ ] **Step 1: Write the failing test**

Add this case to the same test table:

```go
		{
			name: "AIV1AIcallStart returns a reused aicall -- orphaned activeflow is deleted",

			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("90000000-0001-11f0-3333-000000000001"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("90000000-0002-11f0-3333-000000000001"),

			responseCase: &cmcase.Case{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90000000-0002-11f0-3333-000000000001"),
					CustomerID: uuid.FromStringOrNil("90000000-0003-11f0-3333-000000000001"),
				},
			},
			responseAI: &amai.AI{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("90000000-0001-11f0-3333-000000000001"), CustomerID: uuid.FromStringOrNil("90000000-0003-11f0-3333-000000000001")}},

			responseActiveflow: &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("90000000-0004-11f0-3333-000000000001")}},

			// The reused aicall's ActiveflowID does NOT match the one just created above.
			responseAIcallStart: &amaicall.WebhookMessage{
				ID:           uuid.FromStringOrNil("90000000-0005-11f0-3333-000000000001"),
				ActiveflowID: uuid.FromStringOrNil("90000000-9999-11f0-3333-000000000001"), // a DIFFERENT, pre-existing activeflow
			},

			expectActiveflowDeleted: uuid.FromStringOrNil("90000000-0004-11f0-3333-000000000001"), // the freshly-created one, not the reused one
		},
```

Wire `expectActiveflowDeleted` through `mockSetup` to `m.req.EXPECT().FlowV1ActiveflowDelete(ctx, tt.expectActiveflowDeleted).Return(nil, nil)` when set, and assert via the existing success-path mocks (`FlowV1ActiveflowCreate` returning `responseActiveflow`, `AIV1AIcallStart` returning `responseAIcallStart`) that this delete call actually fires — it must NOT fire in the pre-existing "normal create" test cases (add `m.req.EXPECT().FlowV1ActiveflowDelete(...)` with `.Times(0)` implicitly by simply not setting it up — gomock fails on unexpected calls by default).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: FAIL — the current code returns right after `tmp.ConvertWebhookMessage()` with no comparison, so `FlowV1ActiveflowDelete` is never called and the test's expectation is unmet.

- [ ] **Step 3: Write minimal implementation**

In `pkg/servicehandler/serviceagent_aicall.go`, modify the tail of `ServiceAgentAIcallCreate` (currently ends at `res := tmp.ConvertWebhookMessage(); return res, nil`):

```go
	tmp, err := h.reqHandler.AIV1AIcallStart(
		ctx,
		assistanceType,
		assistanceID,
		af.ID,
		referenceType,
		referenceID,
	)
	if err != nil {
		// best-effort cleanup of the orphaned activeflow
		if _, errDelete := h.reqHandler.FlowV1ActiveflowDelete(ctx, af.ID); errDelete != nil {
			log.Errorf("Could not delete orphaned activeflow. activeflow_id: %s, err: %v", af.ID, errDelete)
		}
		return nil, errors.Wrapf(err, "could not create aicall")
	}

	// If AIV1AIcallStart returned a REUSED aicall (its ActiveflowID differs
	// from the activeflow this call just created above), the activeflow we
	// created is now orphaned -- clean it up the same way the error path
	// above does (design doc §3 Round 6/7 finding: ServiceAgentAIcallCreate
	// otherwise leaks one activeflow per call against an already-live
	// contact_case session).
	if tmp.ActiveflowID != af.ID {
		if _, errDelete := h.reqHandler.FlowV1ActiveflowDelete(ctx, af.ID); errDelete != nil {
			log.Errorf("Could not delete orphaned activeflow after aicall reuse. activeflow_id: %s, err: %v", af.ID, errDelete)
		}
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/servicehandler/... -run Test_ServiceAgentAIcallCreate -v`
Expected: PASS.

- [ ] **Step 5: Run the full package test suite**

Run: `go test ./pkg/servicehandler/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/serviceagent_aicall.go bin-api-manager/pkg/servicehandler/serviceagent_aicall_test.go
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Delete the orphaned activeflow when an AIcall create call reuses an existing session

- bin-api-manager: ServiceAgentAIcallCreate now cleans up its freshly-created activeflow whenever AIV1AIcallStart returns a reused (not genuinely new) AIcall, closing an activeflow leak on every reuse"
```

---

## Task 8: RST doc sync

**Files:**
- Modify: `bin-api-manager/docsdev/source/aicall_struct_aicall.rst` (or the closest matching resource page — find it first, see Step 1)

- [ ] **Step 1: Find the right file**

Run: `grep -rln "service_agents/aicalls\|service_agents/aimessages" bin-api-manager/docsdev/source/`

If a `service_agents`-specific overview page exists, edit that. If (as of this writing) only `aicall_struct_aicall.rst` documents the AIcall resource generally, add a subsection there instead — follow whatever heading/section convention that file already uses for documenting error responses (look for how other endpoints in the same file document their error conditions, and match that format exactly).

- [ ] **Step 2: Document the new behavior**

Add a short section (RST format, matching the surrounding file's heading level) covering:
1. `POST /service_agents/aicalls`: `assistance_id` is now optional when `assistance_type=ai` and `reference_type=contact_case` — the customer's own `type=insight` AI is resolved automatically. Returns `404 RESOURCE_NOT_FOUND` if the customer has no Insight AI configured. Returns `403 PERMISSION_DENIED` if `reference_id` (for `contact_case`) doesn't belong to the caller's own customer.
2. `POST /service_agents/aimessages`: may now be rejected with a cooldown error if called again on the same `aicall_id` within a short window of the previous send (default 3 seconds).

- [ ] **Step 3: Rebuild the docs**

Run:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```
Expected: clean build, no Sphinx errors/warnings about the new section.

- [ ] **Step 4: Commit**

```bash
cd bin-api-manager/docsdev
git add source/ 
git add -f build/
git commit -m "NOJIRA-Aicalls-add-messages-endpoint: Document contact_case AI resolution and send cooldown in RST docs

- bin-api-manager: Add RST documentation for the new assistance_id-omitted resolution behavior and the send cooldown on service_agents AIcall endpoints"
```

---

## Final Verification

- [ ] **Step 1: Full verification workflow**

Run from each touched service directory (`bin-ai-manager`, `bin-api-manager`, `bin-openapi-manager` if it has its own module):

```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all green, in every touched service.

- [ ] **Step 2: Confirm the frontend's dependency is satisfiable**

Manually verify (or write a small integration smoke test if the repo has integration-test infra — check `docs/workflows/` for the convention before adding new tooling) that a fresh `POST /service_agents/aicalls` with `assistance_type=ai`, `assistance_id` omitted, `reference_type=contact_case` against a customer with exactly one `type=insight` AI: creates a new AIcall, advances it to `progressing`, and (once the AI's tools produce a response) shows up via `GET /service_agents/aimessages?aicall_id=`. This is the exact sequence the square-admin frontend plan's Task 2 depends on.
