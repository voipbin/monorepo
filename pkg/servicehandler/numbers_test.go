package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
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
		user      *user.User
		pageToken string
		pageSize  uint64

		response  []nmnumber.Number
		expectRes []*number.Number
	}

	tests := []test{
		{
			"normal",
			&user.User{
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
					Status:              nmnumber.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			[]*number.Number{
				{
					ID:               uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().NMNumberGets(tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.NumberGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = ""
				num.TMUpdate = ""
				num.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestOrderNumberGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		response  *nmnumber.Number
		expectRes *number.Number
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&number.Number{
				ID:               uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				Number:           "+821021656521",
				UserID:           1,
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().NMNumberGet(tt.id).Return(tt.response, nil)

			res, err := h.NumberGet(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestOrderNumberGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		response *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            "2021-03-02 01:00:00.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().NMNumberGet(tt.id).Return(tt.response, nil)

			_, err := h.NumberGet(tt.user, tt.id)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func TestNumberCreate(t *testing.T) {
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
		user    *user.User
		numbers string

		response  *nmnumber.Number
		expectRes *number.Number
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"+821021656521",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("f06c8c36-7b1d-11eb-8b01-83e94e91b409"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&number.Number{
				Number:           "+821021656521",
				UserID:           1,
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ctx := context.Background()

			mockReq.EXPECT().NMNumberCreate(tt.user.ID, tt.numbers).Return(tt.response, nil)

			res, err := h.NumberCreate(tt.user, tt.numbers)
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

func TestNumberDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		responseGet    *nmnumber.Number
		responseDelete *nmnumber.Number
		expectRes      *number.Number
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusDeleted,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMCreate:            "2021-10-15 00:00:00.000001",
				TMUpdate:            "2021-10-16 00:00:00.000001",
				TMDelete:            "2021-10-16 00:00:00.000001",
			},
			&number.Number{
				ID:               uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				Number:           "+821021656521",
				UserID:           1,
				Status:           "deleted",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "2021-10-15 00:00:00.000001",
				TMUpdate:         "2021-10-16 00:00:00.000001",
				TMDelete:         "2021-10-16 00:00:00.000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().NMNumberGet(tt.id).Return(tt.responseGet, nil)
			mockReq.EXPECT().NMNumberDelete(tt.id).Return(tt.responseDelete, nil)

			res, err := h.NumberDelete(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestNumberUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name         string
		user         *user.User
		updateNumber *number.Number

		updateNMNumber *nmnumber.Number
		responseGet    *nmnumber.Number
		responseUpdate *nmnumber.Number
		expectRes      *number.Number
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				FlowID: uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
			},

			&nmnumber.Number{
				ID:     uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				FlowID: uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				FlowID:              uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&number.Number{
				ID:               uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				FlowID:           uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
				Number:           "+821021656521",
				UserID:           1,
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().NMNumberGet(tt.updateNumber.ID).Return(tt.responseGet, nil)
			mockReq.EXPECT().NMNumberUpdate(tt.updateNMNumber).Return(tt.responseUpdate, nil)

			res, err := h.NumberUpdate(tt.user, tt.updateNumber)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestNumberUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name         string
		user         *user.User
		updateNumber *number.Number

		responseGet *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&user.User{
				ID: 1,
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				FlowID: uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
			},

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            "2021-03-02 01:00:00.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().NMNumberGet(tt.updateNumber.ID).Return(tt.responseGet, nil)

			_, err := h.NumberUpdate(tt.user, tt.updateNumber)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}
