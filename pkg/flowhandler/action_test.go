package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func TestActionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name string
		flow *flow.Flow
	}

	tests := []test{
		{
			"test normal",
			&flow.Flow{
				ID: uuid.Must(uuid.NewV4()),
				Actions: []action.Action{
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: action.TypeEcho,
					},
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: action.TypeEcho,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			action, err := h.ActionGet(context.Background(), tt.flow.ID, tt.flow.Actions[0].ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*action, tt.flow.Actions[0]) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.flow.Actions[0], *action)
			}
		})
	}
}

// func TestActionPatchGet(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockDB := dbhandler.NewMockDBHandler(mc)

// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Fprintln(w, `[{"type":"hangup"}]`)
// 	}))
// 	defer ts.Close()
// 	targetURL := ts.URL

// 	h := &flowHandler{
// 		db: mockDB,
// 	}

// 	tests := []struct {
// 		name   string
// 		act    *action.Action
// 		callID uuid.UUID
// 	}{
// 		{
// 			"normal",
// 			&action.Action{
// 				ID:     uuid.FromStringOrNil("6e2a0cee-fba2-11ea-a469-a350f2dad844"),
// 				Option: []byte(fmt.Sprintf(`{"event_url": "%s"}`, targetURL)),
// 			},
// 			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			_, err := h.actionPatchGet(tt.act, tt.callID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func TestGetActionsFromFlow(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockDB := dbhandler.NewMockDBHandler(mc)

// 	h := &flowHandler{
// 		db: mockDB,
// 	}

// 	tests := []struct {
// 		name   string
// 		flowID uuid.UUID
// 		flow   *flow.Flow
// 		callID uuid.UUID
// 	}{
// 		{
// 			"normal",
// 			uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
// 			&flow.Flow{
// 				ID: uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
// 			},
// 			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flowID).Return(&flow.Flow{
// 				CustomerID: tt.flow.CustomerID,
// 			}, nil)

// 			_, err := h.getActionsFromFlow(tt.flowID, tt.flow.CustomerID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
