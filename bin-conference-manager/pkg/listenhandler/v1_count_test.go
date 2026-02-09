package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

func Test_processV1ConferencesCountByCustomerGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		responseCount int
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/conferences/count_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b2c3d4e5-0001-0001-0001-000000000001"}`),
			},

			customerID:    uuid.FromStringOrNil("b2c3d4e5-0001-0001-0001-000000000001"),
			responseCount: 5,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":5}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConference := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				conferenceHandler: mockConference,
			}

			mockConference.EXPECT().CountByCustomerID(gomock.Any(), tt.customerID).Return(tt.responseCount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
