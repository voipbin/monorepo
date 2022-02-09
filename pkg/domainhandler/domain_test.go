package domainhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
)

func TestCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &domainHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}

	type test struct {
		name       string
		customerID uuid.UUID

		domainName string
		domainN    string
		detail     string

		domain *domain.Domain
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("202b2592-8967-11ec-aeab-3336a440f2c1"),

			"test.sip.voipbin.net",
			"test name",
			"test detail",

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
		mockDBBin.EXPECT().DomainGet(gomock.Any(), gomock.Any()).Return(&domain.Domain{}, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), domain.EventTypeDomainCreated, gomock.Any())
		_, err := h.Create(ctx, tt.customerID, tt.domainName, tt.domainN, tt.detail)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &domainHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}

	type test struct {
		name string

		id      uuid.UUID
		domainN string
		detail  string

		domain *domain.Domain
	}

	tests := []test{
		{
			"test normal",

			uuid.FromStringOrNil("43b1c268-6eed-11eb-87ce-8f9d9ae03b04"),
			"update name",
			"update detail",

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

		mockDBBin.EXPECT().DomainUpdateBasicInfo(gomock.Any(), tt.id, tt.domainN, tt.detail).Return(nil)
		mockDBBin.EXPECT().DomainGet(gomock.Any(), tt.domain.ID).Return(tt.domain, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), domain.EventTypeDomainUpdated, tt.domain)
		_, err := h.Update(ctx, tt.id, tt.domainN, tt.detail)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockExt := extensionhandler.NewMockExtensionHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &domainHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		extHandler:    mockExt,
		notifyHandler: mockNotify,
	}

	type test struct {
		name string

		domainID uuid.UUID

		responseDomain *domain.Domain
		expectRes      *domain.Domain
	}

	tests := []test{
		{
			"test normal",

			uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),

			&domain.Domain{
				ID: uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),
			},
			&domain.Domain{
				ID: uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockExt.EXPECT().ExtensionDeleteByDomainID(gomock.Any(), tt.domainID).Return([]*extension.Extension{}, nil)
		mockDBBin.EXPECT().DomainDelete(gomock.Any(), tt.domainID).Return(nil)
		mockDBBin.EXPECT().DomainGet(gomock.Any(), tt.domainID).Return(tt.responseDomain, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), domain.EventTypeDomainDeleted, tt.responseDomain)
		res, err := h.Delete(ctx, tt.domainID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.expectRes, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
		}

	}
}
