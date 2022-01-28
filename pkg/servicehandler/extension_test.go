package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestExtensionCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name     string
		customer *cscustomer.Customer

		extension    *extension.Extension
		reqExtension *rmextension.Extension

		response  *rmextension.Extension
		expectRes *extension.Extension
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&extension.Extension{
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				DomainID:   uuid.FromStringOrNil("19835af8-6fa4-11eb-b553-0317e16bca16"),
				Name:       "test",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
			},
			&rmextension.Extension{
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				DomainID:   uuid.FromStringOrNil("19835af8-6fa4-11eb-b553-0317e16bca16"),
				Name:       "test",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("4037dd90-6fa4-11eb-b51b-771a2747271b"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				DomainID:   uuid.FromStringOrNil("19835af8-6fa4-11eb-b553-0317e16bca16"),
				Name:       "test",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
				TMCreate:   "2020-09-20 03:23:20.995000",
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("4037dd90-6fa4-11eb-b51b-771a2747271b"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				DomainID:   uuid.FromStringOrNil("19835af8-6fa4-11eb-b553-0317e16bca16"),
				Name:       "test",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
				TMCreate:   "2020-09-20 03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1ExtensionCreate(gomock.Any(), tt.reqExtension).Return(tt.response, nil)

			res, err := h.ExtensionCreate(tt.customer, tt.extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestExtensionUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		customer  *cscustomer.Customer
		extension *extension.Extension

		requestExt *rmextension.Extension
		response   *rmextension.Extension
		expectRes  *extension.Extension
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&extension.Extension{
				ID:       uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
			},
			&rmextension.Extension{
				ID:       uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "update name",
				Detail:     "update detail",
				DomainID:   uuid.FromStringOrNil("94b2c8b6-6fa5-11eb-8416-779aeb05f3ef"),
				Extension:  "test",
				Password:   "update password",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "2020-09-20 03:23:23.995000",
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "update name",
				Detail:     "update detail",
				DomainID:   uuid.FromStringOrNil("94b2c8b6-6fa5-11eb-8416-779aeb05f3ef"),
				Extension:  "test",
				Password:   "update password",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "2020-09-20 03:23:23.995000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1ExtensionGet(gomock.Any(), tt.extension.ID).Return(&rmextension.Extension{CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988")}, nil)
			mockReq.EXPECT().RMV1ExtensionUpdate(gomock.Any(), tt.requestExt).Return(tt.response, nil)
			res, err := h.ExtensionUpdate(tt.customer, tt.extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestExtensionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name        string
		customer    *cscustomer.Customer
		extensionID uuid.UUID

		response *rmextension.Extension
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("aa1fda4e-6fa6-11eb-8385-a3288e16c056"),

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("aa1fda4e-6fa6-11eb-8385-a3288e16c056"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),

				Name:     "test",
				Detail:   "test detail",
				DomainID: uuid.FromStringOrNil("c1412796-6fa6-11eb-a7d0-576218199a69"),

				Extension: "test",
				Password:  "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1ExtensionGet(gomock.Any(), tt.extensionID).Return(tt.response, nil)
			mockReq.EXPECT().RMV1ExtensionDelete(gomock.Any(), tt.extensionID).Return(nil)

			if err := h.ExtensionDelete(tt.customer, tt.extensionID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestExtensionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name        string
		customer    *cscustomer.Customer
		extensionID uuid.UUID

		response  *rmextension.Extension
		expectRes *extension.Extension
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),

				Name:     "test",
				Detail:   "test detail",
				DomainID: uuid.FromStringOrNil("4537748e-6fab-11eb-85b1-7faa1af90353"),
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				DomainID:   uuid.FromStringOrNil("4537748e-6fab-11eb-85b1-7faa1af90353"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1ExtensionGet(gomock.Any(), tt.extensionID).Return(tt.response, nil)
			res, err := h.ExtensionGet(tt.customer, tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestExtensionGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		customer  *cscustomer.Customer
		domainID  uuid.UUID
		pageToken string
		pageSize  uint64

		response  []rmextension.Extension
		expectRes []*extension.Extension
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("6def6d14-6fab-11eb-9d25-eb0e5ebe6fdc"),
			"2020-10-20T01:00:00.995000",
			10,

			[]rmextension.Extension{
				{
					ID:         uuid.FromStringOrNil("7f88a068-6fab-11eb-916e-f3b27367df79"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					DomainID:   uuid.FromStringOrNil("6def6d14-6fab-11eb-9d25-eb0e5ebe6fdc"),
				},
				{
					ID:         uuid.FromStringOrNil("7f003a16-6fab-11eb-9b0b-9fe0fc962219"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test2",
					Detail:     "test detail2",
					DomainID:   uuid.FromStringOrNil("6def6d14-6fab-11eb-9d25-eb0e5ebe6fdc"),
				},
			},
			[]*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("7f88a068-6fab-11eb-916e-f3b27367df79"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					DomainID:   uuid.FromStringOrNil("6def6d14-6fab-11eb-9d25-eb0e5ebe6fdc"),
				},
				{
					ID:         uuid.FromStringOrNil("7f003a16-6fab-11eb-9b0b-9fe0fc962219"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test2",
					Detail:     "test detail2",
					DomainID:   uuid.FromStringOrNil("6def6d14-6fab-11eb-9d25-eb0e5ebe6fdc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1ExtensionGets(gomock.Any(), tt.domainID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.ExtensionGets(tt.customer, tt.domainID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
