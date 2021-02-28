package numberhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

func TestCreateOrderNumbersTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name    string
		userID  uint64
		numbers []string

		expectRes []*models.Number
	}

	tests := []test{
		{
			"normal us",
			1,
			[]string{"+821021656521"},
			[]*models.Number{
				{
					ID:           uuid.FromStringOrNil("84cdc0a8-79d8-11eb-9179-ffc8c4fc9756"),
					Number:       "+821021656521",
					ProviderName: models.NumberProviderNameTelnyx,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTelnyx.EXPECT().CreateOrderNumbers(tt.userID, tt.numbers).Return(tt.expectRes, nil)

			res, err := h.CreateOrderNumbers(tt.userID, tt.numbers)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// func TestOrderNumbers(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	mockDB := dbhandler.NewMockDBHandler(mc)
// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	h := NewNumberHandler(mockReq, mockDB, mockCache)

// 	type test struct {
// 		name      string
// 		userID    uint64
// 		numbers   []string
// 		expectRes []*models.Number
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			1,
// 			[]string{"+821021656521"},
// 			[]*models.Number{
// 				{
// 					ID:                  [16]byte{},
// 					Number:              "",
// 					FlowID:              [16]byte{},
// 					UserID:              0,
// 					ProviderName:        "",
// 					ProviderReferenceID: "",
// 					Status:              "",
// 					T38Enabled:          false,
// 					EmergencyEnabled:    false,
// 					TMPurchase:          "",
// 					TMCreate:            "",
// 					TMUpdate:            "",
// 					TMDelete:            "",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockReq.EXPECT().TelnyxNumberOrdersPost(tt.numbers)
// 			for _, num := range tt.numbers {
// 				mockReq.EXPECT().TelnyxPhoneNumbersGet(1, "", num).Return([]*telnyx.PhoneNumber{{ID: "b954ea5e-7924-11eb-b7e9-f3f26145e121"}}, nil)
// 				mockReq.EXPECT().TelnyxPhoneNumbersIDUpdateConnectionID("b954ea5e-7924-11eb-b7e9-f3f26145e121", ConnectionID).Return(&telnyx.PhoneNumber{}, nil)

// 				tmp := &telnyx.PhoneNumber{}
// 				mockDB.EXPECT().NumberCreate(gomock.Any(), tmp.ConvertNumber())

// 			}

// 			res, err := h.OrderNumbers(tt.userID, tt.numbers)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectRes, res) != true {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
