package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
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
		name    string
		user    *user.User
		flow    *flow.Flow
		reqFlow *fmflow.Flow

		response  *fmflow.Flow
		expectRes *flow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			&flow.Flow{
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
				Persist: true,
			},
			&fmflow.Flow{
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []fmaction.Action{},
				Persist: true,
			},
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []fmaction.Action{},
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
		{
			"webhook",
			&user.User{
				ID: 1,
			},
			&flow.Flow{
				UserID:     1,
				Name:       "test",
				Detail:     "test detail",
				Actions:    []action.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
			},
			&fmflow.Flow{
				UserID:     1,
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("5d70b47c-82f5-11eb-9d41-53331f170b23"),
				UserID:     1,
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("5d70b47c-82f5-11eb-9d41-53331f170b23"),
				UserID:     1,
				Name:       "test",
				Detail:     "test detail",
				Actions:    []action.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.user.ID, tt.reqFlow.Name, tt.reqFlow.Detail, tt.reqFlow.WebhookURI, tt.reqFlow.Actions, tt.reqFlow.Persist).Return(tt.response, nil)
			res, err := h.FlowCreate(tt.user, tt.flow.Name, tt.flow.Detail, tt.flow.WebhookURI, tt.flow.Actions, tt.flow.Persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		flow *flow.Flow

		requestFlow *fmflow.Flow
		response    *fmflow.Flow
		expectRes   *flow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				Name:    "update name",
				Detail:  "update detail",
				Actions: []action.Action{},
			},
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				Name:    "update name",
				Detail:  "update detail",
				Actions: []fmaction.Action{},
			},
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				UserID:  1,
				Name:    "update name",
				Detail:  "update detail",
				Actions: []fmaction.Action{},
				Persist: true,
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				UserID:  1,
				Name:    "update name",
				Detail:  "update detail",
				Actions: []action.Action{},
				Persist: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMV1FlowGet(gomock.Any(), tt.flow.ID).Return(&fmflow.Flow{UserID: 1}, nil)
			mockReq.EXPECT().FMV1FlowUpdate(gomock.Any(), tt.requestFlow).Return(tt.response, nil)
			res, err := h.FlowUpdate(tt.user, tt.flow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowDelete(t *testing.T) {
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

		response *fmflow.Flow
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),

			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []fmaction.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMV1FlowGet(gomock.Any(), tt.flowID).Return(tt.response, nil)
			mockReq.EXPECT().FMV1FlowDelete(gomock.Any(), tt.flowID).Return(nil)

			if err := h.FlowDelete(tt.user, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
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
				Actions: []fmaction.Action{},
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
				UserID:  1,
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
			},
		},
		{
			"action answer",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),

			&fmflow.Flow{
				ID:     uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
				UserID: 1,
				Name:   "test",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("61f86f60-66af-11eb-917f-838fd6836e1f"),
						Type: fmaction.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
				UserID: 1,
				Name:   "test",
				Detail: "test detail",
				Actions: []action.Action{
					{
						Type: "answer",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMV1FlowGet(gomock.Any(), tt.flowID).Return(tt.response, nil)

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

func TestFlowGets(t *testing.T) {
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
				{
					ID:      uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					UserID:  1,
					Name:    "test1",
					Detail:  "test detail1",
					Actions: []fmaction.Action{},
				},
				{
					ID:      uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					UserID:  1,
					Name:    "test2",
					Detail:  "test detail2",
					Actions: []fmaction.Action{},
				},
			},
			[]*flow.Flow{
				{
					ID:      uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					UserID:  1,
					Name:    "test1",
					Detail:  "test detail1",
					Actions: []action.Action{},
				},
				{
					ID:      uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					UserID:  1,
					Name:    "test2",
					Detail:  "test detail2",
					Actions: []action.Action{},
				},
			},
		},
		{
			"1 action",
			&user.User{
				ID: 1,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]fmflow.Flow{
				{
					ID:     uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
					UserID: 1,
					Name:   "test1",
					Detail: "test detail1",
					Actions: []fmaction.Action{
						{
							ID:   uuid.FromStringOrNil("775f5cde-66ae-11eb-9626-0f488d332e1e"),
							Type: fmaction.TypeAnswer,
						},
					},
				},
			},
			[]*flow.Flow{
				{
					ID:     uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
					UserID: 1,
					Name:   "test1",
					Detail: "test detail1",
					Actions: []action.Action{
						{
							Type: "answer",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().FMV1FlowGets(gomock.Any(), tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.FlowGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
