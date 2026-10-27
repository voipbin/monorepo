package flowhandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

// newDependencyErrorTestHandler returns a handler whose dependencies are strict
// mocks. Any call that is not registered by a test fails the test.
func newDependencyErrorTestHandler(mc *gomock.Controller) (*flowHandler, *utilhandler.MockUtilHandler, *requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *actionhandler.MockActionHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)

	h := &flowHandler{
		util:          mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		actionHandler: mockAction,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
	}

	return h, mockUtil, mockReq, mockDB, mockAction
}

var (
	testDepFlowID     = uuid.FromStringOrNil("b4d1c0de-5a6e-4f7a-9c11-0a1b2c3d4e01")
	testDepCustomerID = uuid.FromStringOrNil("b4d1c0de-5a6e-4f7a-9c11-0a1b2c3d4e02")
	testDepDirectID   = uuid.FromStringOrNil("b4d1c0de-5a6e-4f7a-9c11-0a1b2c3d4e03")
)

func Test_Get_dependencyError(t *testing.T) {
	errDB := errors.New("synthetic database failure")

	tests := []struct {
		name string

		dbErr error

		expectNotFound bool
	}{
		{
			name:           "record not found is mapped to a typed not found error",
			dbErr:          dbhandler.ErrNotFound,
			expectNotFound: true,
		},
		{
			name:  "other database error is returned as is",
			dbErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, mockDB, _ := newDependencyErrorTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, testDepFlowID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.dbErr) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", tt.dbErr, err)
			}

			var typed *cerrors.VoipbinError
			isTyped := errors.As(err, &typed)
			if isTyped != tt.expectNotFound {
				t.Fatalf("Wrong match. expect typed: %v, got: %v, err: %v", tt.expectNotFound, isTyped, err)
			}
			if tt.expectNotFound && (typed.Status != cerrors.StatusNotFound || typed.Reason != "FLOW_NOT_FOUND") {
				t.Errorf("Wrong match. expect: not found FLOW_NOT_FOUND, got: %s %s", typed.Status, typed.Reason)
			}
		})
	}
}

func Test_Create_dependencyError(t *testing.T) {
	actions := []action.Action{{Type: action.TypeAnswer}}
	errSynthetic := errors.New("synthetic failure")

	tests := []struct {
		name string

		persist bool

		generateErr error
		cacheErr    error
		insertErr   error
		cleanupErr  error
		getErr      error

		expectErr error
	}{
		{
			name:        "generating the actions fails",
			persist:     true,
			generateErr: errSynthetic,
			expectErr:   errSynthetic,
		},
		{
			name:      "temporary flow can not be stored in the cache",
			persist:   false,
			cacheErr:  errSynthetic,
			expectErr: errSynthetic,
		},
		{
			name:       "insert fails and the orphaned direct hash can not be cleaned up",
			persist:    true,
			insertErr:  errSynthetic,
			cleanupErr: errors.New("synthetic cleanup failure"),
			expectErr:  errSynthetic,
		},
		{
			name:      "created flow can not be read back",
			persist:   true,
			getErr:    errSynthetic,
			expectErr: errSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockReq, mockDB, mockAction := newDependencyErrorTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().FlowCountByCustomerID(ctx, testDepCustomerID).Return(0, nil)

			if tt.generateErr != nil {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(nil, tt.generateErr)
			} else {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(actions, nil)
				mockUtil.EXPECT().UUIDCreate().Return(testDepFlowID)

				if tt.persist {
					mockReq.EXPECT().DirectV1DirectCreate(ctx, testDepCustomerID, "flow", testDepFlowID).Return(&dmdirect.Direct{
						Identity: commonidentity.Identity{ID: testDepDirectID},
						Hash:     "synthetic-hash",
					}, nil)
				}
				mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())

				switch {
				case tt.cacheErr != nil:
					mockDB.EXPECT().FlowSetToCache(ctx, gomock.Any()).Return(tt.cacheErr)
				case tt.insertErr != nil:
					mockDB.EXPECT().FlowCreate(ctx, gomock.Any()).Return(tt.insertErr)
					mockReq.EXPECT().DirectV1DirectDelete(ctx, testDepDirectID).Return(nil, tt.cleanupErr)
				default:
					mockDB.EXPECT().FlowCreate(ctx, gomock.Any()).Return(nil)
					mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.getErr)
				}
			}

			res, err := h.Create(ctx, testDepCustomerID, flow.TypeFlow, "synthetic", "synthetic detail", tt.persist, actions, uuid.Nil)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_List_dependencyError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB, _ := newDependencyErrorTestHandler(mc)
	ctx := context.Background()

	errDB := errors.New("synthetic database failure")
	filters := map[flow.Field]any{flow.FieldCustomerID: testDepCustomerID}
	mockDB.EXPECT().FlowList(ctx, "synthetic-token", uint64(7), filters).Return(nil, errDB)

	res, err := h.List(ctx, "synthetic-token", 7, filters)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !errors.Is(err, errDB) {
		t.Errorf("Wrong match. expect: %v in the chain, got: %v", errDB, err)
	}
}

func Test_Update_dependencyError(t *testing.T) {
	actions := []action.Action{{Type: action.TypeAnswer}}
	errSynthetic := errors.New("synthetic failure")

	tests := []struct {
		name string

		generateErr error
		updateErr   error
		getErr      error
	}{
		{name: "generating the actions fails", generateErr: errSynthetic},
		{name: "updating the flow fails", updateErr: errSynthetic},
		{name: "updated flow can not be read back", getErr: errSynthetic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, mockDB, mockAction := newDependencyErrorTestHandler(mc)
			ctx := context.Background()

			if tt.generateErr != nil {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(nil, tt.generateErr)
			} else {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(actions, nil)
				mockDB.EXPECT().FlowUpdate(ctx, testDepFlowID, map[flow.Field]any{flow.FieldActions: actions}).Return(tt.updateErr)
				if tt.updateErr == nil {
					mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.getErr)
				}
			}

			res, err := h.Update(ctx, testDepFlowID, nil, nil, actions, nil)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", errSynthetic, err)
			}
		})
	}
}

func Test_Delete_dependencyError(t *testing.T) {
	errSynthetic := errors.New("synthetic failure")

	tests := []struct {
		name string

		directID uuid.UUID

		getErr    error
		directErr error
		deleteErr error
		reGetErr  error

		expectErr        error
		expectErrMessage string
	}{
		{
			name:      "reading the flow fails",
			getErr:    errSynthetic,
			expectErr: errSynthetic,
		},
		{
			name:      "deleting the flow fails and the direct hash is not retried",
			directID:  testDepDirectID,
			directErr: errors.New("synthetic direct failure"),
			deleteErr: errSynthetic,
			expectErr: errSynthetic,
		},
		{
			name:             "deleted flow can not be read back",
			reGetErr:         errSynthetic,
			expectErrMessage: "could not get deleted flow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockReq, mockDB, _ := newDependencyErrorTestHandler(mc)
			ctx := context.Background()

			found := &flow.Flow{Identity: commonidentity.Identity{ID: testDepFlowID}, DirectID: tt.directID}

			if tt.getErr != nil {
				mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.getErr)
			} else {
				mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(found, nil)
				if tt.directID != uuid.Nil {
					mockReq.EXPECT().DirectV1DirectDelete(ctx, tt.directID).Return(nil, tt.directErr)
				}
				mockDB.EXPECT().FlowDelete(ctx, testDepFlowID).Return(tt.deleteErr)
				if tt.deleteErr == nil {
					mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.reGetErr)
				}
			}

			res, err := h.Delete(ctx, testDepFlowID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", tt.expectErr, err)
			}
			if tt.expectErrMessage != "" && (err == nil || err.Error() != tt.expectErrMessage) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErrMessage, err)
			}
		})
	}
}

func Test_UpdateActions_dependencyError(t *testing.T) {
	actions := []action.Action{{Type: action.TypeAnswer}}
	errSynthetic := errors.New("synthetic failure")

	tests := []struct {
		name string

		generateErr error
		updateErr   error
		getErr      error

		expectErrMessage string
	}{
		{name: "generating the actions fails", generateErr: errSynthetic},
		{
			name:             "updating the actions fails",
			updateErr:        errSynthetic,
			expectErrMessage: "could not update the flow actions. flow_id: " + testDepFlowID.String() + ": synthetic failure",
		},
		{name: "updated flow can not be read back", getErr: errSynthetic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, mockDB, mockAction := newDependencyErrorTestHandler(mc)
			ctx := context.Background()

			if tt.generateErr != nil {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(nil, tt.generateErr)
			} else {
				mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(actions, nil)
				mockDB.EXPECT().FlowUpdate(ctx, testDepFlowID, map[flow.Field]any{flow.FieldActions: actions}).Return(tt.updateErr)
				if tt.updateErr == nil {
					mockDB.EXPECT().FlowGet(ctx, testDepFlowID).Return(nil, tt.getErr)
				}
			}

			res, err := h.UpdateActions(ctx, testDepFlowID, actions)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", errSynthetic, err)
			}
			if tt.expectErrMessage != "" && err.Error() != tt.expectErrMessage {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErrMessage, err)
			}
		})
	}
}
