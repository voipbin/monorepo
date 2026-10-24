package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func newTerminateTestHandler() *pipecatcallHandler {
	return &pipecatcallHandler{
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}
}

// registerTerminateTestSession registers a session whose Asterisk connection is "ready"
// but nil, so SessionStop has nothing to close.
func registerTerminateTestSession(t *testing.T, h *pipecatcallHandler, id uuid.UUID) *pipecatcall.Session {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	connAstReady := make(chan struct{})
	close(connAstReady)

	se := &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: id},
		Ctx:          ctx,
		Cancel:       cancel,
		ConnAstReady: connAstReady,
	}
	h.mapPipecatcallSession[id] = se
	return se
}

func Test_Terminate_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-1001-4000-8000-000000000001")

	t.Run("get fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := newTerminateTestHandler()
		h.db = mockDB

		mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(nil, fmt.Errorf("db down"))

		res, err := h.Terminate(context.Background(), pcID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
		if !strings.Contains(err.Error(), "could not get pipecatcall info") || !strings.Contains(err.Error(), "db down") {
			t.Errorf("wrong error: %v", err)
		}
	})

	t.Run("registered session is torn down and the event is published once", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		mockPython := NewMockPythonRunner(mc)
		h := newTerminateTestHandler()
		h.db = mockDB
		h.notifyHandler = mockNotify
		h.pythonRunner = mockPython

		pc := &pipecatcall.Pipecatcall{
			Identity:      commonidentity.Identity{ID: pcID},
			ReferenceType: pipecatcall.ReferenceType("unknown"),
		}
		registerTerminateTestSession(t, h, pcID)

		mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(pc, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypePipecatcallTerminated, pc).Times(1)
		mockPython.EXPECT().Stop(gomock.Any(), pcID).Return(nil).Times(1)

		res, err := h.Terminate(context.Background(), pcID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != pc {
			t.Errorf("wrong result. expect: %v, got: %v", pc, res)
		}
		if _, errGet := h.SessionGet(pcID); errGet == nil {
			t.Errorf("the session must be removed by terminate")
		}
	})
}

func Test_terminate_cleanupOutcome(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-1002-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1002-4000-8000-000000000002")

	tests := []struct {
		name string

		referenceType pipecatcall.ReferenceType
		setup         func(mockReq *requesthandler.MockRequestHandler)

		expectSessionRemoved bool
	}{
		{
			name:          "call cleanup fails keeps the session",
			referenceType: pipecatcall.ReferenceTypeCall,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("call lookup failed"))
			},
			expectSessionRemoved: false,
		},
		{
			name:          "call cleanup succeeds removes the session",
			referenceType: pipecatcall.ReferenceTypeCall,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{Status: cmcall.StatusHangup}, nil)
			},
			expectSessionRemoved: true,
		},
		{
			name:          "ai call cleanup fails keeps the session",
			referenceType: pipecatcall.ReferenceTypeAICall,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall lookup failed"))
			},
			expectSessionRemoved: false,
		},
		{
			name:          "ai call cleanup succeeds removes the session",
			referenceType: pipecatcall.ReferenceTypeAICall,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{ReferenceType: amaicall.ReferenceTypeTask}, nil)
			},
			expectSessionRemoved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockPython := NewMockPythonRunner(mc)
			h := newTerminateTestHandler()
			h.requestHandler = mockReq
			h.notifyHandler = mockNotify
			h.pythonRunner = mockPython

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID},
				ReferenceType: tt.referenceType,
				ReferenceID:   referenceID,
			}
			se := registerTerminateTestSession(t, h, pcID)

			// the event is published before the cleanup, whatever its outcome.
			mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypePipecatcallTerminated, pc).Times(1)
			tt.setup(mockReq)
			if tt.expectSessionRemoved {
				mockPython.EXPECT().Stop(gomock.Any(), pcID).Return(nil).Times(1)
			}

			h.terminate(context.Background(), pc)

			_, errGet := h.SessionGet(pcID)
			if tt.expectSessionRemoved {
				if errGet == nil {
					t.Errorf("the session must be removed")
				}
				if se.Ctx.Err() == nil {
					t.Errorf("the session context must be cancelled")
				}
				return
			}
			if errGet != nil {
				t.Errorf("the session must be kept when the cleanup fails: %v", errGet)
			}
			if se.Ctx.Err() != nil {
				t.Errorf("the session must not be cancelled when the cleanup fails")
			}
		})
	}
}

func Test_terminateReferenceTypeCall_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-1003-4000-8000-000000000001")
	callID := uuid.FromStringOrNil("b2c3d4e5-1003-4000-8000-000000000002")

	tests := []struct {
		name string

		getResult  *cmcall.Call
		getErr     error
		stopErr    error
		expectStop bool

		expectErr string
	}{
		{name: "call lookup fails", getErr: fmt.Errorf("call lookup failed"), expectErr: "could not get call info: call lookup failed"},
		{name: "not progressing does not stop", getResult: &cmcall.Call{Status: cmcall.StatusHangup}},
		{name: "progressing stops the external media of the pipecatcall id", getResult: &cmcall.Call{Status: cmcall.StatusProgressing}, expectStop: true},
		{name: "stop fails", getResult: &cmcall.Call{Status: cmcall.StatusProgressing}, expectStop: true, stopErr: fmt.Errorf("stop failed"), expectErr: "could not stop external media: stop failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &pipecatcallHandler{requestHandler: mockReq}

			pc := &pipecatcall.Pipecatcall{
				Identity:    commonidentity.Identity{ID: pcID},
				ReferenceID: callID,
			}

			mockReq.EXPECT().CallV1CallGet(gomock.Any(), callID).Return(tt.getResult, tt.getErr)
			if tt.expectStop {
				var em *cmexternalmedia.ExternalMedia
				if tt.stopErr == nil {
					em = &cmexternalmedia.ExternalMedia{ID: pcID}
				}
				// the pipecatcall id (not the call id) identifies the external media.
				mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), pcID).Return(em, tt.stopErr).Times(1)
			}

			err := h.terminateReferenceTypeCall(context.Background(), pc)
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("wrong error. expect contains: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_terminateReferenceTypeAICall_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-1004-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("b2c3d4e5-1004-4000-8000-000000000002")

	tests := []struct {
		name string

		getResult  *amaicall.AIcall
		getErr     error
		stopErr    error
		expectStop bool

		expectErr string
	}{
		{name: "ai call lookup fails", getErr: fmt.Errorf("aicall lookup failed"), expectErr: "could not get ai call info: aicall lookup failed"},
		{name: "call reference stops the external media of the pipecatcall id", getResult: &amaicall.AIcall{ReferenceType: amaicall.ReferenceTypeCall}, expectStop: true},
		{name: "call reference stop fails", getResult: &amaicall.AIcall{ReferenceType: amaicall.ReferenceTypeCall}, expectStop: true, stopErr: fmt.Errorf("stop failed"), expectErr: "could not stop external media: stop failed"},
		{name: "other reference does not stop", getResult: &amaicall.AIcall{ReferenceType: amaicall.ReferenceTypeConversation}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &pipecatcallHandler{requestHandler: mockReq}

			pc := &pipecatcall.Pipecatcall{
				Identity:    commonidentity.Identity{ID: pcID},
				ReferenceID: aicallID,
			}

			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(tt.getResult, tt.getErr)
			if tt.expectStop {
				var em *cmexternalmedia.ExternalMedia
				if tt.stopErr == nil {
					em = &cmexternalmedia.ExternalMedia{ID: pcID}
				}
				mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), pcID).Return(em, tt.stopErr).Times(1)
			}

			err := h.terminateReferenceTypeAICall(context.Background(), pc)
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("wrong error. expect contains: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}
