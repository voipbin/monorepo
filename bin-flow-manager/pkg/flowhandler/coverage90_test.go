package flowhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

var (
	errCov90        = errors.New("synthetic dependency failure")
	cov90FlowID     = uuid.FromStringOrNil("c9f1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")
	cov90ActionID   = uuid.FromStringOrNil("c9f1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
	cov90CustomerID = uuid.FromStringOrNil("c9f1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c03")
	cov90DirectID   = uuid.FromStringOrNil("c9f1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c04")
)

type cov90Mocks struct {
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	util   *utilhandler.MockUtilHandler
	notify *notifyhandler.MockNotifyHandler
}

func newCov90Handler(mc *gomock.Controller) (*flowHandler, *cov90Mocks) {
	m := &cov90Mocks{
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}

	return &flowHandler{util: m.util, db: m.db, reqHandler: m.req, notifyHandler: m.notify}, m
}

func Test_ActionGet_coverage90(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		responseFlow *flow.Flow
		responseErr  error

		expectErr error
		expectRes *action.Action
	}{
		{
			name:        "the flow can not be read",
			responseErr: errCov90,
			expectErr:   errCov90,
		},
		{
			name: "the action is not in the flow",
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{ID: cov90FlowID},
				Actions:  []action.Action{{ID: uuid.FromStringOrNil("c9f1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c99"), Type: action.TypeAnswer}},
			},
			expectErr: dbhandler.ErrNotFound,
		},
		{
			name: "the action is returned",
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{ID: cov90FlowID},
				Actions:  []action.Action{{ID: cov90ActionID, Type: action.TypeAnswer}},
			},
			expectRes: &action.Action{ID: cov90ActionID, Type: action.TypeAnswer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(tt.responseFlow, tt.responseErr)

			res, err := h.ActionGet(ctx, cov90FlowID, cov90ActionID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error.\nexpect: %v\ngot: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("no action is expected with the error. got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res == nil || res.ID != tt.expectRes.ID || res.Type != tt.expectRes.Type {
				t.Errorf("Wrong action.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CountByCustomerID_failure_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newCov90Handler(mc)
	m.db.EXPECT().FlowCountByCustomerID(gomock.Any(), cov90CustomerID).Return(0, errCov90)

	res, err := h.CountByCustomerID(context.Background(), cov90CustomerID)
	if !errors.Is(err, errCov90) {
		t.Errorf("The cause must be kept. got: %v", err)
	}
	if res != 0 {
		t.Errorf("zero is expected with the error. got: %d", res)
	}
}

func Test_DirectHashRegenerate_failure_coverage90(t *testing.T) {
	ctx := context.Background()

	existing := &flow.Flow{Identity: commonidentity.Identity{ID: cov90FlowID, CustomerID: cov90CustomerID}, DirectID: cov90DirectID}
	fresh := &flow.Flow{Identity: commonidentity.Identity{ID: cov90FlowID, CustomerID: cov90CustomerID}}
	direct := &dmdirect.Direct{ID: cov90DirectID, Hash: "synthetic-hash"}

	tests := []struct {
		name string

		setup func(m *cov90Mocks)

		expectMessage string
	}{
		{
			name: "the regenerate request fails",
			setup: func(m *cov90Mocks) {
				m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(existing, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(nil, errCov90)
			},
			expectMessage: "could not regenerate direct hash",
		},
		{
			name: "the create request fails",
			setup: func(m *cov90Mocks) {
				m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(fresh, nil)
				m.req.EXPECT().DirectV1DirectCreate(ctx, cov90CustomerID, "flow", cov90FlowID).Return(nil, errCov90)
			},
			expectMessage: "could not create direct hash",
		},
		{
			name: "the flow update fails",
			setup: func(m *cov90Mocks) {
				m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(existing, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(direct, nil)
				m.db.EXPECT().FlowUpdate(ctx, cov90FlowID, map[flow.Field]any{flow.FieldDirectID: cov90DirectID, flow.FieldDirectHash: "synthetic-hash"}).Return(errCov90)
			},
			expectMessage: "could not update flow",
		},
		{
			name: "the updated flow can not be read",
			setup: func(m *cov90Mocks) {
				gomock.InOrder(
					m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(existing, nil),
					m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(direct, nil),
					m.db.EXPECT().FlowUpdate(ctx, cov90FlowID, gomock.Any()).Return(nil),
					m.db.EXPECT().FlowGet(ctx, cov90FlowID).Return(nil, errCov90),
				)
			},
			expectMessage: errCov90.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			tt.setup(m)

			res, err := h.DirectHashRegenerate(ctx, cov90FlowID)
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) {
				t.Fatalf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectMessage, err)
			}
			if !errors.Is(err, errCov90) {
				t.Errorf("The cause must be kept. got: %v", err)
			}
			if res != nil {
				t.Errorf("no flow is expected with the error. got: %v", res)
			}
		})
	}
}

func Test_EventCustomerDeleted_failure_coverage90(t *testing.T) {
	ctx := context.Background()
	cu := &cmcustomer.Customer{}
	cu.ID = cov90CustomerID

	t.Run("the list failure is reported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		m.util.EXPECT().TimeGetCurTime().Return("2020-04-18 03:22:17.995000")
		m.db.EXPECT().FlowList(ctx, "2020-04-18 03:22:17.995000", uint64(1000), gomock.Any()).Return(nil, errCov90)

		err := h.EventCustomerDeleted(ctx, cu)
		if !errors.Is(err, errCov90) {
			t.Errorf("The cause must be kept. got: %v", err)
		}
	})

	t.Run("a delete failure does not stop the others", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		first := &flow.Flow{Identity: commonidentity.Identity{ID: cov90FlowID}}
		second := &flow.Flow{Identity: commonidentity.Identity{ID: cov90ActionID}}

		m.util.EXPECT().TimeGetCurTime().Return("2020-04-18 03:22:17.995000")
		m.db.EXPECT().FlowList(ctx, gomock.Any(), uint64(1000), gomock.Any()).Return([]*flow.Flow{first, second}, nil)
		// the first flow can not be read, so the delete of it fails, and the second one is read, deleted and read again.
		m.db.EXPECT().FlowGet(ctx, first.ID).Return(nil, errCov90)
		gomock.InOrder(
			m.db.EXPECT().FlowGet(ctx, second.ID).Return(second, nil),
			m.db.EXPECT().FlowDelete(ctx, second.ID).Return(nil),
			m.db.EXPECT().FlowGet(ctx, second.ID).Return(second, nil),
		)
		m.notify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowDeleted, second)

		if err := h.EventCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("A delete failure must not fail the event. got: %v", err)
		}
	})
}

func Test_NewFlowHandler_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)
	mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)

	res := NewFlowHandler(mockDB, mockReq, mockNotify, mockAction, mockActiveflow)
	h, ok := res.(*flowHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}

	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.actionHandler != mockAction || h.activeflowHandler != mockActiveflow {
		t.Errorf("The given dependencies must be kept")
	}
	if h.util == nil {
		t.Errorf("The util handler must be created")
	}
}
