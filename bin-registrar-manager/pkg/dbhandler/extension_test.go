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

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func Test_ExtensionCreate(t *testing.T) {

	type test struct {
		name string
		ext  *extension.Extension

		responseCurTime string
		expectRes       *extension.Extension
	}

	tests := []test{
		{
			"normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				CustomerID: uuid.FromStringOrNil("83db3318-7fec-11ec-a205-736ad70c9180"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension:  "608cbfae-6ebc-11eb-a74b-671d17dda173",
				DomainName: "83db3318-7fec-11ec-a205-736ad70c9180",

				Realm:    "83db3318-7fec-11ec-a205-736ad70c9180.registrar.voipbin.net",
				Username: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password: "7818abce-6ebc-11eb-b4fe-e748480c228a",
			},

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				CustomerID: uuid.FromStringOrNil("83db3318-7fec-11ec-a205-736ad70c9180"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension:  "608cbfae-6ebc-11eb-a74b-671d17dda173",
				DomainName: "83db3318-7fec-11ec-a205-736ad70c9180",

				Realm:    "83db3318-7fec-11ec-a205-736ad70c9180.registrar.voipbin.net",
				Username: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password: "7818abce-6ebc-11eb-b4fe-e748480c228a",

				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: DefaultTimeStamp,
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
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			res, err := h.ExtensionGet(ctx, tt.ext.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			mockCache.EXPECT().ExtensionGetByEndpointID(ctx, tt.ext.EndpointID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			resGetByExtension, err := h.ExtensionGetByEndpointID(ctx, tt.ext.EndpointID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, resGetByExtension) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, resGetByExtension)
			}
		})
	}
}

func Test_ExtensionGetByExtension(t *testing.T) {

	type test struct {
		name string
		ext  *extension.Extension

		customerID uuid.UUID
		exten      string

		responseCurTime string
		expectRes       *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("569711e0-564d-11ee-97bc-e73899c004b9"),
				CustomerID: uuid.FromStringOrNil("56c83c70-564d-11ee-b707-d3539191ce8c"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",
				AORID:      "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",
				AuthID:     "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",

				Extension: "56f79dda-564d-11ee-9b02-2ff26b372f36",

				DomainName: "8cadaf5c-7fec-11ec-b004-53f79c2b8387",
				Username:   "56f79dda-564d-11ee-9b02-2ff26b372f36",
				Password:   "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",
			},

			uuid.FromStringOrNil("56c83c70-564d-11ee-b707-d3539191ce8c"),
			"56f79dda-564d-11ee-9b02-2ff26b372f36",

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("569711e0-564d-11ee-97bc-e73899c004b9"),
				CustomerID: uuid.FromStringOrNil("56c83c70-564d-11ee-b707-d3539191ce8c"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",
				AORID:      "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",
				AuthID:     "56f79dda-564d-11ee-9b02-2ff26b372f36@test.sip.voipbin.net",

				Extension: "56f79dda-564d-11ee-9b02-2ff26b372f36",

				DomainName: "8cadaf5c-7fec-11ec-b004-53f79c2b8387",
				Username:   "56f79dda-564d-11ee-9b02-2ff26b372f36",
				Password:   "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",

				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: DefaultTimeStamp,
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
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGetByCustomerIDANDExtension(ctx, tt.customerID, tt.exten).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			res, err := h.ExtensionGetByExtension(ctx, tt.customerID, tt.exten)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDelete(t *testing.T) {

	type test struct {
		name string
		ext  *extension.Extension

		responseCurTime string
		expectRes       *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("def11a70-6ebc-11eb-ae2b-d31ef2c6d22d"),
				CustomerID: uuid.FromStringOrNil("8cadaf5c-7fec-11ec-b004-53f79c2b8387"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AORID:      "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AuthID:     "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",

				Extension: "e56c33b2-6ebc-11eb-bada-4f15e459e32f",

				DomainName: "8cadaf5c-7fec-11ec-b004-53f79c2b8387",
				Username:   "e56c33b2-6ebc-11eb-bada-4f15e459e32f",
				Password:   "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",
			},

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("def11a70-6ebc-11eb-ae2b-d31ef2c6d22d"),
				CustomerID: uuid.FromStringOrNil("8cadaf5c-7fec-11ec-b004-53f79c2b8387"),

				Name:   "test name",
				Detail: "test detail",

				EndpointID: "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AORID:      "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AuthID:     "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",

				Extension: "e56c33b2-6ebc-11eb-bada-4f15e459e32f",

				DomainName: "8cadaf5c-7fec-11ec-b004-53f79c2b8387",
				Username:   "e56c33b2-6ebc-11eb-bada-4f15e459e32f",
				Password:   "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",

				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: "2021-02-26 18:26:49.000",
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
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			if err := h.ExtensionDelete(ctx, tt.ext.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(ctx, tt.ext.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			res, err := h.ExtensionGet(ctx, tt.ext.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionUpdate(t *testing.T) {

	type test struct {
		name      string
		extension *extension.Extension

		id            uuid.UUID
		extensionName string
		detail        string
		password      string

		responseCurTime string
		expectRes       *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),

				Name:   "test",
				Detail: "detail",

				Extension: "test",

				DomainName: "935e91e0-7fec-11ec-a93e-a3c37f19587c",
				Username:   "test",
				Password:   "password",
			},

			uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
			"update name",
			"update detail",
			"update password",

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),

				Name:   "update name",
				Detail: "update detail",

				Extension: "test",

				DomainName: "935e91e0-7fec-11ec-a93e-a3c37f19587c",
				Username:   "test",
				Password:   "update password",

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
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.extension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionUpdate(ctx, tt.id, tt.extensionName, tt.detail, tt.password); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(gomock.Any(), tt.extension.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			res, err := h.ExtensionGet(context.Background(), tt.extension.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionGets(t *testing.T) {

	type test struct {
		name       string
		extensions []extension.Extension

		limit   uint64
		filters map[string]string

		responseCurTime string
		expectRes       []*extension.Extension
	}

	tests := []test{
		{
			"normal",
			[]extension.Extension{
				{
					ID:         uuid.FromStringOrNil("f3672bd4-cdc8-11ee-843b-1f9dbb177486"),
					CustomerID: uuid.FromStringOrNil("f3bcb6d0-cdc8-11ee-82a8-430793535c91"),
					Name:       "test1",
				},
				{
					ID:         uuid.FromStringOrNil("f39e3124-cdc8-11ee-8e67-fb88ad978691"),
					CustomerID: uuid.FromStringOrNil("f3bcb6d0-cdc8-11ee-82a8-430793535c91"),
					Name:       "test2",
				},
			},

			10,
			map[string]string{
				"deleted":     "false",
				"customer_id": "f3bcb6d0-cdc8-11ee-82a8-430793535c91",
			},

			"2021-02-26 18:26:49.000",
			[]*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("f3672bd4-cdc8-11ee-843b-1f9dbb177486"),
					CustomerID: uuid.FromStringOrNil("f3bcb6d0-cdc8-11ee-82a8-430793535c91"),
					Name:       "test1",
					TMCreate:   "2021-02-26 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("f39e3124-cdc8-11ee-8e67-fb88ad978691"),
					CustomerID: uuid.FromStringOrNil("f3bcb6d0-cdc8-11ee-82a8-430793535c91"),
					Name:       "test2",
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

			for _, d := range tt.extensions {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
				if err := h.ExtensionCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			exts, err := h.ExtensionGets(ctx, tt.limit, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(exts, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], exts[0])
			}
		})
	}
}
