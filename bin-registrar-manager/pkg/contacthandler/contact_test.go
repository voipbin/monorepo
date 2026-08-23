package contacthandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_ContactListByDomainID(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		ext        string

		responseEndpoint string
		responseContacts []*astcontact.AstContact
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("8815c2c0-5652-11ee-b3a1-0b960b5e92e8"),
			"testexten",

			"testexten@8815c2c0-5652-11ee-b3a1-0b960b5e92e8.registrar.voipbin.net",
			[]*astcontact.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "testexten@8815c2c0-5652-11ee-b3a1-0b960b5e92e8.registrar.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
		},
		{
			"short domain",

			uuid.FromStringOrNil("9a2b3c4d-5652-11ee-b3a1-0b960b5e92e8"),
			"1001",

			"1001@ab12.reg.voipbin.net",
			[]*astcontact.AstContact{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
			h := &contactHandler{
				dbAst:                 mockDBAst,
				dbBin:                 mockDBBin,
				customerDomainHandler: mockCustomerDomain,
			}
			ctx := context.Background()

			// read path: lookup-only EndpointGet (never EnsureByCustomerID)
			mockCustomerDomain.EXPECT().EndpointGet(ctx, tt.customerID, tt.ext).Return(tt.responseEndpoint, nil)
			mockDBAst.EXPECT().AstContactGetsByEndpoint(ctx, tt.responseEndpoint).Return(tt.responseContacts, nil)
			res, err := h.ContactGetsByExtension(ctx, tt.customerID, tt.ext)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContacts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContacts, res)
			}
		})

	}
}

func Test_ContactGetsByExtension_endpointError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	h := &contactHandler{
		dbAst:                 mockDBAst,
		dbBin:                 mockDBBin,
		customerDomainHandler: mockCustomerDomain,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aa15c2c0-5652-11ee-b3a1-0b960b5e92e8")

	mockCustomerDomain.EXPECT().EndpointGet(ctx, customerID, "1001").Return("", fmt.Errorf("db down"))
	_, err := h.ContactGetsByExtension(ctx, customerID, "1001")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_ContactRefreshByEndpoint(t *testing.T) {

	type test struct {
		name string

		customerID       uuid.UUID
		extension        string
		responseEndpoint string
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("dc16be1a-5706-11ee-8481-23541c418477"),
			"test-extension",
			"test-extension@dc16be1a-5706-11ee-8481-23541c418477.registrar.voipbin.net",
		},
		{
			"short domain",

			uuid.FromStringOrNil("ec16be1a-5706-11ee-8481-23541c418477"),
			"2001",
			"2001@zz99.reg.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
			h := &contactHandler{
				dbAst:                 mockDBAst,
				dbBin:                 mockDBBin,
				customerDomainHandler: mockCustomerDomain,
			}
			ctx := context.Background()

			// read path: lookup-only EndpointGet (never EnsureByCustomerID)
			mockCustomerDomain.EXPECT().EndpointGet(ctx, tt.customerID, tt.extension).Return(tt.responseEndpoint, nil)
			mockDBAst.EXPECT().AstContactDeleteFromCache(ctx, tt.responseEndpoint).Return(nil)
			if err := h.ContactRefreshByEndpoint(ctx, tt.customerID, tt.extension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ContactRefreshByEndpoint_endpointError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	h := &contactHandler{
		dbAst:                 mockDBAst,
		dbBin:                 mockDBBin,
		customerDomainHandler: mockCustomerDomain,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("fc16be1a-5706-11ee-8481-23541c418477")

	mockCustomerDomain.EXPECT().EndpointGet(ctx, customerID, "2001").Return("", fmt.Errorf("db down"))
	if err := h.ContactRefreshByEndpoint(ctx, customerID, "2001"); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
