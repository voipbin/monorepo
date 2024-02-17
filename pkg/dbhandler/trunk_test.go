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

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func Test_TrunkCreate(t *testing.T) {

	type test struct {
		name  string
		trunk *trunk.Trunk

		responseCurTime string
		expectRes       *trunk.Trunk
	}

	tests := []test{
		{
			"have all",
			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("5de945ba-519a-11ee-809d-0397adb97529"),
				CustomerID: uuid.FromStringOrNil("64bb7020-519a-11ee-b9ae-0f71e4c28f81"),
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

			"2021-02-26 18:26:49.000",
			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("5de945ba-519a-11ee-809d-0397adb97529"),
				CustomerID: uuid.FromStringOrNil("64bb7020-519a-11ee-b9ae-0f71e4c28f81"),
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
				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("21ed74e4-cc80-11ee-b64b-b36a53c6cafc"),
			},

			"2021-02-26 18:26:49.000",
			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("21ed74e4-cc80-11ee-b64b-b36a53c6cafc"),
				AuthTypes:  []sipauth.AuthType{},
				AllowedIPs: []string{},
				TMCreate:   "2021-02-26 18:26:49.000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
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
		})
	}
}

func Test_TrunkGets(t *testing.T) {
	type test struct {
		name string
		data []trunk.Trunk

		limit   uint64
		token   string
		filters map[string]string

		responseCurTime string

		expectRes []*trunk.Trunk
	}

	tests := []test{
		{
			"normal",
			[]trunk.Trunk{
				{
					ID:         uuid.FromStringOrNil("1c4b4fd8-cdc1-11ee-914a-67975f17aab4"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					DomainName: "test1",
				},
				{
					ID:         uuid.FromStringOrNil("1c829d94-cdc1-11ee-9ae0-0700acee5380"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					DomainName: "test2",
				},
			},

			10,
			"",
			map[string]string{
				"deleted":     "false",
				"domain_name": "test2",
			},

			"2021-02-26 18:26:49.000",

			[]*trunk.Trunk{
				{
					ID:         uuid.FromStringOrNil("1c829d94-cdc1-11ee-9ae0-0700acee5380"),
					CustomerID: uuid.FromStringOrNil("423ec352-7fec-11ec-a715-a3caa41c981c"),
					DomainName: "test2",
					AuthTypes:  []sipauth.AuthType{},
					AllowedIPs: []string{},
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

			for _, d := range tt.data {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TrunkSet(gomock.Any(), gomock.Any())
				if err := h.TrunkCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TrunkGets(ctx, tt.limit, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TrunkUpdateBasicInfo(t *testing.T) {

	type test struct {
		name  string
		trunk *trunk.Trunk

		id         uuid.UUID
		domainN    string
		detail     string
		authTypes  []sipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		responseCurTime string

		expectDomain *trunk.Trunk
	}

	tests := []test{
		{
			"test normal",
			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
				CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
				DomainName: "cc1fa35e-519e-11ee-adcf-1f15aa304cb4",
			},

			uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
			"update name",
			"update detail",
			[]sipauth.AuthType{sipauth.AuthTypeBasic},
			"test_username",
			"test_password",
			[]string{
				"1.2.3.4",
				"5.6.7.8",
			},

			"2021-02-26 18:26:49.000",

			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234"),
				CustomerID: uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204"),
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
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			if err := h.TrunkCreate(ctx, tt.trunk); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			if err := h.TrunkUpdateBasicInfo(ctx, tt.id, tt.domainN, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TrunkGet(ctx, tt.trunk.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TrunkSet(ctx, gomock.Any())
			res, err := h.TrunkGet(ctx, tt.trunk.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectDomain, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDomain, res)
			}
		})
	}
}
