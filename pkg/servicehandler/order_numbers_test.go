package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/nmnumber"
)

func TestOrderNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *models.User
		pageToken string
		pageSize  uint64

		response  []nmnumber.Number
		expectRes []*models.Number
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]nmnumber.Number{
				{
					ID:                  uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
					Number:              "+821021656521",
					UserID:              1,
					ProviderName:        "telnyx",
					ProviderReferenceID: "",
					Status:              nmnumber.NumberStatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			[]*models.Number{
				{
					ID:               uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
					Number:           "+821021656521",
					UserID:           0,
					Status:           "active",
					T38Enabled:       false,
					EmergencyEnabled: false,
					TMPurchase:       "",
					TMCreate:         "",
					TMUpdate:         "",
					TMDelete:         "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().NMOrderNumberGets(tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			_, err := h.OrderNumberGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestOrderNumberCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name    string
		user    *models.User
		numbers string

		response  *nmnumber.Number
		expectRes *models.Number
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			"+821021656521",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("f06c8c36-7b1d-11eb-8b01-83e94e91b409"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.NumberStatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&models.Number{
				Number:           "+821021656521",
				UserID:           1,
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ctx := context.Background()

			mockReq.EXPECT().NMOrderNumberCreate(tt.user.ID, tt.numbers).Return(tt.response, nil)

			res, err := h.OrderNumberCreate(tt.user, tt.numbers)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.ID = uuid.Nil
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
