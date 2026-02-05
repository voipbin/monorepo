package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_TrunkCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name  string
		trunk *trunk.Trunk

		responseCurTime *time.Time
		expectRes       *trunk.Trunk
	}

	tests := []test{
		{
			"have all",
			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5de945ba-519a-11ee-809d-0397adb97529"),
					CustomerID: uuid.FromStringOrNil("64bb7020-519a-11ee-b9ae-0f71e4c28f81"),
				},
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:      "test.trunk.voipbin.net",
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
					"1.2.3.5",
				},
			},

			curTime,
			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5de945ba-519a-11ee-809d-0397adb97529"),
					CustomerID: uuid.FromStringOrNil("64bb7020-519a-11ee-b9ae-0f71e4c28f81"),
				},
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:      "test.trunk.voipbin.net",
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
					"1.2.3.5",
				},
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
		{
			"empty",
			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("21ed74e4-cc80-11ee-b64b-b36a53c6cafc"),
				},
			},

			curTime,
			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("21ed74e4-cc80-11ee-b64b-b36a53c6cafc"),
				},
				AuthTypes:  []sipauth.AuthType{},
				AllowedIPs: []string{},
				TMCreate:   curTime,
				TMUpdate:   nil,
				TMDelete:   nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			if err := h.TrunkCreate(ctx, tt.trunk); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TrunkGet(ctx, tt.trunk.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			res, err := h.TrunkGet(ctx, tt.trunk.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().TrunkGetByDomainName(ctx, tt.trunk.DomainName).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			res, err = h.TrunkGetByDomainName(ctx, tt.trunk.DomainName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_TrunkList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		data []trunk.Trunk

		limit   uint64
		token   string
		filters map[trunk.Field]any

		responseCurTime *time.Time

		expectRes []*trunk.Trunk
	}

	tests := []test{
		{
			"normal",
			[]trunk.Trunk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1c4b4fd8-cdc1-11ee-914a-67975f17aab4"),
						CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					},
					DomainName: "test1",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1c829d94-cdc1-11ee-9ae0-0700acee5380"),
						CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					},
					DomainName: "test2",
				},
			},

			10,
			"",
			map[trunk.Field]any{
				trunk.FieldDeleted:    false,
				trunk.FieldDomainName: "test2",
			},

			curTime,

			[]*trunk.Trunk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1c829d94-cdc1-11ee-9ae0-0700acee5380"),
						CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					},
					DomainName: "test2",
					AuthTypes:  []sipauth.AuthType{},
					AllowedIPs: []string{},
					TMCreate:   curTime,
					TMUpdate:   nil,
					TMDelete:   nil,
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

			for _, d := range tt.data {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().TrunkSet(gomock.Any(), gomock.Any())
				if err := h.TrunkCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TrunkList(ctx, tt.limit, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TrunkUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name        string
		trunkCreate *trunk.Trunk

		id     uuid.UUID
		fields map[trunk.Field]any

		responseCurTime *time.Time

		expectDomain *trunk.Trunk
	}

	tests := []test{
		{
			"test normal",
			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
					CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				},
				DomainName: "cc1fa35e-519e-11ee-adcf-1f15aa304cb4",
			},

			uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
			map[trunk.Field]any{
				trunk.FieldName:       "update name",
				trunk.FieldDetail:     "update detail",
				trunk.FieldAuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
				trunk.FieldUsername:   "test_username",
				trunk.FieldPassword:   "test_password",
				trunk.FieldAllowedIPs: []string{"1.2.3.4", "5.6.7.8"},
			},

			curTime,

			&trunk.Trunk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
					CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				},
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "cc1fa35e-519e-11ee-adcf-1f15aa304cb4",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
				Username:   "test_username",
				Password:   "test_password",
				AllowedIPs: []string{
					"1.2.3.4",
					"5.6.7.8",
				},

				TMCreate: curTime,
				TMUpdate: curTime,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			if err := h.TrunkCreate(ctx, tt.trunkCreate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			if err := h.TrunkUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TrunkGet(ctx, tt.trunkCreate.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			res, err := h.TrunkGet(ctx, tt.trunkCreate.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectDomain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomain, res)
			}
		})
	}
}
