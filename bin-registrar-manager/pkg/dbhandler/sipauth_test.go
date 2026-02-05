package dbhandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_SIPAuthCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name    string
		sipauth *sipauth.SIPAuth

		responseCurTime *time.Time
		expectRes       *sipauth.SIPAuth
	}

	tests := []test{
		{
			"normal",
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("35e51522-cda4-11ee-a4e3-835eaf8559f0"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs:    []string{"1.2.3.4", "1.2.3.5"},
			},

			curTime,
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("35e51522-cda4-11ee-a4e3-835eaf8559f0"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
					"1.2.3.5",
				},
				TMCreate: curTime,
				TMUpdate: nil,
			},
		},
		{
			"empty",
			&sipauth.SIPAuth{
				ID: uuid.FromStringOrNil("36174498-cda4-11ee-ad5c-93439335fc1a"),
			},

			curTime,
			&sipauth.SIPAuth{
				ID:         uuid.FromStringOrNil("36174498-cda4-11ee-ad5c-93439335fc1a"),
				AuthTypes:  []sipauth.AuthType{},
				AllowedIPs: []string{},
				TMCreate:   curTime,
				TMUpdate:   nil,
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
			if err := h.SIPAuthCreate(ctx, tt.sipauth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.SIPAuthGet(ctx, tt.sipauth.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SIPAuthUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name    string
		sipauth *sipauth.SIPAuth

		updateFields map[sipauth.Field]any

		responseCurTime *time.Time
		expectRes       *sipauth.SIPAuth
	}

	tests := []test{
		{
			"normal",
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("3859e9de-cda6-11ee-b6c1-678b4af08a31"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs:    []string{"1.2.3.4", "1.2.3.5"},
			},

			map[sipauth.Field]any{
				sipauth.FieldAuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
				sipauth.FieldRealm:      "update.trunk.voipbin.net",
				sipauth.FieldUsername:   "updateusername",
				sipauth.FieldPassword:   "updatepassword",
				sipauth.FieldAllowedIPs: []string{"1.2.3.6", "1.2.3.7"},
			},

			curTime,
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("3859e9de-cda6-11ee-b6c1-678b4af08a31"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
				Realm:         "update.trunk.voipbin.net",
				Username:      "updateusername",
				Password:      "updatepassword",
				AllowedIPs:    []string{"1.2.3.6", "1.2.3.7"},
				TMCreate:      curTime,
				TMUpdate:      curTime,
			},
		},
		{
			"empty update",
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("3829303c-cda6-11ee-9273-672228e0f5ba"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs:    []string{"1.2.3.4", "1.2.3.5"},
			},

			map[sipauth.Field]any{
				sipauth.FieldAuthTypes:  []sipauth.AuthType{},
				sipauth.FieldRealm:      "",
				sipauth.FieldUsername:   "",
				sipauth.FieldPassword:   "",
				sipauth.FieldAllowedIPs: []string{},
			},

			curTime,
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("3829303c-cda6-11ee-9273-672228e0f5ba"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{},
				AllowedIPs:    []string{},
				TMCreate:      curTime,
				TMUpdate:      curTime,
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
			if err := h.SIPAuthCreate(ctx, tt.sipauth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.SIPAuthUpdate(ctx, tt.sipauth.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.SIPAuthGet(ctx, tt.sipauth.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SIPAuthDelete(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name    string
		sipauth *sipauth.SIPAuth

		responseCurTime *time.Time
	}

	tests := []test{
		{
			"normal",
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("0384a71a-cda8-11ee-a445-9b562961478e"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs:    []string{"1.2.3.4", "1.2.3.5"},
			},

			curTime,
		},
		{
			"empty",
			&sipauth.SIPAuth{
				ID:            uuid.FromStringOrNil("03b73c3e-cda8-11ee-9955-fb090d924111"),
				ReferenceType: sipauth.ReferenceTypeTrunk,
				AuthTypes:     []sipauth.AuthType{sipauth.AuthTypeBasic},
				Realm:         "test.trunk.voipbin.net",
				Username:      "testusername",
				Password:      "testpassword",
				AllowedIPs:    []string{"1.2.3.4", "1.2.3.5"},
			},

			curTime,
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
			if err := h.SIPAuthCreate(ctx, tt.sipauth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.SIPAuthDelete(ctx, tt.sipauth.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			_, err := h.SIPAuthGet(ctx, tt.sipauth.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

		})
	}
}
