package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	dbhandler "gitlab.com/voipbin/bin-manager/flow-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
)

func TestFlowCreate(t *testing.T) {
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
			&flow.Flow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any(), uuid.Nil).Return(&flow.Flow{}, nil)

			h.FlowCreate(context.Background(), &flow.Flow{})

		})
	}
}

func TestFlowGet(t *testing.T) {
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
			&flow.Flow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any(), gomock.Any()).Return(&flow.Flow{}, nil)

			h.FlowGet(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
		})
	}
}

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
				ID:       uuid.Must(uuid.NewV4()),
				Revision: uuid.Must(uuid.NewV4()),
				Actions: []flow.Action{
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: flow.ActionTypeEcho,
					},
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: flow.ActionTypeEcho,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID, tt.flow.Revision).Return(tt.flow, nil)
			action, err := h.ActionGet(context.Background(), tt.flow.ID, tt.flow.Revision, tt.flow.Actions[0].ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*action, tt.flow.Actions[0]) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.flow.Actions[0], *action)
			}
		})
	}
}
