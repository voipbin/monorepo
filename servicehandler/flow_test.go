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
