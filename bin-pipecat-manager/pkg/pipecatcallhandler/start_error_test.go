package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Start_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-2001-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d4e5f6a7-2001-4000-8000-000000000002")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-2001-4000-8000-000000000003")

	start := func(h *pipecatcallHandler) (*pipecatcall.Pipecatcall, error) {
		return h.Start(
			context.Background(), pcID, customerID, uuid.Nil,
			pipecatcall.ReferenceTypeAICall, referenceID,
			pipecatcall.LLMType("openai.gpt-5"), nil,
			pipecatcall.STTTypeNone, "", pipecatcall.TTSTypeNone, "", "",
		)
	}

	t.Run("db create fails and nothing else runs", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no other expectation: any request, publish or session work fails the test.
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &pipecatcallHandler{
			db:                    mockDB,
			requestHandler:        requesthandler.NewMockRequestHandler(mc),
			notifyHandler:         notifyhandler.NewMockNotifyHandler(mc),
			mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
			muPipecatcallSession:  sync.Mutex{},
		}

		mockDB.EXPECT().PipecatcallCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert failed"))

		res, err := start(h)
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not create pipecatcall") || !strings.Contains(err.Error(), "insert failed") {
			t.Errorf("wrong error: %v", err)
		}
		if len(h.mapPipecatcallSession) != 0 {
			t.Errorf("no session may be created")
		}
	})

	t.Run("ai call lookup fails after the record is created", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{
			db:                    mockDB,
			requestHandler:        mockReq,
			notifyHandler:         mockNotify,
			mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
			muPipecatcallSession:  sync.Mutex{},
		}

		created := &pipecatcall.Pipecatcall{
			Identity:      commonidentity.Identity{ID: pcID, CustomerID: customerID},
			ReferenceType: pipecatcall.ReferenceTypeAICall,
			ReferenceID:   referenceID,
			LLMType:       pipecatcall.LLMType("openai.gpt-5"),
		}
		mockDB.EXPECT().PipecatcallCreate(gomock.Any(), gomock.Any()).Return(nil)
		mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(created, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypeCreated, created)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall lookup failed"))

		res, err := start(h)
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
		if err == nil ||
			!strings.Contains(err.Error(), "could not start reference type ai call") ||
			!strings.Contains(err.Error(), "could not get ai call info") ||
			!strings.Contains(err.Error(), "aicall lookup failed") {
			t.Errorf("wrong error: %v", err)
		}
		if len(h.mapPipecatcallSession) != 0 {
			t.Errorf("no session may be created")
		}
	})
}

func Test_startReferenceTypeAIcall_duplicateSession(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-2002-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("b2c3d4e5-2002-4000-8000-000000000002")
	aiID := uuid.FromStringOrNil("c3d4e5f6-2002-4000-8000-000000000003")
	callID := uuid.FromStringOrNil("d4e5f6a7-2002-4000-8000-000000000004")

	tests := []struct {
		name string

		aicallReferenceType amaicall.ReferenceType
	}{
		{name: "call reference branch", aicallReferenceType: amaicall.ReferenceTypeCall},
		{name: "default branch", aicallReferenceType: amaicall.ReferenceTypeTask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// only the lookups are expected. The external media start, the python runner and
			// every other collaborator are left without expectations, so reaching them fails.
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &pipecatcallHandler{
				requestHandler:        mockReq,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   aicallID,
				LLMType:       pipecatcall.LLMType("openai.gpt-5"),
			}

			// a session with the same id (the key is pc.ID in both branches) already exists.
			existing := &pipecatcall.Session{Identity: commonidentity.Identity{ID: pcID}}
			h.mapPipecatcallSession[pcID] = existing

			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
				Identity:       commonidentity.Identity{ID: aicallID},
				ReferenceType:  tt.aicallReferenceType,
				ReferenceID:    callID,
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   aiID,
			}, nil)
			mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
				Identity:    commonidentity.Identity{ID: aiID},
				Type:        amai.TypeNormal,
				EngineModel: amai.EngineModel("openai.gpt-5"),
				EngineKey:   "synthetic-key",
			}, nil)

			err := h.startReferenceTypeAIcall(context.Background(), pc)
			if err == nil ||
				!strings.Contains(err.Error(), "could not create pipecatcall session") ||
				!strings.Contains(err.Error(), "session already exists") {
				t.Errorf("wrong error: %v", err)
			}
			if got := h.mapPipecatcallSession[pcID]; got != existing {
				t.Errorf("the existing session must be left untouched")
			}
		})
	}
}
