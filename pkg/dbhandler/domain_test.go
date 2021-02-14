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
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainCreate(context.Background(), tt.domain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			if err := h.DomainDelete(context.Background(), tt.domain.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().DomainGet(gomock.Any(), tt.domain.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().DomainSet(gomock.Any(), gomock.Any())
			res, err := h.DomainGet(context.Background(), tt.domain.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
		})
	}
}
