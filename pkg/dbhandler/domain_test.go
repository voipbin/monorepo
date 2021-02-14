package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func TestDomainCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		domain       *models.Domain
		expectDomain *models.Domain
	}

	tests := []test{
		{
			"test normal",
			&models.Domain{
				ID:         uuid.FromStringOrNil("f8f75f20-6e0c-11eb-8dba-435a87e89b48"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("f8f75f20-6e0c-11eb-8dba-435a87e89b48"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
			},
		},
		{
			"with name detail",
			&models.Domain{
				ID:         uuid.FromStringOrNil("d55f111a-6edf-11eb-b978-277f5400b4e8"),
				UserID:     1,
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test1.sip.voipbin.net",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("d55f111a-6edf-11eb-b978-277f5400b4e8"),
				UserID:     1,
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test1.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainCreate(context.Background(), tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(gomock.Any(), tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			res, err := h.DomainGet(context.Background(), tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectDomain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomain, res)
			}
		})
	}
}

func TestDomainDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name   string
		domain *models.Domain
	}

	tests := []test{
		{
			"test normal",
			&models.Domain{
				ID:         uuid.FromStringOrNil("81b73c40-6e0d-11eb-9a4b-0fe8ac8ec4c3"),
				UserID:     1,
				DomainName: "81b73c40-6e0d-11eb-9a4b-0fe8ac8ec4c3.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainCreate(ctx, tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainDelete(ctx, tt.domain.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(gomock.Any(), tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			res, err := h.DomainGet(ctx, tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
		})
	}
}

func TestDomainGetByDomainName(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name   string
		domain *models.Domain
	}

	tests := []test{
		{
			"test normal",
			&models.Domain{
				ID:         uuid.FromStringOrNil("3e765cc0-6ee1-11eb-b9e9-33589a46f50e"),
				UserID:     1,
				DomainName: "5140d1b4-6ee1-11eb-b35e-03eb172540ec.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainCreate(ctx, tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.DomainGetByDomainName(ctx, tt.domain.DomainName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.domain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.domain, res)
			}
		})
	}
}

func TestDomainGetsByUserID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name          string
		userID        uint64
		limit         uint64
		domains       []models.Domain
		expectDomains []*models.Domain
	}

	tests := []test{
		{
			"have no actions",
			4,
			10,
			[]models.Domain{
				{
					ID:         uuid.FromStringOrNil("ef2f65b8-6ee4-11eb-a688-dbb959113359"),
					UserID:     4,
					Name:       "test1",
					DomainName: "ef2f65b8-6ee4-11eb-a688-dbb959113359.sip.voipbin.net",
				},
				{
					ID:         uuid.FromStringOrNil("05c29e76-6ee5-11eb-bc50-6b162fbf37b3"),
					UserID:     4,
					Name:       "test2",
					DomainName: "05c29e76-6ee5-11eb-bc50-6b162fbf37b3.sip.voipbin.net",
				},
			},
			[]*models.Domain{
				{
					ID:         uuid.FromStringOrNil("05c29e76-6ee5-11eb-bc50-6b162fbf37b3"),
					UserID:     4,
					Name:       "test2",
					DomainName: "05c29e76-6ee5-11eb-bc50-6b162fbf37b3.sip.voipbin.net",
				},
				{
					ID:         uuid.FromStringOrNil("ef2f65b8-6ee4-11eb-a688-dbb959113359"),
					UserID:     4,
					Name:       "test1",
					DomainName: "ef2f65b8-6ee4-11eb-a688-dbb959113359.sip.voipbin.net",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			for _, d := range tt.domains {
				mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
				if err := h.DomainCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			domains, err := h.DomainGetsByUserID(ctx, tt.userID, getCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, d := range domains {
				d.TMCreate = ""
			}

			if reflect.DeepEqual(domains, tt.expectDomains) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomains, domains)
			}
		})
	}
}

func TestDomainUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		domain       *models.Domain
		updateDomain *models.Domain
		expectDomain *models.Domain
	}

	tests := []test{
		{
			"test normal",
			&models.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				UserID:     1,
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				UserID:     1,
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("8e11791c-6eec-11eb-9d29-835387182e69"),
				UserID:     1,
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "8e11791c-6eec-11eb-9d29-835387182e69.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainCreate(ctx, tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainUpdate(ctx, tt.updateDomain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(gomock.Any(), tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			res, err := h.DomainGet(context.Background(), tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectDomain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomain, res)
			}
		})
	}
}
