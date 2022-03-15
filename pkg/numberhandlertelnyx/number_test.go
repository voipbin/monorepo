package numberhandlertelnyx

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal/models/telnyx"
)

func Test_CreateNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := numberHandlerTelnyx{
		reqHandler:      mockReq,
		db:              mockDB,
		requestExternal: mockExternal,
	}

	type test struct {
		name string

		customerID uuid.UUID
		flowID     uuid.UUID
		number     string
		numberName string
		detail     string

		responseOrder  *telnyx.OrderNumber
		responseNumber *telnyx.PhoneNumber

		expectRes *number.Number
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("42975e92-7ff4-11ec-a6f9-0b55edda8dc3"),
			uuid.FromStringOrNil("039bb60e-8821-11ec-86b3-8f80fcaf5d9f"),
			"+821021656521",
			"test name",
			"test detail",

			&telnyx.OrderNumber{
				PhoneNumbers: []telnyx.OrderNumberPhoneNumber{
					{
						ID:          "1748688147379652251",
						PhoneNumber: "+821021656521",
						Status:      "active",
					},
				},
			},
			&telnyx.PhoneNumber{
				ID:                    "1748688147379652251",
				RecordType:            "phone_number",
				PhoneNumber:           "+12704940136",
				Status:                telnyx.PhoneNumberStatusActive,
				Tags:                  []string{},
				ConnectionID:          "tmp connection id",
				T38FaxGatewayEnabled:  true,
				PurchasedAt:           "2021-02-26T18:26:49Z",
				EmergencyEnabled:      false,
				CallForwardingEnabled: true,
				CNAMListingEnabled:    false,
				CallRecordingEnabled:  false,
				CreatedAt:             "2021-02-26T18:26:49.277Z",
				UpdatedAt:             "2021-02-27T17:07:16.234Z",
			},

			&number.Number{
				Number:              "+12704940136",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "1748688147379652251",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			numbers := []string{tt.number}
			mockExternal.EXPECT().TelnyxNumberOrdersPost(numbers).Return(tt.responseOrder, nil)
			mockExternal.EXPECT().TelnyxPhoneNumbersIDGet(tt.responseOrder.PhoneNumbers[0].ID).Return(tt.responseNumber, nil)
			res, err := h.CreateNumber(tt.customerID, tt.number, tt.flowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestReleaseNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := numberHandlerTelnyx{
		reqHandler:      mockReq,
		db:              mockDB,
		requestExternal: mockExternal,
	}

	type test struct {
		name   string
		number *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("d8659476-79e1-11eb-a59b-9301c8a84847"),
				ProviderReferenceID: "1580568175064384684",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockExternal.EXPECT().TelnyxPhoneNumbersIDDelete(tt.number.ProviderReferenceID)
			mockDB.EXPECT().NumberDelete(gomock.Any(), tt.number.ID)
			mockDB.EXPECT().NumberGet(gomock.Any(), tt.number.ID)
			_, err := h.ReleaseNumber(ctx, tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
