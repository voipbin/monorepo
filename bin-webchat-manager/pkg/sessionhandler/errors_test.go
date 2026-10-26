package sessionhandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

var (
	errTestDB   = errors.New("test db failure")
	errTestFlow = errors.New("test flow failure")

	testSessionID  = uuid.FromStringOrNil("1b2c3d4e-5f60-4a71-8b92-a3b4c5d6e7f8")
	testCustomerID = uuid.FromStringOrNil("2c3d4e5f-6071-4b82-9ca3-b4c5d6e7f809")
	testWidgetID   = uuid.FromStringOrNil("3d4e5f60-7182-4c93-adb4-c5d6e7f80910")
)

func Test_NewSessionHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockWidget := widgethandler.NewMockWidgetHandler(mc)

	res := NewSessionHandler(mockReq, mockNotify, mockDB, mockWidget)
	h, ok := res.(*sessionHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB || h.widgetHandler != mockWidget {
		t.Errorf("Dependencies were not stored as given.")
	}
	if h.utilHandler == nil {
		t.Errorf("Util handler must not be nil.")
	}
}

func Test_SessionHandler_Get_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &sessionHandler{db: mockDB}
	ctx := context.Background()

	mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(nil, errTestDB)

	res, err := h.Get(ctx, testSessionID)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_SessionHandler_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &sessionHandler{db: mockDB}
	ctx := context.Background()

	filters := map[session.Field]any{session.FieldCustomerID: testCustomerID}
	mockDB.EXPECT().SessionList(ctx, uint64(5), "token-1", filters).Return(nil, errTestDB)

	res, err := h.List(ctx, 5, "token-1", filters)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_SessionHandler_Delete_errorPaths(t *testing.T) {
	tests := []struct {
		name        string
		deleteErr   error
		getErr      error
		expectGet   bool
		expectErrIs error
	}{
		{name: "delete fails and no read follows", deleteErr: errTestDB, expectErrIs: errTestDB},
		{name: "re-read fails", getErr: errTestDB, expectGet: true, expectErrIs: errTestDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &sessionHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().SessionDelete(ctx, testSessionID).Return(tt.deleteErr)
			if tt.expectGet {
				mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(nil, tt.getErr)
			}

			res, err := h.Delete(ctx, testSessionID)
			if !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErrIs, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_SessionHandler_End_errorPaths(t *testing.T) {
	tests := []struct {
		name      string
		updateErr error
		getErr    error
		expectGet bool
	}{
		{name: "update fails and no read follows", updateErr: errTestDB},
		{name: "re-read fails and no event is published", getErr: errTestDB, expectGet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			// strict notify mock without expectations: any publish fails the test
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &sessionHandler{db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().SessionUpdate(ctx, testSessionID, map[session.Field]any{session.FieldStatus: session.StatusEnded}).Return(tt.updateErr)
			if tt.expectGet {
				mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(nil, tt.getErr)
			}

			res, err := h.End(ctx, testSessionID)
			if !errors.Is(err, errTestDB) {
				t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Create_errorPaths(t *testing.T) {
	sess := &session.Session{
		Identity: commonidentity.Identity{ID: testSessionID, CustomerID: testCustomerID},
		WidgetID: testWidgetID,
		Status:   session.StatusActive,
	}

	t.Run("store fails with a generated id and the raw error is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &sessionHandler{utilHandler: mockUtil, db: mockDB}
		ctx := context.Background()

		mockUtil.EXPECT().UUIDCreate().Return(testSessionID)
		mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(errTestDB)

		res, err := h.Create(ctx, uuid.Nil, testCustomerID, testWidgetID, "", "")
		if !errors.Is(err, errTestDB) {
			t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
		}
		if res != nil {
			t.Errorf("Wrong result. expect: nil, got: %v", res)
		}
	})

	t.Run("store fails with a caller specified id and a non duplicate error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &sessionHandler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: mockDB}
		ctx := context.Background()

		mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(errTestDB)

		res, err := h.Create(ctx, testSessionID, testCustomerID, testWidgetID, "", "")
		if !errors.Is(err, errTestDB) {
			t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
		}
		if res != nil {
			t.Errorf("Wrong result. expect: nil, got: %v", res)
		}
	})

	t.Run("re-read fails after store", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockWidget := widgethandler.NewMockWidgetHandler(mc)
		h := &sessionHandler{utilHandler: mockUtil, db: mockDB, widgetHandler: mockWidget}
		ctx := context.Background()

		mockUtil.EXPECT().UUIDCreate().Return(testSessionID)
		mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(nil, errTestDB)

		res, err := h.Create(ctx, uuid.Nil, testCustomerID, testWidgetID, "", "")
		if !errors.Is(err, errTestDB) {
			t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
		}
		if res != nil {
			t.Errorf("Wrong result. expect: nil, got: %v", res)
		}
	})

	t.Run("flow trigger failure still returns the session and records nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockWidget := widgethandler.NewMockWidgetHandler(mc)
		h := &sessionHandler{utilHandler: mockUtil, db: mockDB, reqHandler: mockReq, widgetHandler: mockWidget}
		ctx := context.Background()

		flowID := uuid.FromStringOrNil("4e5f6071-8293-4da4-bec5-d6e7f8091021")
		mockUtil.EXPECT().UUIDCreate().Return(testSessionID)
		mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(sess, nil)
		mockWidget.EXPECT().Get(ctx, testWidgetID).Return(&widget.Widget{SessionFlowID: flowID}, nil)
		mockReq.EXPECT().ConversationV1ConversationCreateAndExecuteFlow(ctx, testCustomerID, flowID, cvconversation.TypeWebchat, "", gomock.Any(), gomock.Any()).Return(nil, errTestFlow)
		// no SessionUpdate expectation: recording the marker after a failed trigger fails the test

		res, err := h.Create(ctx, uuid.Nil, testCustomerID, testWidgetID, "", "")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != sess {
			t.Errorf("Wrong result. expect the stored session, got: %v", res)
		}
	})

	t.Run("marker update failure still returns the session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockWidget := widgethandler.NewMockWidgetHandler(mc)
		h := &sessionHandler{utilHandler: mockUtil, db: mockDB, reqHandler: mockReq, widgetHandler: mockWidget}
		ctx := context.Background()

		flowID := uuid.FromStringOrNil("4e5f6071-8293-4da4-bec5-d6e7f8091021")
		conversationID := uuid.FromStringOrNil("5f607182-93a4-4eb5-8fd6-e7f809102132")
		mockUtil.EXPECT().UUIDCreate().Return(testSessionID)
		mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(sess, nil)
		mockWidget.EXPECT().Get(ctx, testWidgetID).Return(&widget.Widget{SessionFlowID: flowID}, nil)
		mockReq.EXPECT().ConversationV1ConversationCreateAndExecuteFlow(ctx, testCustomerID, flowID, cvconversation.TypeWebchat, "", gomock.Any(), gomock.Any()).
			Return(&cvconversation.Conversation{Identity: commonidentity.Identity{ID: conversationID}}, nil)
		mockDB.EXPECT().SessionUpdate(ctx, testSessionID, map[session.Field]any{session.FieldActiveflowID: conversationID}).Return(errTestDB)

		res, err := h.Create(ctx, uuid.Nil, testCustomerID, testWidgetID, "", "")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != sess {
			t.Errorf("Wrong result. expect the stored session, got: %v", res)
		}
	})
}
