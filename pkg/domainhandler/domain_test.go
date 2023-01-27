package domainhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		domainName string
		domainN    string
		detail     string

		responseUUID   uuid.UUID
		responseDomain *domain.Domain

		expectDomain *domain.Domain
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("202b2592-8967-11ec-aeab-3336a440f2c1"),
			"test-domain",
			"test name",
			"test detail",

			uuid.FromStringOrNil("d7e1a4ce-8b8b-4b28-a300-5ded30918882"),
			&domain.Domain{
				ID: uuid.FromStringOrNil("d7e1a4ce-8b8b-4b28-a300-5ded30918882"),
			},

			&domain.Domain{
				ID:         uuid.FromStringOrNil("d7e1a4ce-8b8b-4b28-a300-5ded30918882"),
				CustomerID: uuid.FromStringOrNil("202b2592-8967-11ec-aeab-3336a440f2c1"),
				DomainName: "test-domain",
				Name:       "test name",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &domainHandler{
				utilHandler:   mockUtil,
				dbAst:         mockDBAst,
				dbBin:         mockDBBin,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDBBin.EXPECT().DomainGetByDomainName(ctx, tt.expectDomain.DomainName).Return(nil, fmt.Errorf(""))
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDBBin.EXPECT().DomainCreate(ctx, tt.expectDomain)
			mockDBBin.EXPECT().DomainGet(ctx, tt.expectDomain.ID).Return(tt.responseDomain, nil)
			mockNotify.EXPECT().PublishEvent(ctx, domain.EventTypeDomainCreated, tt.responseDomain)

			res, err := h.Create(ctx, tt.customerID, tt.domainName, tt.domainN, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDomain, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDomain, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseDomain *domain.Domain
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507"),

			&domain.Domain{
				CustomerID: uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507"),
			},
		},
	}

	for _, tt := range tests {
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

		ctx := context.Background()

		mockDBBin.EXPECT().DomainGet(ctx, tt.id).Return(tt.responseDomain, nil)
		res, err := h.Get(ctx, tt.id)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.responseDomain, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDomain, res)
		}
	}
}

func Test_GetByDomainName(t *testing.T) {

	type test struct {
		name string

		domainName string

		responseDomain *domain.Domain
	}

	tests := []test{
		{
			"normal",

			"test",

			&domain.Domain{
				CustomerID: uuid.FromStringOrNil("b6ce1618-a5d5-4fe2-a3f4-981c98543175"),
			},
		},
	}

	for _, tt := range tests {
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

		ctx := context.Background()

		mockDBBin.EXPECT().DomainGetByDomainName(ctx, tt.domainName).Return(tt.responseDomain, nil)
		res, err := h.GetByDomainName(ctx, tt.domainName)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.responseDomain, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDomain, res)
		}
	}
}

func Test_Update(t *testing.T) {

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

func Test_Delete(t *testing.T) {

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
