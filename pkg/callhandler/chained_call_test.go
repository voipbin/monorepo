package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestChainedCallIDAdd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name         string
		id           uuid.UUID
		chaindCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("eb71954c-2504-11eb-a92f-0bd8129658a9"),
			uuid.FromStringOrNil("ed893c22-2504-11eb-a0ed-839c010855ed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallAddChainedCallID(gomock.Any(), tt.id, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(gomock.Any(), tt.chaindCallID, tt.id).Return(nil)
			if err := h.ChainedCallIDAdd(tt.id, tt.chaindCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestChainedCallIDRemove(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name         string
		id           uuid.UUID
		chaindCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("7c1020dc-2505-11eb-a65d-ebd04dfb0fe9"),
			uuid.FromStringOrNil("7ef1feba-2505-11eb-8b6d-2f7e05afeb84"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallRemoveChainedCallID(gomock.Any(), tt.id, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(gomock.Any(), tt.chaindCallID, uuid.Nil).Return(nil)
			if err := h.ChainedCallIDRemove(tt.id, tt.chaindCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
