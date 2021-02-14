package domainhandler

import (
	"context"
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
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
		domain *models.Domain
	}

	tests := []test{
		{
			"test normal",
			&models.Domain{
				UserID:     1,
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
