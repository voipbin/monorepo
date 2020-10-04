package listenhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
)

func TestV1ActiveFlowsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name         string
		request      *rabbitmqhandler.Request
		expectCallID uuid.UUID
		expectFlowID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"call_id": "1d8dacf4-05ee-11eb-9eae-037ddd66443e", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},
			uuid.FromStringOrNil("1d8dacf4-05ee-11eb-9eae-037ddd66443e"),
			uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().ActiveFlowCreate(gomock.Any(), tt.expectCallID, tt.expectFlowID).Return(nil, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}
