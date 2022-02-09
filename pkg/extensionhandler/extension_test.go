package extensionhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

func TestExtensionCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}

	type test struct {
		name     string
		ext      *extension.Extension
		domain   *domain.Domain
		aor      *astaor.AstAOR
		auth     *astauth.AstAuth
		endpoint *astendpoint.AstEndpoint
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				DomainID:   uuid.FromStringOrNil("ce060aae-6ec1-11eb-a550-cb46a3229b89"),
				Extension:  "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Password:   "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",
			},
			&domain.Domain{
				ID:         uuid.FromStringOrNil("ce060aae-6ec1-11eb-a550-cb46a3229b89"),
				DomainName: "test.sip.voipbin.net",
			},
			&astaor.AstAOR{
				ID:             getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				MaxContacts:    getIntegerPointer(1),
				RemoveExisting: getStringPointer("yes"),
			},
			&astauth.AstAuth{
				ID:       getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				AuthType: getStringPointer("userpass"),
				Username: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4"),
				Password: getStringPointer("cf6917ba-6ec1-11eb-8810-e3829c2dfab8"),
				Realm:    getStringPointer("test.sip.voipbin.net"),
			},
			&astendpoint.AstEndpoint{
				ID:   getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				AORs: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				Auth: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().DomainGet(gomock.Any(), tt.ext.DomainID).Return(tt.domain, nil)
		mockDBAst.EXPECT().AstAORCreate(gomock.Any(), tt.aor).Return(nil)
		mockDBAst.EXPECT().AstAuthCreate(gomock.Any(), tt.auth).Return(nil)
		mockDBAst.EXPECT().AstEndpointCreate(gomock.Any(), tt.endpoint).Return(nil)
		mockDBBin.EXPECT().ExtensionCreate(gomock.Any(), gomock.Any()).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), gomock.Any()).Return(tt.ext, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), extension.EventTypeExtensionCreated, tt.ext)
		_, err := h.ExtensionCreate(ctx, tt.ext)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestExtensionUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}

	type test struct {
		name       string
		ext        *extension.Extension
		updateAuth *astauth.AstAuth
		updateExt  *extension.Extension
		updatedExt *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.sip.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},
			&astauth.AstAuth{
				ID:       getStringPointer("66f6b86c-6f44-11eb-ab55-934942c23f91@test.sip.voipbin.net"),
				Password: getStringPointer("update password"),
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.sip.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.sip.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), tt.updateExt.ID).Return(tt.ext, nil)
		mockDBAst.EXPECT().AstAuthUpdate(gomock.Any(), tt.updateAuth).Return(nil)
		mockDBBin.EXPECT().ExtensionUpdate(gomock.Any(), tt.updateExt)
		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), tt.ext.ID).Return(tt.updatedExt, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), extension.EventTypeExtensionUpdated, tt.updateExt)
		_, err := h.ExtensionUpdate(ctx, tt.updateExt)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestExtensionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}

	type test struct {
		name string
		ext  *extension.Extension

		expectRes *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("4a6b7618-6f46-11eb-a2fb-1f7595db4195"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "test name",
				Detail:     "test detail",
				AuthID:     "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				EndpointID: "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				AORID:      "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				Extension:  "4a6b7618-6f46-11eb-a2fb-1f7595db4195",
				Password:   "test password",
			},

			&extension.Extension{
				ID:         uuid.FromStringOrNil("4a6b7618-6f46-11eb-a2fb-1f7595db4195"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "test name",
				Detail:     "test detail",
				AuthID:     "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				EndpointID: "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				AORID:      "4a6b7618-6f46-11eb-a2fb-1f7595db4195@test.sip.voipbin.net",
				Extension:  "4a6b7618-6f46-11eb-a2fb-1f7595db4195",
				Password:   "test password",
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(tt.ext, nil)
		mockDBBin.EXPECT().ExtensionDelete(ctx, tt.ext.ID).Return(nil)
		mockDBAst.EXPECT().AstEndpointDelete(ctx, tt.ext.EndpointID).Return(nil)
		mockDBAst.EXPECT().AstAuthDelete(ctx, tt.ext.AuthID).Return(nil)
		mockDBAst.EXPECT().AstAORDelete(ctx, tt.ext.AORID).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(tt.ext, nil)
		mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionDeleted, tt.ext)

		res, err := h.ExtensionDelete(ctx, tt.ext.ID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.expectRes, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
		}
	}
}

func TestExtensionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{
		dbAst: mockDBAst,
		dbBin: mockDBBin,
	}

	type test struct {
		name string
		ext  *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("798f8bcc-6f47-11eb-8908-efd77279298d"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "test name",
				Detail:     "test detail",
				AuthID:     "798f8bcc-6f47-11eb-8908-efd77279298d@test.sip.voipbin.net",
				EndpointID: "798f8bcc-6f47-11eb-8908-efd77279298d@test.sip.voipbin.net",
				AORID:      "798f8bcc-6f47-11eb-8908-efd77279298d@test.sip.voipbin.net",
				Extension:  "798f8bcc-6f47-11eb-8908-efd77279298d",
				Password:   "test password",
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(tt.ext, nil)
		res, err := h.ExtensionGet(ctx, tt.ext.ID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if reflect.DeepEqual(tt.ext, res) == false {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.ext, res)
		}

	}
}

func TestExtensionGetsByDomainID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{
		dbAst: mockDBAst,
		dbBin: mockDBBin,
	}

	type test struct {
		name     string
		domainID uuid.UUID
		token    string
		exts     []*extension.Extension
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("bd57214a-6f4b-11eb-aad8-579de27e6b7f"),
			"2021-02-15 17:31:59.519672",
			[]*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("c9c736a4-6f4b-11eb-899a-575b7ce222e6"),
					CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
					Name:       "test name",
					Detail:     "test detail",
					DomainID:   uuid.FromStringOrNil("bd57214a-6f4b-11eb-aad8-579de27e6b7f"),
					AuthID:     "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
					EndpointID: "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
					AORID:      "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
					Extension:  "d1f16192-6f4b-11eb-83aa-27a0be9dffd1",
					Password:   "test password",
					TMCreate:   "2021-02-14 17:31:59.519672",
				},
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGetsByDomainID(gomock.Any(), tt.domainID, tt.token, uint64(10)).Return(tt.exts, nil)
		res, err := h.ExtensionGetsByDomainID(ctx, tt.domainID, tt.token, uint64(10))
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if reflect.DeepEqual(tt.exts, res) == false {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.exts, res)
		}

	}
}
