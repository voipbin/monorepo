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
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				CustomerID: uuid.FromStringOrNil("83db3318-7fec-11ec-a205-736ad70c9180"),

				DomainID: uuid.FromStringOrNil("4dc1e430-6ebc-11eb-b355-b35fc1cfc5a1"),

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password:  "7818abce-6ebc-11eb-b4fe-e748480c228a",
			},

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				CustomerID: uuid.FromStringOrNil("83db3318-7fec-11ec-a205-736ad70c9180"),

				DomainID: uuid.FromStringOrNil("4dc1e430-6ebc-11eb-b355-b35fc1cfc5a1"),

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password:  "7818abce-6ebc-11eb-b4fe-e748480c228a",

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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

			mockCache.EXPECT().ExtensionGetByExtension(ctx, tt.ext.Extension).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			resGetByExtension, err := h.ExtensionGetByExtension(ctx, tt.ext.Extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, resGetByExtension) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, resGetByExtension)
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

				DomainID: uuid.FromStringOrNil("e22acb78-6ebc-11eb-848e-bfb26fcad363"),

				EndpointID: "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AORID:      "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AuthID:     "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",

				Extension: "e56c33b2-6ebc-11eb-bada-4f15e459e32f",
				Password:  "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",
			},

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("def11a70-6ebc-11eb-ae2b-d31ef2c6d22d"),
				CustomerID: uuid.FromStringOrNil("8cadaf5c-7fec-11ec-b004-53f79c2b8387"),

				DomainID: uuid.FromStringOrNil("e22acb78-6ebc-11eb-848e-bfb26fcad363"),

				EndpointID: "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AORID:      "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AuthID:     "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",

				Extension: "e56c33b2-6ebc-11eb-bada-4f15e459e32f",
				Password:  "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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

func Test_ExtensionGetsByDomainID(t *testing.T) {

	type test struct {
		name       string
		domainID   uuid.UUID
		limit      uint64
		extensions []extension.Extension

		responseCurTime string
		expectRes       []*extension.Extension
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
			10,
			[]extension.Extension{
				{
					ID:         uuid.FromStringOrNil("1d2cb402-6f49-11eb-a22c-5f2f23cba3a2"),
					CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),
					Name:       "test1",
					DomainID:   uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
				{
					ID:         uuid.FromStringOrNil("1d792bb6-6f49-11eb-be2e-0ff2f1c87d93"),
					CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),
					Name:       "test2",
					DomainID:   uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
			},

			"2021-02-26 18:26:49.000",
			[]*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("1d792bb6-6f49-11eb-be2e-0ff2f1c87d93"),
					CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),
					Name:       "test2",
					DomainID:   uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
					TMCreate:   "2021-02-26 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("1d2cb402-6f49-11eb-a22c-5f2f23cba3a2"),
					CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),
					Name:       "test1",
					DomainID:   uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
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
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
				if err := h.ExtensionCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			exts, err := h.ExtensionGetsByDomainID(ctx, tt.domainID, utilhandler.GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(exts, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, exts)
			}
		})
	}
}

func Test_ExtensionUpdate(t *testing.T) {

	type test struct {
		name            string
		extension       *extension.Extension
		updateExtension *extension.Extension

		responseCurTime string
		expectRes       *extension.Extension
	}

	tests := []test{
		{
			"test normal",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),

				Name:     "test",
				Detail:   "detail",
				DomainID: uuid.FromStringOrNil("b2277afa-711b-11eb-a695-f71fad093e64"),

				Extension: "test",
				Password:  "password",
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),
				Name:       "update name",
				Detail:     "update detail",

				Password: "update password",
			},

			"2021-02-26 18:26:49.000",
			&extension.Extension{
				ID:         uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				CustomerID: uuid.FromStringOrNil("935e91e0-7fec-11ec-a93e-a3c37f19587c"),

				Name:   "update name",
				Detail: "update detail",

				DomainID: uuid.FromStringOrNil("b2277afa-711b-11eb-a695-f71fad093e64"),

				Extension: "test",
				Password:  "update password",

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.extension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionUpdate(ctx, tt.updateExtension); err != nil {
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
