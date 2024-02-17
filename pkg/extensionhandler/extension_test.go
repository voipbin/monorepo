package extensionhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		extName    string
		detail     string
		ext        string
		password   string

		responseDomain          *domain.Domain
		responseUUIDExtensionID uuid.UUID
		responseExtension       *extension.Extension

		expectAOR       *astaor.AstAOR
		expectAuth      *astauth.AstAuth
		expectEndpoint  *astendpoint.AstEndpoint
		expectExtension *extension.Extension
		expectSIPAuth   *sipauth.SIPAuth
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
			extName:    "test name",
			detail:     "test detail",
			ext:        "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
			password:   "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",

			responseDomain: &domain.Domain{
				ID:         uuid.FromStringOrNil("ce060aae-6ec1-11eb-a550-cb46a3229b89"),
				DomainName: "test",
			},
			responseUUIDExtensionID: uuid.FromStringOrNil("b2fce137-6ece-4259-8480-473b6c1f2dee"),
			responseExtension: &extension.Extension{
				ID:        uuid.FromStringOrNil("b2fce137-6ece-4259-8480-473b6c1f2dee"),
				Extension: "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Realm:     "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:  "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Password:  "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",
			},

			expectAOR: &astaor.AstAOR{
				// ID:             getStringPointer("7515a10e-5959-11ee-a4f2-3f55a7e37970"),
				ID:             getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
				MaxContacts:    getIntegerPointer(defaultMaxContacts),
				RemoveExisting: getStringPointer("yes"),
			},
			expectAuth: &astauth.AstAuth{
				ID:       getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
				AuthType: getStringPointer("userpass"),
				Username: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4"),
				Password: getStringPointer("cf6917ba-6ec1-11eb-8810-e3829c2dfab8"),
				Realm:    getStringPointer("0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
			},
			expectEndpoint: &astendpoint.AstEndpoint{
				ID:   getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
				AORs: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
				Auth: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net"),
			},
			expectExtension: &extension.Extension{
				ID:         uuid.FromStringOrNil("b2fce137-6ece-4259-8480-473b6c1f2dee"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "test name",
				Detail:     "test detail",
				EndpointID: "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				AORID:      "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				AuthID:     "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Extension:  "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				DomainName: "0040713e-7fed-11ec-954b-ff6d17e2a264",
				Realm:      "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:   "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Password:   "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",
			},
			expectSIPAuth: &sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("b2fce137-6ece-4259-8480-473b6c1f2dee"),
				ReferenceType: sipauth.ReferenceTypeExtension,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:      "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Password:      "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",
				AllowedIPs:    []string{},
				TMCreate:      "",
				TMUpdate:      "",
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &extensionHandler{
			utilHandler:   mockUtil,
			dbAst:         mockDBAst,
			dbBin:         mockDBBin,
			notifyHandler: mockNotify,
		}

		ctx := context.Background()

		mockDBAst.EXPECT().AstAORCreate(ctx, tt.expectAOR).Return(nil)
		mockDBAst.EXPECT().AstAuthCreate(ctx, tt.expectAuth).Return(nil)
		mockDBAst.EXPECT().AstEndpointCreate(ctx, tt.expectEndpoint).Return(nil)
		mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDExtensionID)
		mockDBBin.EXPECT().ExtensionCreate(ctx, tt.expectExtension).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(ctx, tt.expectExtension.ID).Return(tt.responseExtension, nil)
		mockDBBin.EXPECT().SIPAuthCreate(ctx, tt.expectSIPAuth).Return(nil)
		mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionCreated, tt.responseExtension)

		res, err := h.Create(ctx, tt.customerID, tt.extName, tt.detail, tt.ext, tt.password)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(res, tt.responseExtension) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExtension, res)
		}
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseExtension *extension.Extension
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("b38b9d45-f81d-4505-b9ef-9f44da1860cf"),

			&extension.Extension{
				ID: uuid.FromStringOrNil("b38b9d45-f81d-4505-b9ef-9f44da1860cf"),
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &extensionHandler{
			utilHandler:   mockUtil,
			dbAst:         mockDBAst,
			dbBin:         mockDBBin,
			notifyHandler: mockNotify,
		}
		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(ctx, tt.id).Return(tt.responseExtension, nil)

		res, err := h.Get(ctx, tt.id)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(res, tt.responseExtension) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExtension, res)
		}
	}
}

// func Test_GetByEndpoint(t *testing.T) {

// 	type test struct {
// 		name string

// 		endpoint string

// 		expectEndpointID  string
// 		responseExtension *extension.Extension
// 	}

// 	tests := []test{
// 		{
// 			"normal",

// 			"test_ext@test_domain",

// 			"test_ext@test_domain.sip.voipbin.net",
// 			&extension.Extension{
// 				ID: uuid.FromStringOrNil("256c7fd2-e461-4871-83c0-8f60ab3acb84"),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		mc := gomock.NewController(t)
// 		defer mc.Finish()

// 		mockUtil := utilhandler.NewMockUtilHandler(mc)
// 		mockDBAst := dbhandler.NewMockDBHandler(mc)
// 		mockDBBin := dbhandler.NewMockDBHandler(mc)
// 		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 		h := &extensionHandler{
// 			utilHandler:   mockUtil,
// 			dbAst:         mockDBAst,
// 			dbBin:         mockDBBin,
// 			notifyHandler: mockNotify,
// 		}
// 		ctx := context.Background()

// 		mockDBBin.EXPECT().ExtensionGetByEndpointID(ctx, tt.expectEndpointID).Return(tt.responseExtension, nil)

// 		res, err := h.GetByEndpoint(ctx, tt.endpoint)
// 		if err != nil {
// 			t.Errorf("Wrong match. expect: ok, got: %v", err)
// 		}

// 		if !reflect.DeepEqual(res, tt.responseExtension) {
// 			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExtension, res)
// 		}
// 	}
// }

func Test_Update(t *testing.T) {

	type test struct {
		name string

		id            uuid.UUID
		extensionName string
		detail        string
		password      string

		responseExtension *extension.Extension
		updateAuth        *astauth.AstAuth
		updateExt         *extension.Extension
		updatedExt        *extension.Extension

		expectSIPAuth *sipauth.SIPAuth
	}

	tests := []test{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
			extensionName: "update name",
			detail:        "update detail",
			password:      "update password",

			responseExtension: &extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.registrar.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				DomainName: "",
				Realm:      "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:   "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},

			updateAuth: &astauth.AstAuth{
				ID:       getStringPointer("66f6b86c-6f44-11eb-ab55-934942c23f91@test.registrar.voipbin.net"),
				Password: getStringPointer("update password"),
			},
			updateExt: &extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.registrar.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},
			updatedExt: &extension.Extension{
				ID:         uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
				Name:       "update name",
				Detail:     "update detail",
				AuthID:     "66f6b86c-6f44-11eb-ab55-934942c23f91@test.registrar.voipbin.net",
				Extension:  "66f6b86c-6f44-11eb-ab55-934942c23f91",
				DomainName: "",
				Realm:      "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:   "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
			},

			expectSIPAuth: &sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91"),
				ReferenceType: sipauth.ReferenceTypeExtension,
				AuthTypes: []sipauth.AuthType{
					sipauth.AuthTypeBasic,
				},
				Realm:      "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
				Username:   "66f6b86c-6f44-11eb-ab55-934942c23f91",
				Password:   "update password",
				AllowedIPs: []string{},
			},
		},
	}

	for _, tt := range tests {
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

		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), tt.updateExt.ID).Return(tt.responseExtension, nil)
		mockDBAst.EXPECT().AstAuthUpdate(gomock.Any(), tt.updateAuth).Return(nil)
		mockDBBin.EXPECT().ExtensionUpdate(gomock.Any(), tt.id, tt.extensionName, tt.detail, tt.password).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), tt.responseExtension.ID).Return(tt.updatedExt, nil)
		mockDBBin.EXPECT().SIPAuthUpdateAll(ctx, tt.expectSIPAuth).Return(nil)

		mockNotify.EXPECT().PublishEvent(gomock.Any(), extension.EventTypeExtensionUpdated, tt.updatedExt)
		_, err := h.Update(ctx, tt.id, tt.extensionName, tt.detail, tt.password)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func Test_ExtensionDelete(t *testing.T) {

	type test struct {
		name              string
		responseExtension *extension.Extension

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

		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(ctx, tt.responseExtension.ID).Return(tt.responseExtension, nil)
		mockDBBin.EXPECT().ExtensionDelete(ctx, tt.responseExtension.ID).Return(nil)
		mockDBAst.EXPECT().AstEndpointDelete(ctx, tt.responseExtension.EndpointID).Return(nil)
		mockDBAst.EXPECT().AstAuthDelete(ctx, tt.responseExtension.AuthID).Return(nil)
		mockDBAst.EXPECT().AstAORDelete(ctx, tt.responseExtension.AORID).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(ctx, tt.responseExtension.ID).Return(tt.responseExtension, nil)
		mockDBBin.EXPECT().SIPAuthDelete(ctx, tt.responseExtension.ID).Return(nil)
		mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionDeleted, tt.responseExtension)

		res, err := h.Delete(ctx, tt.responseExtension.ID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.expectRes, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
		}
	}
}

func Test_ExtensionGet(t *testing.T) {

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
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		h := &extensionHandler{
			dbAst: mockDBAst,
			dbBin: mockDBBin,
		}

		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(tt.ext, nil)
		res, err := h.Get(ctx, tt.ext.ID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if reflect.DeepEqual(tt.ext, res) == false {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.ext, res)
		}

	}
}

// func Test_ExtensionGetsByDomainID(t *testing.T) {

// 	type test struct {
// 		name     string
// 		domainID uuid.UUID
// 		token    string
// 		exts     []*extension.Extension
// 	}

// 	tests := []test{
// 		{
// 			"test normal",
// 			uuid.FromStringOrNil("bd57214a-6f4b-11eb-aad8-579de27e6b7f"),
// 			"2021-02-15 17:31:59.519672",
// 			[]*extension.Extension{
// 				{
// 					ID:         uuid.FromStringOrNil("c9c736a4-6f4b-11eb-899a-575b7ce222e6"),
// 					CustomerID: uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264"),
// 					Name:       "test name",
// 					Detail:     "test detail",
// 					AuthID:     "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
// 					EndpointID: "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
// 					AORID:      "d1f16192-6f4b-11eb-83aa-27a0be9dffd1@test.sip.voipbin.net",
// 					Extension:  "d1f16192-6f4b-11eb-83aa-27a0be9dffd1",
// 					Password:   "test password",
// 					TMCreate:   "2021-02-14 17:31:59.519672",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		mc := gomock.NewController(t)
// 		defer mc.Finish()

// 		mockDBAst := dbhandler.NewMockDBHandler(mc)
// 		mockDBBin := dbhandler.NewMockDBHandler(mc)
// 		h := &extensionHandler{
// 			dbAst: mockDBAst,
// 			dbBin: mockDBBin,
// 		}

// 		ctx := context.Background()

// 		mockDBBin.EXPECT().ExtensionGetsByDomainID(gomock.Any(), tt.domainID, tt.token, uint64(10)).Return(tt.exts, nil)
// 		res, err := h.GetsByDomainID(ctx, tt.domainID, tt.token, uint64(10))
// 		if err != nil {
// 			t.Errorf("Wrong match. expect: ok, got: %v", err)
// 		}

// 		if reflect.DeepEqual(tt.exts, res) == false {
// 			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.exts, res)
// 		}

// 	}
// }

func Test_ExtensionGetsByCustomerID(t *testing.T) {

	type test struct {
		name       string
		customerID uuid.UUID
		token      string
		exts       []*extension.Extension
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f415f9b8-4fef-11ee-a071-0f13ee41a35d"),
			"2021-02-15 17:31:59.519672",
			[]*extension.Extension{
				{
					ID: uuid.FromStringOrNil("f449ace0-4fef-11ee-a0f8-c71aa984ea34"),
				},
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		h := &extensionHandler{
			dbAst: mockDBAst,
			dbBin: mockDBBin,
		}

		ctx := context.Background()

		mockDBBin.EXPECT().ExtensionGetsByCustomerID(gomock.Any(), tt.customerID, tt.token, uint64(10)).Return(tt.exts, nil)
		res, err := h.GetsByCustomerID(ctx, tt.customerID, tt.token, uint64(10))
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if reflect.DeepEqual(tt.exts, res) == false {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.exts, res)
		}

	}
}
