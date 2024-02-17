package trunkhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		trunkName  string
		detail     string
		domainName string
		authTypes  []sipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		responseUUID  uuid.UUID
		responseTrunk *trunk.Trunk

		expectTrunk *trunk.Trunk
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("202b2592-8967-11ec-aeab-3336a440f2c1"),
			"test name",
			"test detail",
			"test-domain",
			[]sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
			"testusername",
			"testpassword",
			[]string{
				"1.2.3.4",
			},

			uuid.FromStringOrNil("1e9d3fb8-5228-11ee-a4d1-f34adf6b433e"),
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("1e9d3fb8-5228-11ee-a4d1-f34adf6b433e"),
			},

			&trunk.Trunk{
				ID:         uuid.FromStringOrNil("1e9d3fb8-5228-11ee-a4d1-f34adf6b433e"),
				CustomerID: uuid.FromStringOrNil("202b2592-8967-11ec-aeab-3336a440f2c1"),
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test-domain",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
				Realm:      "test-domain.trunk.voipbin.net",
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &trunkHandler{
				utilHandler:   mockUtil,
				db:            mockDBBin,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDBBin.EXPECT().TrunkGetByDomainName(ctx, tt.expectTrunk.DomainName).Return(nil, fmt.Errorf(""))
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDBBin.EXPECT().TrunkCreate(ctx, tt.expectTrunk)
			mockDBBin.EXPECT().TrunkGet(ctx, tt.expectTrunk.ID).Return(tt.responseTrunk, nil)
			mockNotify.EXPECT().PublishEvent(ctx, trunk.EventTypeTrunkCreated, tt.responseTrunk)

			res, err := h.Create(ctx, tt.customerID, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseTrunk, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTrunk, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseTrunk *trunk.Trunk
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507"),

			&trunk.Trunk{
				CustomerID: uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507"),
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &trunkHandler{
			db:            mockDBBin,
			notifyHandler: mockNotify,
		}
		ctx := context.Background()

		mockDBBin.EXPECT().TrunkGet(ctx, tt.id).Return(tt.responseTrunk, nil)
		res, err := h.Get(ctx, tt.id)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.responseTrunk, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTrunk, res)
		}
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseGets []*trunk.Trunk
		expectRes    []*trunk.Trunk
	}{
		{
			"normal",

			uuid.FromStringOrNil("aa2be0f0-5234-11ee-960c-43d098822966"),
			10,
			"2020-05-03%2021:35:02.809",

			[]*trunk.Trunk{
				{
					ID: uuid.FromStringOrNil("ab7dcb80-5234-11ee-a234-f7fd070d72e4"),
				},
			},
			[]*trunk.Trunk{
				{
					ID: uuid.FromStringOrNil("ab7dcb80-5234-11ee-a234-f7fd070d72e4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &trunkHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TrunkGetsByCustomerID(ctx, tt.customerID, tt.token, tt.size).Return(tt.responseGets, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.token, tt.size)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_GetByDomainName(t *testing.T) {

	type test struct {
		name string

		domainName string

		responseTrunk *trunk.Trunk
	}

	tests := []test{
		{
			"normal",

			"test",

			&trunk.Trunk{
				CustomerID: uuid.FromStringOrNil("34dce34a-5229-11ee-ac53-470b6e1ee43b"),
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &trunkHandler{
			db:            mockDBBin,
			notifyHandler: mockNotify,
		}
		ctx := context.Background()

		mockDBBin.EXPECT().TrunkGetByDomainName(ctx, tt.domainName).Return(tt.responseTrunk, nil)
		res, err := h.GetByDomainName(ctx, tt.domainName)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.responseTrunk, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTrunk, res)
		}
	}
}

func Test_Update(t *testing.T) {

	type test struct {
		name string

		id         uuid.UUID
		trunkName  string
		detail     string
		authTypes  []sipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		responseTrunk *trunk.Trunk
	}

	tests := []test{
		{
			"test normal",

			uuid.FromStringOrNil("80a7dd20-5229-11ee-bf8c-a3fb6b428056"),
			"update name",
			"update detail",
			[]sipauth.AuthType{sipauth.AuthTypeBasic},
			"updateusername",
			"updatepassword",
			[]string{
				"1.2.3.4",
			},

			&trunk.Trunk{
				ID: uuid.FromStringOrNil("80a7dd20-5229-11ee-bf8c-a3fb6b428056"),
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &trunkHandler{
			db:            mockDBBin,
			notifyHandler: mockNotify,
		}

		ctx := context.Background()

		mockDBBin.EXPECT().TrunkUpdateBasicInfo(gomock.Any(), tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(nil)
		mockDBBin.EXPECT().TrunkGet(gomock.Any(), tt.responseTrunk.ID).Return(tt.responseTrunk, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), trunk.EventTypeTrunkUpdated, tt.responseTrunk)
		_, err := h.Update(ctx, tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func Test_Delete(t *testing.T) {

	type test struct {
		name string

		trunkID uuid.UUID

		responseTrunk *trunk.Trunk
		expectRes     *trunk.Trunk
	}

	tests := []test{
		{
			"test normal",

			uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),

			&trunk.Trunk{
				ID: uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),
			},
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("8a603afc-6f31-11eb-8ca1-0777f2a6f66e"),
			},
		},
	}

	for _, tt := range tests {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &trunkHandler{
			db:            mockDBBin,
			notifyHandler: mockNotify,
		}
		ctx := context.Background()

		mockDBBin.EXPECT().TrunkDelete(ctx, tt.trunkID).Return(nil)
		mockDBBin.EXPECT().TrunkGet(ctx, tt.trunkID).Return(tt.responseTrunk, nil)
		mockNotify.EXPECT().PublishEvent(ctx, trunk.EventTypeTrunkDeleted, tt.responseTrunk)
		res, err := h.Delete(ctx, tt.trunkID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if !reflect.DeepEqual(tt.expectRes, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
		}
	}
}
