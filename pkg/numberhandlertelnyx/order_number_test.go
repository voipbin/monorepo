package numberhandlertelnyx

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler/models/telnyx"
)

func TestCreateNumberByTelnyxOrderNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := numberHandler{
		reqHandler: mockReq,
		db:         mockDB,
		cache:      mockCache,
	}

	type test struct {
		name   string
		userID uint64
		number string

		phoneNumbers []*telnyx.PhoneNumber
	}

	tests := []test{
		{
			"normal",
			1,
			"+821021656521",
			[]*telnyx.PhoneNumber{
				{
					ID:                    "1580568175064384684",
					RecordType:            "phone_number",
					PhoneNumber:           "+12704940136",
					Status:                telnyx.PhoneNumberStatusActive,
					Tags:                  []string{},
					ConnectionID:          ConnectionID,
					T38FaxGatewayEnabled:  true,
					PurchasedAt:           "2021-02-26T18:26:49Z",
					EmergencyEnabled:      false,
					CallForwardingEnabled: true,
					CNAMListingEnabled:    false,
					CallRecordingEnabled:  false,
					CreatedAt:             "2021-02-26T18:26:49.277Z",
					UpdatedAt:             "2021-02-27T17:07:16.234Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().TelnyxPhoneNumbersGet(uint(1), "", tt.number).Return(tt.phoneNumbers, nil)
			mockReq.EXPECT().TelnyxPhoneNumbersIDUpdateConnectionID(tt.phoneNumbers[0].ID, ConnectionID).Return(tt.phoneNumbers[0], nil)
			mockDB.EXPECT().NumberCreate(gomock.Any(), gomock.Any())
			mockDB.EXPECT().NumberGet(gomock.Any(), gomock.Any())
			_, err := h.createNumberByTelnyxOrderNumber(tt.userID, tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestReleaseNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := numberHandler{
		reqHandler: mockReq,
		db:         mockDB,
		cache:      mockCache,
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

			mockReq.EXPECT().TelnyxPhoneNumbersIDDelete(tt.number.ProviderReferenceID)
			mockDB.EXPECT().NumberDelete(gomock.Any(), tt.number.ID)
			mockDB.EXPECT().NumberGet(gomock.Any(), tt.number.ID)
			_, err := h.ReleaseOrderNumber(ctx, tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
