package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func Test_DomainCreate(t *testing.T) {

	type test struct {
		name   string
		domain *domain.Domain

		responseCurTime string
		expectRes       *domain.Domain
	}

	tests := []test{
		{
			"normal",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("f8f75f20-6e0c-11eb-8dba-435a87e89b48"),
				CustomerID: uuid.FromStringOrNil("5a58b664-7fec-11ec-81a8-eb20974bc536"),
				DomainName: "test.sip.voipbin.net",
			},

			"2021-02-26 18:26:49.000",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("f8f75f20-6e0c-11eb-8dba-435a87e89b48"),
				CustomerID: uuid.FromStringOrNil("5a58b664-7fec-11ec-81a8-eb20974bc536"),
				DomainName: "test.sip.voipbin.net",
				TMCreate:   "2021-02-26 18:26:49.000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
			},
		},
		{
			"with name detail",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("d55f111a-6edf-11eb-b978-277f5400b4e8"),
				CustomerID: uuid.FromStringOrNil("62d63960-7fec-11ec-8e8b-7be1888fdeeb"),
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test1.sip.voipbin.net",
			},

			"2021-02-26 18:26:49.000",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("d55f111a-6edf-11eb-b978-277f5400b4e8"),
				CustomerID: uuid.FromStringOrNil("62d63960-7fec-11ec-8e8b-7be1888fdeeb"),
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test1.sip.voipbin.net",
				TMCreate:   "2021-02-26 18:26:49.000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().DomainSet(ctx, gomock.Any())
			if err := h.DomainCreate(ctx, tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(ctx, tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(ctx, gomock.Any())
			res, err := h.DomainGet(ctx, tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainGetByDomainName(t *testing.T) {
	type test struct {
		name   string
		domain *domain.Domain

		responseCurTime string

		expectRes *domain.Domain
	}

	tests := []test{
		{
			"normal",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("3e765cc0-6ee1-11eb-b9e9-33589a46f50e"),
				CustomerID: uuid.FromStringOrNil("718fdf92-7fec-11ec-8408-dba09d1a7bd2"),
				DomainName: "5140d1b4-6ee1-11eb-b35e-03eb172540ec.sip.voipbin.net",
			},

			"2021-02-26 18:26:49.000",

			&domain.Domain{
				ID:         uuid.FromStringOrNil("3e765cc0-6ee1-11eb-b9e9-33589a46f50e"),
				CustomerID: uuid.FromStringOrNil("718fdf92-7fec-11ec-8408-dba09d1a7bd2"),
				DomainName: "5140d1b4-6ee1-11eb-b35e-03eb172540ec.sip.voipbin.net",
				TMCreate:   "2021-02-26 18:26:49.000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if errCreate := h.DomainCreate(ctx, tt.domain); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockCache.EXPECT().DomainGetByDomainName(ctx, tt.domain.DomainName).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			res, err := h.DomainGetByDomainName(ctx, tt.domain.DomainName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainGetsByCustomerID(t *testing.T) {
	type test struct {
		name       string
		customerID uuid.UUID
		limit      uint64
		domains    []domain.Domain

		responseCurTime string

		expectRes []*domain.Domain
	}

	tests := []test{
		{
			"have no actions",
			uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
			10,
			[]domain.Domain{
				{
					ID:         uuid.FromStringOrNil("ef2f65b8-6ee4-11eb-a688-dbb959113359"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					Name:       "test1",
					DomainName: "ef2f65b8-6ee4-11eb-a688-dbb959113359.sip.voipbin.net",
				},
				{
					ID:         uuid.FromStringOrNil("05c29e76-6ee5-11eb-bc50-6b162fbf37b3"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					Name:       "test2",
					DomainName: "05c29e76-6ee5-11eb-bc50-6b162fbf37b3.sip.voipbin.net",
				},
			},

			"2021-02-26 18:26:49.000",

			[]*domain.Domain{
				{
					ID:         uuid.FromStringOrNil("ef2f65b8-6ee4-11eb-a688-dbb959113359"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					Name:       "test1",
					DomainName: "ef2f65b8-6ee4-11eb-a688-dbb959113359.sip.voipbin.net",
					TMCreate:   "2021-02-26 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("05c29e76-6ee5-11eb-bc50-6b162fbf37b3"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					Name:       "test2",
					DomainName: "05c29e76-6ee5-11eb-bc50-6b162fbf37b3.sip.voipbin.net",
					TMCreate:   "2021-02-26 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, d := range tt.domains {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
				if err := h.DomainCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			domains, err := h.DomainGetsByCustomerID(ctx, tt.customerID, utilhandler.TimeGetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(domains, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, domains)
			}
		})
	}
}

func Test_DomainUpdateBasicInfo(t *testing.T) {

	type test struct {
		name   string
		domain *domain.Domain

		id      uuid.UUID
		domainN string
		detail  string

		responseCurTime string

		updateDomain *domain.Domain
		expectDomain *domain.Domain
	}

	tests := []test{
		{
			"test normal",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",
			},

			uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
			"update name",
			"update detail",

			"2021-02-26 18:26:49.000",

			&domain.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",

				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: "2021-02-26 18:26:49.000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().DomainSet(ctx, gomock.Any())
			if err := h.DomainCreate(ctx, tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().DomainSet(ctx, gomock.Any())
			if err := h.DomainUpdateBasicInfo(ctx, tt.id, tt.domainN, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(ctx, tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(ctx, gomock.Any())
			res, err := h.DomainGet(ctx, tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectDomain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomain, res)
			}
		})
	}
}
