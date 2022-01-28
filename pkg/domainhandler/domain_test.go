package domainhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
)

func TestDomainCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &domainHandler{
		dbAst: mockDBAst,
		dbBin: mockDBBin,
	}

	type test struct {
		name   string
		domain *domain.Domain
	}

	tests := []test{
		{
			"test normal",
			&domain.Domain{
				CustomerID: uuid.FromStringOrNil("e2531ce4-7fec-11ec-ae7d-4fc565b03cba"),
				DomainName: "test.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().DomainGetByDomainName(gomock.Any(), tt.domain.DomainName).Return(nil, fmt.Errorf(""))
		mockDBBin.EXPECT().DomainCreate(gomock.Any(), gomock.Any())
		mockDBBin.EXPECT().DomainGet(gomock.Any(), gomock.Any())
		_, err := h.DomainCreate(ctx, tt.domain)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestDomainUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &domainHandler{
		dbAst: mockDBAst,
		dbBin: mockDBBin,
	}

	type test struct {
		name   string
		domain *domain.Domain
	}

	tests := []test{
		{
			"test normal",
			&domain.Domain{
				ID:         uuid.FromStringOrNil("43b1c268-6eed-11eb-87ce-8f9d9ae03b04"),
				CustomerID: uuid.FromStringOrNil("ea7bf81e-7fec-11ec-ab22-bf62853d679c"),
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "43b1c268-6eed-11eb-87ce-8f9d9ae03b04.sip.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().DomainUpdate(gomock.Any(), tt.domain)
		mockDBBin.EXPECT().DomainGet(gomock.Any(), tt.domain.ID)
		_, err := h.DomainUpdate(ctx, tt.domain)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestDomainDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockExt := extensionhandler.NewMockExtensionHandler(mc)
	h := &domainHandler{
		dbAst:      mockDBAst,
		dbBin:      mockDBBin,
		extHandler: mockExt,
	}

	type test struct {
		name     string
		domainID uuid.UUID
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockExt.EXPECT().ExtensionDeleteByDomainID(gomock.Any(), tt.domainID).Return(nil)
		mockDBBin.EXPECT().DomainDelete(gomock.Any(), tt.domainID)
		if err := h.DomainDelete(ctx, tt.domainID); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}
