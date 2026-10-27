package aicallhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_EventCMConfbridgeJoined(t *testing.T) {
	callID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000001")
	aicallID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000002")
	customerID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000003")

	tests := []struct {
		name string

		getRes *aicall.AIcall
		getErr error

		expectStart bool
		updateErr   error
	}{
		{
			name:   "lookup failure does nothing",
			getErr: fmt.Errorf("select failed"),
		},
		{
			name: "terminated aicall is ignored",
			getRes: &aicall.AIcall{
				Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
				Status:   aicall.StatusTerminated,
			},
		},
		{
			name: "terminating aicall is ignored",
			getRes: &aicall.AIcall{
				Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
				Status:   aicall.StatusTerminating,
			},
		},
		{
			name: "initiating aicall is started",
			getRes: &aicall.AIcall{
				Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
				Status:   aicall.StatusInitiating,
			},
			expectStart: true,
		},
		{
			name: "start failure is swallowed",
			getRes: &aicall.AIcall{
				Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
				Status:   aicall.StatusInitiating,
			},
			expectStart: true,
			updateErr:   fmt.Errorf("update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &aicallHandler{db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, callID).Return(tt.getRes, tt.getErr)
			if tt.expectStart {
				mockDB.EXPECT().AIcallUpdate(ctx, aicallID, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing}).Return(tt.updateErr)
				if tt.updateErr == nil {
					progressing := &aicall.AIcall{
						Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
						Status:   aicall.StatusProgressing,
					}
					mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(progressing, nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, aicall.EventTypeStatusProgressing, progressing)
				}
			}

			h.EventCMConfbridgeJoined(ctx, &cmconfbridge.EventConfbridgeJoined{JoinedCallID: callID})
		})
	}
}

func Test_EventCMConfbridgeLeaved(t *testing.T) {
	callID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000011")
	aicallID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000012")
	customerID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000013")
	confbridgeID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000014")
	otherConfbridgeID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000015")

	newAIcall := func(cbID uuid.UUID) *aicall.AIcall {
		return &aicall.AIcall{
			Identity:      commonidentity.Identity{ID: aicallID, CustomerID: customerID},
			ReferenceType: aicall.ReferenceTypeCall,
			ReferenceID:   callID,
			ConfbridgeID:  cbID,
			Status:        aicall.StatusProgressing,
		}
	}

	tests := []struct {
		name string

		getRes *aicall.AIcall
		getErr error

		expectTerminate bool
		terminateGetErr error
	}{
		{
			name:   "lookup failure does nothing",
			getErr: fmt.Errorf("select failed"),
		},
		{
			name:   "other confbridge is ignored",
			getRes: newAIcall(otherConfbridgeID),
		},
		{
			name:            "matching confbridge terminates the aicall",
			getRes:          newAIcall(confbridgeID),
			expectTerminate: true,
		},
		{
			name:            "terminate failure is swallowed",
			getRes:          newAIcall(confbridgeID),
			expectTerminate: true,
			terminateGetErr: fmt.Errorf("select failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify, utilHandler: mockUtil}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, callID).Return(tt.getRes, tt.getErr)
			if tt.expectTerminate {
				if tt.terminateGetErr != nil {
					mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(nil, tt.terminateGetErr)
				} else {
					mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(tt.getRes, nil).Times(2)
					mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, uuid.Nil, aicallID, 0).Return(nil)
					mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, confbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
					now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
					mockUtil.EXPECT().TimeNow().Return(&now)
					mockDB.EXPECT().AIcallUpdate(ctx, aicallID, map[aicall.Field]any{
						aicall.FieldStatus: aicall.StatusTerminated,
						aicall.FieldTMEnd:  &now,
					}).Return(nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, aicall.EventTypeStatusTerminated, tt.getRes)
				}
			}

			h.EventCMConfbridgeLeaved(ctx, &cmconfbridge.EventConfbridgeLeaved{
				Confbridge:   cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: confbridgeID}},
				LeavedCallID: callID,
			})
		})
	}
}

func Test_EventPMPipecatcallInitialized_branches(t *testing.T) {
	pcID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000021")
	aicallID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000022")
	callID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000023")

	tests := []struct {
		name string

		refType pmpipecatcall.ReferenceType

		expectGet  bool
		getRes     *aicall.AIcall
		getErr     error
		expectStop bool
		stopErr    error
	}{
		{
			name:    "other reference type does nothing",
			refType: pmpipecatcall.ReferenceType("other"),
		},
		{
			name:      "aicall lookup failure does nothing",
			refType:   pmpipecatcall.ReferenceTypeAICall,
			expectGet: true,
			getErr:    fmt.Errorf("select failed"),
		},
		{
			name:      "non-call aicall does not stop media",
			refType:   pmpipecatcall.ReferenceTypeAICall,
			expectGet: true,
			getRes: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID},
				ReferenceType: aicall.ReferenceTypeConversation,
			},
		},
		{
			name:      "call aicall stops media",
			refType:   pmpipecatcall.ReferenceTypeAICall,
			expectGet: true,
			getRes: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   callID,
			},
			expectStop: true,
		},
		{
			name:      "media stop failure is swallowed",
			refType:   pmpipecatcall.ReferenceTypeAICall,
			expectGet: true,
			getRes: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   callID,
			},
			expectStop: true,
			stopErr:    fmt.Errorf("media down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{db: mockDB, reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectGet {
				mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(tt.getRes, tt.getErr)
			}
			if tt.expectStop {
				mockReq.EXPECT().CallV1CallMediaStop(ctx, callID).Return(tt.stopErr)
			}

			h.EventPMPipecatcallInitialized(ctx, &pmpipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID},
				ReferenceType: tt.refType,
				ReferenceID:   aicallID,
			})
		})
	}
}

func Test_ProcessStart_updateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000031")

	mockDB.EXPECT().AIcallUpdate(ctx, id, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing}).Return(fmt.Errorf("update failed"))

	res, err := h.ProcessStart(ctx, &aicall.AIcall{Identity: commonidentity.Identity{ID: id}})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error/nil, got: %v/%v", err, res)
	}
}

func Test_ProcessTerminate_dependencyFailures(t *testing.T) {
	id := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000041")
	customerID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000042")
	activeflowID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000043")
	pcID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000044")
	cbID := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000045")
	notFound := &cerrors.VoipbinError{Status: cerrors.StatusNotFound, Reason: "NOT_FOUND", Message: "gone"}

	tests := []struct {
		name string

		referenceType aicall.ReferenceType
		pipecatcallID uuid.UUID
		confbridgeID  uuid.UUID

		serviceStopErr    error
		continueErr       error
		pcGetRes          *pmpipecatcall.Pipecatcall
		pcGetErr          error
		pcTerminateErr    error
		confbridgeErr     error
		finalUpdateErr    error
		expectContinue    bool
		expectPCTerminate bool
		expectConfbridge  bool
		expectFinalize    bool

		expectErr bool
	}{
		{
			name:           "service stop failure does not abort",
			referenceType:  aicall.ReferenceTypeCall,
			serviceStopErr: fmt.Errorf("flow down"),
			expectFinalize: true,
		},
		{
			name:           "continue failure does not abort",
			referenceType:  aicall.ReferenceTypeTask,
			continueErr:    fmt.Errorf("flow down"),
			expectContinue: true,
			expectFinalize: true,
		},
		{
			name:          "pipecatcall lookup failure aborts",
			referenceType: aicall.ReferenceTypeCall,
			pipecatcallID: pcID,
			pcGetErr:      fmt.Errorf("pipecat down"),
			expectErr:     true,
		},
		{
			name:              "pipecatcall gone during terminate is benign",
			referenceType:     aicall.ReferenceTypeCall,
			pipecatcallID:     pcID,
			pcGetRes:          &pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: pcID}, HostID: "host-a"},
			pcTerminateErr:    notFound,
			expectPCTerminate: true,
			expectFinalize:    true,
		},
		{
			name:              "pipecatcall terminate failure does not abort",
			referenceType:     aicall.ReferenceTypeCall,
			pipecatcallID:     pcID,
			pcGetRes:          &pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: pcID}, HostID: "host-a"},
			pcTerminateErr:    fmt.Errorf("pipecat down"),
			expectPCTerminate: true,
			expectFinalize:    true,
		},
		{
			name:             "confbridge terminate failure does not abort",
			referenceType:    aicall.ReferenceTypeCall,
			confbridgeID:     cbID,
			confbridgeErr:    fmt.Errorf("call down"),
			expectConfbridge: true,
			expectFinalize:   true,
		},
		{
			name:           "final status update failure aborts",
			referenceType:  aicall.ReferenceTypeCall,
			finalUpdateErr: fmt.Errorf("update failed"),
			expectFinalize: true,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify, utilHandler: mockUtil}
			ctx := context.Background()

			current := &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: id, CustomerID: customerID},
				ActiveflowID:  activeflowID,
				ReferenceType: tt.referenceType,
				PipecatcallID: tt.pipecatcallID,
				ConfbridgeID:  tt.confbridgeID,
				Status:        aicall.StatusProgressing,
			}

			mockDB.EXPECT().AIcallGet(ctx, id).Return(current, nil)
			mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, activeflowID, id, 0).Return(tt.serviceStopErr)
			if tt.expectContinue {
				mockReq.EXPECT().FlowV1ActiveflowContinue(ctx, activeflowID, id).Return(tt.continueErr)
			}
			if tt.pipecatcallID != uuid.Nil {
				mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, pcID).Return(tt.pcGetRes, tt.pcGetErr)
			}
			if tt.expectPCTerminate {
				mockReq.EXPECT().PipecatV1PipecatcallTerminate(ctx, "host-a", pcID).Return(tt.pcGetRes, tt.pcTerminateErr)
			}
			if tt.expectConfbridge {
				mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, cbID).Return(nil, tt.confbridgeErr)
			}
			if tt.expectFinalize {
				now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
				mockUtil.EXPECT().TimeNow().Return(&now)
				mockDB.EXPECT().AIcallUpdate(ctx, id, map[aicall.Field]any{
					aicall.FieldStatus: aicall.StatusTerminated,
					aicall.FieldTMEnd:  &now,
				}).Return(tt.finalUpdateErr)
				if tt.finalUpdateErr == nil {
					terminated := &aicall.AIcall{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: aicall.StatusTerminated}
					mockDB.EXPECT().AIcallGet(ctx, id).Return(terminated, nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, aicall.EventTypeStatusTerminated, terminated)
				}
			}

			res, err := h.ProcessTerminate(ctx, id)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error/nil, got: %v/%v", err, res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.Status != aicall.StatusTerminated {
				t.Errorf("Wrong result. got: %v", res)
			}
		})
	}
}

func Test_ProcessTerminate_getFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("6d3e0c10-3c00-4c00-8000-000000000051")

	mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, fmt.Errorf("select failed"))

	res, err := h.ProcessTerminate(ctx, id)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error/nil, got: %v/%v", err, res)
	}
}
