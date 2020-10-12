package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
)

func TestFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name       string
		user       *user.User
		flowID     uuid.UUID
		flowName   string
		flowDetail string
		actions    []action.Action
		persist    bool

		response  *fmflow.Flow
		expectRes *flow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
			"test",
			"test detail",
			[]action.Action{},
			true,
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
				Persist: true,
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
				Persist: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ctx := context.Background()

			mockReq.EXPECT().FMFlowCreate(tt.user.ID, tt.flowID, tt.flowName, tt.flowDetail, tt.actions, tt.persist).Return(tt.response, nil)

			res, err := h.FlowCreate(tt.user, tt.flowID, tt.flowName, tt.flowDetail, tt.actions, tt.persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name   string
		user   *user.User
		flowID uuid.UUID

		response  *fmflow.Flow
		expectRes *flow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),

			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMFlowGet(tt.flowID).Return(tt.response, nil)

			res, err := h.FlowGet(tt.user, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowGetsByUserID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *user.User
		pageToken string
		pageSize  uint64

		response  []fmflow.Flow
		expectRes []*flow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]fmflow.Flow{
				fmflow.Flow{
					ID:      uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					UserID:  1,
					Name:    "test1",
					Detail:  "test detail1",
					Actions: []action.Action{},
				},
				fmflow.Flow{
					ID:      uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					UserID:  1,
					Name:    "test2",
					Detail:  "test detail2",
					Actions: []action.Action{},
				},
			},
			[]*flow.Flow{
				&flow.Flow{
					ID:      uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					UserID:  1,
					Name:    "test1",
					Detail:  "test detail1",
					Actions: []action.Action{},
				},

				&flow.Flow{
					ID:      uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					UserID:  1,
					Name:    "test2",
					Detail:  "test detail2",
					Actions: []action.Action{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMFlowGets(tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.FlowGetsByUserID(tt.user, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
