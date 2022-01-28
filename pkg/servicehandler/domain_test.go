package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestDomainCreate(t *testing.T) {
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

		DomainName      string
		DomainTmpName   string
		DomainTmpDetail string

		response  *rmdomain.Domain
		expectRes *domain.Domain
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			"test.sip.voipbin.net",
			"test name",
			"test detail",
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1DomainCreate(gomock.Any(), tt.customer.ID, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail).Return(tt.response, nil)

			res, err := h.DomainCreate(tt.customer, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestDomainUpdate(t *testing.T) {
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
		domain   *domain.Domain

		requestDomain *rmdomain.Domain
		response      *rmdomain.Domain
		expectRes     *domain.Domain
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1DomainGet(gomock.Any(), tt.domain.ID).Return(&rmdomain.Domain{CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3")}, nil)
			mockReq.EXPECT().RMV1DomainUpdate(gomock.Any(), tt.requestDomain).Return(tt.response, nil)
			res, err := h.DomainUpdate(tt.customer, tt.domain)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestDomainDelete(t *testing.T) {
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
		domainID uuid.UUID

		response *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1DomainGet(gomock.Any(), tt.domainID).Return(tt.response, nil)
			mockReq.EXPECT().RMV1DomainDelete(gomock.Any(), tt.domainID).Return(nil)

			if err := h.DomainDelete(tt.customer, tt.domainID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestDomainGet(t *testing.T) {
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
		DomainID uuid.UUID

		response  *rmdomain.Domain
		expectRes *domain.Domain
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1DomainGet(gomock.Any(), tt.DomainID).Return(tt.response, nil)

			res, err := h.DomainGet(tt.customer, tt.DomainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestDomainGets(t *testing.T) {
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
		pageToken string
		pageSize  uint64

		response  []rmdomain.Domain
		expectRes []*domain.Domain
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]rmdomain.Domain{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
				},
			},
			[]*domain.Domain{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMV1DomainGets(gomock.Any(), tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.DomainGets(tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
