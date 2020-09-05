package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
)

func TestFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		flow       *flow.Flow
		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"have no actions",
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have 1 action echo without option",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have 1 action echo with option",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type:   action.TypeEcho,
						Option: []byte(`{"duration":180}`),
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type:   action.TypeEcho,
						Option: []byte(`{"duration":180}`),
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created flow. flow: %v", res)

			if reflect.DeepEqual(tt.expectFlow, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectFlow, res)
			}
		})
	}
}
