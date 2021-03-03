package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/rmdomain"
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
		name string
		user *models.User

		DomainName      string
		DomainTmpName   string
		DomainTmpDetail string

		response  *rmdomain.Domain
		expectRes *models.Domain
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			"test.sip.voipbin.net",
			"test name",
			"test detail",
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMDomainCreate(tt.user.ID, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail).Return(tt.response, nil)

			res, err := h.DomainCreate(tt.user, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail)
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
		name   string
		user   *models.User
		domain *models.Domain

		requestDomain *rmdomain.Domain
		response      *rmdomain.Domain
		expectRes     *models.Domain
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			&models.Domain{
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
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMDomainGet(tt.domain.ID).Return(&rmdomain.Domain{UserID: 1}, nil)
			mockReq.EXPECT().RMDomainUpdate(tt.requestDomain).Return(tt.response, nil)
			res, err := h.DomainUpdate(tt.user, tt.domain)
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
		user     *models.User
		domainID uuid.UUID

		response *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMDomainGet(tt.domainID).Return(tt.response, nil)
			mockReq.EXPECT().RMDomainDelete(tt.domainID).Return(nil)

			if err := h.DomainDelete(tt.user, tt.domainID); err != nil {
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
		user     *models.User
		DomainID uuid.UUID

		response  *rmdomain.Domain
		expectRes *models.Domain
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMDomainGet(tt.DomainID).Return(tt.response, nil)

			res, err := h.DomainGet(tt.user, tt.DomainID)
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
		user      *models.User
		pageToken string
		pageSize  uint64

		response  []rmdomain.Domain
		expectRes []*models.Domain
	}

	tests := []test{
		{
			"normal",
			&models.User{
				ID: 1,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]rmdomain.Domain{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					UserID:     1,
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					UserID:     1,
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
				},
			},
			[]*models.Domain{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					UserID:     1,
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					UserID:     1,
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().RMDomainGets(tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.DomainGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
