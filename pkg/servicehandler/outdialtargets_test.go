package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
)

func Test_OutdialtargetCreate(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer

		outdialID         uuid.UUID
		outdialtargetName string
		detail            string
		data              string

		destination0 *cmaddress.Address
		destination1 *cmaddress.Address
		destination2 *cmaddress.Address
		destination3 *cmaddress.Address
		destination4 *cmaddress.Address

		responseOutdial *omoutdial.Outdial
		response        *omoutdialtarget.OutdialTarget
		expectRes       *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("410fa394-8764-4300-a8b0-a6e6108c4208"),
			"test name",
			"test detail",
			"test data",

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000002",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000003",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000004",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000005",
			},

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("410fa394-8764-4300-a8b0-a6e6108c4208"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
			},
			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
			},
		},
		{
			"has 1 address",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("7aff596d-1db2-4456-95b4-bdab04296cd8"),
			"test name",
			"test detail",
			"test data",

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			nil,
			nil,
			nil,
			nil,

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("7aff596d-1db2-4456-95b4-bdab04296cd8"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("895b86d9-4e58-4778-af61-d389dbeb9cf7"),
			},
			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("895b86d9-4e58-4778-af61-d389dbeb9cf7"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OMV1OutdialtargetCreate(gomock.Any(), tt.outdialID, tt.outdialtargetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4).Return(tt.response, nil)
			res, err := h.OutdialtargetCreate(tt.customer, tt.outdialID, tt.outdialtargetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialtargetDelete(t *testing.T) {

	tests := []struct {
		name            string
		customer        *cscustomer.Customer
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		responseOutdial *omoutdial.Outdial
		response        *omoutdialtarget.OutdialTarget
		expectRes       *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			},
			&omoutdialtarget.WebhookMessage{
				ID:        uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OMV1OutdialtargetGet(gomock.Any(), tt.outdialtargetID).Return(tt.response, nil)
			mockReq.EXPECT().OMV1OutdialtargetDelete(gomock.Any(), tt.outdialtargetID).Return(tt.response, nil)
			res, err := h.OutdialtargetDelete(tt.customer, tt.outdialID, tt.outdialtargetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
