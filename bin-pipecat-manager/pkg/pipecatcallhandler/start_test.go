package pipecatcallhandler

import (
	"context"
	"fmt"
	"sync"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall_callGetFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet fails
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(nil, fmt.Errorf("call not found"))

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func Test_startReferenceTypeCall_externalMediaFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockPythonRunner := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		pythonRunner:          mockPythonRunner,
		toolHandler:           mockTool,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
	callID := uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet succeeds
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(&cmcall.Call{
			Identity: commonidentity.Identity{
				ID: callID,
			},
		}, nil)

	// CallV1ExternalMediaStart fails
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(),
		pcID,
		cmexternalmedia.ReferenceTypeCall,
		callID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	).Return(nil, fmt.Errorf("external media creation failed"))

	// RunnerStart goroutine may call these before context is cancelled
	mockTool.EXPECT().GetAll().AnyTimes()
	mockPythonRunner.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockPythonRunner.EXPECT().Stop(gomock.Any(), gomock.Any()).AnyTimes()

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func Test_startReferenceTypeCall_websocketDialFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWS := NewMockWebsocketHandler(mc)
	mockPythonRunner := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      mockWS,
		pythonRunner:          mockPythonRunner,
		toolHandler:           mockTool,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
	callID := uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666")
	emID := uuid.FromStringOrNil("e5f6a7b8-1111-2222-3333-444455556666")
	mediaURI := "ws://asterisk:8088/ws/test-media"

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet succeeds
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(&cmcall.Call{
			Identity: commonidentity.Identity{
				ID: callID,
			},
		}, nil)

	// CallV1ExternalMediaStart succeeds
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(),
		pcID,
		cmexternalmedia.ReferenceTypeCall,
		callID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	).Return(&cmexternalmedia.ExternalMedia{
		ID:       emID,
		MediaURI: mediaURI,
	}, nil)

	// WebSocket dial fails
	mockWS.EXPECT().DialContext(gomock.Any(), mediaURI, gomock.Any()).
		Return(nil, nil, fmt.Errorf("connection refused"))

	// Cleanup: CallV1ExternalMediaStop should be called with em.ID
	mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), emID).
		Return(&cmexternalmedia.ExternalMedia{ID: emID}, nil)

	// RunnerStart goroutine may call these before context is cancelled
	mockTool.EXPECT().GetAll().AnyTimes()
	mockPythonRunner.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockPythonRunner.EXPECT().Stop(gomock.Any(), gomock.Any()).AnyTimes()

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
