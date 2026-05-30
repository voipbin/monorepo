package aicallhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_ProcessStart(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ed69462-a705-11ed-a47b-cfb979f9f07d"),
					CustomerID: uuid.FromStringOrNil("6f12ea52-a705-11ed-86d3-8b796a5da603"),
				},
				ActiveflowID:  uuid.FromStringOrNil("5b2ad484-093a-11f0-928b-3bd49b19fd87"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6f69db50-a705-11ed-bc35-177b3c1673d4"),
				STTLanguage:   "en-US",
			},

			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6f40a2c6-a705-11ed-8981-d78afab8acba"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, tt.aicall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.aicall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.aicall.CustomerID, aicall.EventTypeStatusProgressing, tt.aicall)

			res, err := h.ProcessStart(ctx, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.aicall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.aicall, res)
			}
		})
	}
}

func Test_ProcessTerminate(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAicall         *aicall.AIcall
		responsePipecatcall    *pmpipecatcall.Pipecatcall
		pipecatcallGetErr      error // non-nil to simulate stale/missing pipecatcall
	}{
		{
			// Already-terminated: idempotency guard returns early without any further RPCs.
			name: "already terminated - no-op",

			id: uuid.FromStringOrNil("f0f0f0f0-d9d8-11f0-bf0e-13aea4f95ca9"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0f0f0f0-d9d8-11f0-bf0e-13aea4f95ca9"),
				},
				Status: aicall.StatusTerminated,
			},
		},
		{
			name: "normal - reference type is task",

			id: uuid.FromStringOrNil("dd188916-d791-11f0-b284-4359b8729dde"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd188916-d791-11f0-b284-4359b8729dde"),
				},
				ConfbridgeID:  uuid.FromStringOrNil("a213c5f8-d794-11f0-9e01-3738b1dbf1d6"),
				PipecatcallID: uuid.FromStringOrNil("a2460c84-d794-11f0-a9e3-8ffda8bbf25b"),
				ReferenceType: aicall.ReferenceTypeTask,
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2460c84-d794-11f0-a9e3-8ffda8bbf25b"),
				},
				HostID: "host-12345",
			},
		},
		{
			name: "normal - reference type is call",

			id: uuid.FromStringOrNil("d9ee9868-d9d8-11f0-bf0e-13aea4f95ca9"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d9ee9868-d9d8-11f0-bf0e-13aea4f95ca9"),
				},
				ConfbridgeID:  uuid.FromStringOrNil("da1c1086-d9d8-11f0-b6e2-73784ba2c3f2"),
				PipecatcallID: uuid.FromStringOrNil("da4a0d92-d9d8-11f0-9dfe-4b563d93e22c"),
				ReferenceType: aicall.ReferenceTypeCall,
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("da4a0d92-d9d8-11f0-9dfe-4b563d93e22c"),
				},
				HostID: "host-12345",
			},
		},
		{
			// Stale aicall: pipecatcall_id is set but the pipecatcall no longer exists.
			// ProcessTerminate should skip the terminate RPC and continue to StatusTerminated.
			name: "stale aicall - pipecatcall not found",

			id: uuid.FromStringOrNil("e1a2b3c4-d9d8-11f0-bf0e-13aea4f95ca9"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e1a2b3c4-d9d8-11f0-bf0e-13aea4f95ca9"),
				},
				PipecatcallID: uuid.FromStringOrNil("f1a2b3c4-d9d8-11f0-9dfe-4b563d93e22c"),
				ReferenceType: aicall.ReferenceTypeTask,
			},
			pipecatcallGetErr: &cerrors.VoipbinError{
				Status:  cerrors.StatusNotFound,
				Reason:  "PIPECATCALL_NOT_FOUND",
				Message: "The pipecat call was not found.",
			},
		},
		{
			// Stale aicall with a live confbridge: pipecatcall not found, but confbridge
			// still exists and must still be terminated.
			name: "stale aicall - pipecatcall not found, confbridge present",

			id: uuid.FromStringOrNil("a1b2c3d4-d9d8-11f0-bf0e-13aea4f95ca9"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-d9d8-11f0-bf0e-13aea4f95ca9"),
				},
				ConfbridgeID:  uuid.FromStringOrNil("b1b2c3d4-d9d8-11f0-b6e2-73784ba2c3f2"),
				PipecatcallID: uuid.FromStringOrNil("c1b2c3d4-d9d8-11f0-9dfe-4b563d93e22c"),
				ReferenceType: aicall.ReferenceTypeTask,
			},
			pipecatcallGetErr: &cerrors.VoipbinError{
				Status:  cerrors.StatusNotFound,
				Reason:  "PIPECATCALL_NOT_FOUND",
				Message: "The pipecat call was not found.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAicall, nil)

			if tt.responseAicall.Status != aicall.StatusTerminated {
				mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, tt.responseAicall.ActiveflowID, tt.responseAicall.ID, 0).Return(nil)
				if tt.responseAicall.ReferenceType != aicall.ReferenceTypeCall {
					mockReq.EXPECT().FlowV1ActiveflowContinue(ctx, tt.responseAicall.ActiveflowID, tt.responseAicall.ID).Return(nil)
				}

				if tt.responseAicall.PipecatcallID != uuid.Nil {
					if tt.pipecatcallGetErr != nil {
						mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, tt.responseAicall.PipecatcallID).Return(nil, tt.pipecatcallGetErr)
					} else {
						mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, tt.responseAicall.PipecatcallID).Return(tt.responsePipecatcall, nil)
						mockReq.EXPECT().PipecatV1PipecatcallTerminate(ctx, tt.responsePipecatcall.HostID, tt.responsePipecatcall.ID).Return(tt.responsePipecatcall, nil)
					}
				}

				if tt.responseAicall.ConfbridgeID != uuid.Nil {
					mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.responseAicall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
				}

				now := time.Now()
				mockUtil.EXPECT().TimeNow().Return(&now)
				mockDB.EXPECT().AIcallUpdate(ctx, tt.responseAicall.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().AIcallGet(ctx, tt.responseAicall.ID).Return(tt.responseAicall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAicall.CustomerID, aicall.EventTypeStatusTerminated, tt.responseAicall)
			}

			res, err := h.ProcessTerminate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAicall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAicall, res)
			}
		})
	}
}

// Test_ProcessTerminate_autoAuditTrigger verifies that the auto-audit fire-and-forget hook
// fires exactly when MetaKeyAutoAuditEnabled is true, and that a publish error from
// AIV1AIAuditCreateWithDelay does NOT fail termination.
func Test_ProcessTerminate_autoAuditTrigger(t *testing.T) {
	// Shared IDs used across test cases.
	aicallID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	// baseAIcall is the aicall returned by the first Get call (h.Get at the top of ProcessTerminate).
	// ReferenceTypeCall skips the FlowV1ActiveflowContinue branch.
	// PipecatcallID == uuid.Nil and ConfbridgeID == uuid.Nil skip the pipecat/confbridge branches.
	makeBaseAIcall := func(meta map[string]any) *aicall.AIcall {
		return &aicall.AIcall{
			Identity: commonidentity.Identity{
				ID:         aicallID,
				CustomerID: customerID,
			},
			ReferenceType: aicall.ReferenceTypeCall,
			PipecatcallID: uuid.Nil,
			ConfbridgeID:  uuid.Nil,
			Metadata:      meta,
		}
	}

	tests := []struct {
		name string

		metadata map[string]any

		// auditPublishErr, if non-nil, is the error AIV1AIAuditCreateWithDelay returns.
		// Only meaningful when expectAuditCall == true.
		auditPublishErr error

		// expectAuditCall controls whether we register an EXPECT for AIV1AIAuditCreateWithDelay.
		expectAuditCall bool
	}{
		{
			name:            "flag true - audit triggered",
			metadata:        map[string]any{aicall.MetaKeyAutoAuditEnabled: true},
			expectAuditCall: true,
			auditPublishErr: nil,
		},
		{
			name:            "flag false - audit not triggered",
			metadata:        map[string]any{aicall.MetaKeyAutoAuditEnabled: false},
			expectAuditCall: false,
		},
		{
			name:            "flag absent - audit not triggered",
			metadata:        map[string]any{},
			expectAuditCall: false,
		},
		{
			name:            "flag true but publish error - termination still succeeds",
			metadata:        map[string]any{aicall.MetaKeyAutoAuditEnabled: true},
			expectAuditCall: true,
			auditPublishErr: fmt.Errorf("publish failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			// The aicall returned by the first Get (top of ProcessTerminate).
			baseAIcall := makeBaseAIcall(tt.metadata)

			// The aicall returned by the second Get (inside UpdateStatus after the DB write).
			// It carries the same Metadata so the hook can read the flag.
			terminatedAIcall := makeBaseAIcall(tt.metadata)

			// First Get: called at the top of ProcessTerminate.
			mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(baseAIcall, nil)

			// FlowV1ActiveflowServiceStop: always called for non-terminated aicalls.
			mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, baseAIcall.ActiveflowID, baseAIcall.ID, 0).Return(nil)

			// ReferenceTypeCall: FlowV1ActiveflowContinue is NOT called (branch is skipped).

			// UpdateStatus internals: TimeNow, AIcallUpdate, second AIcallGet, PublishWebhookEvent.
			now := time.Now()
			mockUtil.EXPECT().TimeNow().Return(&now)
			mockDB.EXPECT().AIcallUpdate(ctx, aicallID, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(terminatedAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, aicall.EventTypeStatusTerminated, terminatedAIcall)

			// Auto-audit hook: only expected when the flag is true.
			if tt.expectAuditCall {
				mockReq.EXPECT().
					AIV1AIAuditCreateWithDelay(ctx, customerID, aicallID, "", autoAuditTriggerDelay).
					Return(tt.auditPublishErr)
			}

			res, err := h.ProcessTerminate(ctx, aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, terminatedAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", terminatedAIcall, res)
			}
		})
	}
}
