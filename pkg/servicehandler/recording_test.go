package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestRecordingGets(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		size  uint64
		token string

		response  []cmrecording.Recording
		expectRes []*cmrecording.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			10,
			"2020-10-20 01:00:00.995000",

			[]cmrecording.Recording{
				{
					ID:         uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Filename:   "call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
				},
				{
					ID:         uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
					Filename:   "call_2f167946-b2b4-4370-94fa-d6c2c57c84da_2020-12-04T18:48:03Z",
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				},
			},

			[]*cmrecording.WebhookMessage{
				{
					ID:          uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
					ReferenceID: uuid.Nil,
				},
				{
					ID:          uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
					ReferenceID: uuid.Nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CallV1RecordingGets(gomock.Any(), tt.customer.ID, tt.size, tt.token).Return(tt.response, nil)

			res, err := h.RecordingGets(tt.customer, tt.size, tt.token)

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res[0])
			}
		})
	}
}
