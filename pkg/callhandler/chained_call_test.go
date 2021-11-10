package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestChainedCallIDAdd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name         string
		call         *call.Call
		chaindCallID uuid.UUID
	}

	tests := []test{
		{
			"call status progressing",
			&call.Call{
				ID:     uuid.FromStringOrNil("eb71954c-2504-11eb-a92f-0bd8129658a9"),
				Status: call.StatusProgressing,
			},

			uuid.FromStringOrNil("ed893c22-2504-11eb-a0ed-839c010855ed"),
		},
		{
			"call status dialing",
			&call.Call{
				ID:     uuid.FromStringOrNil("a31ccbbe-256c-11eb-8d6a-7b6b14b71912"),
				Status: call.StatusDialing,
			},

			uuid.FromStringOrNil("a396a86c-256c-11eb-b3ab-d708913b7832"),
		},
		{
			"call status ringing",
			&call.Call{
				ID:     uuid.FromStringOrNil("a3cc1dd0-256c-11eb-afd6-871abdb9c625"),
				Status: call.StatusRinging,
			},

			uuid.FromStringOrNil("a4025af8-256c-11eb-bd53-9b3a0b24844d"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.call.ID, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(gomock.Any(), tt.chaindCallID, tt.call.ID).Return(nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

			if err := h.ChainedCallIDAdd(tt.call.ID, tt.chaindCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestChainedCallIDAddFailStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name         string
		call         *call.Call
		chaindCallID uuid.UUID
	}

	tests := []test{
		{
			"call status terminating",
			&call.Call{
				ID:     uuid.FromStringOrNil("aa6e29e8-256d-11eb-8433-6ffc4a89784b"),
				Status: call.StatusTerminating,
			},

			uuid.FromStringOrNil("aaa92570-256d-11eb-b535-8fa2fdbf4dd0"),
		},
		{
			"call status canceling",
			&call.Call{
				ID:     uuid.FromStringOrNil("aadeebd8-256d-11eb-8b7d-13422cdc50af"),
				Status: call.StatusCanceling,
			},

			uuid.FromStringOrNil("ab13f30a-256d-11eb-bb85-e76164670011"),
		},
		{
			"call status hangup",
			&call.Call{
				ID:     uuid.FromStringOrNil("ab4f22e0-256d-11eb-a3e1-9be571423d11"),
				Status: call.StatusHangup,
			},

			uuid.FromStringOrNil("ab8784d2-256d-11eb-89fe-8f088d13b59f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), false)

			if err := h.ChainedCallIDAdd(tt.call.ID, tt.chaindCallID); err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func TestChainedCallIDRemove(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name         string
		call         *call.Call
		chaindCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID: uuid.FromStringOrNil("4786ca48-256c-11eb-be3c-7361101fde14"),
			},

			uuid.FromStringOrNil("47cb9c0e-256c-11eb-aff4-7b1173c946b1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXRemoveChainedCallID(gomock.Any(), tt.call.ID, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(gomock.Any(), tt.chaindCallID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

			if err := h.ChainedCallIDRemove(tt.call.ID, tt.chaindCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
