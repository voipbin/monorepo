package messagehandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

var (
	errTestDB = errors.New("test db failure")

	testMessageID  = uuid.FromStringOrNil("6a718293-a4b5-4fc6-9ae7-f80910213243")
	testSessionID  = uuid.FromStringOrNil("7b8293a4-b5c6-4ad7-8bf8-091021324354")
	testCustomerID = uuid.FromStringOrNil("8c93a4b5-c6d7-4be8-9c09-102132435465")
	testSenderID   = uuid.FromStringOrNil("9da4b5c6-d7e8-4cf9-ad10-213243546576")
)

func Test_NewMessageHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	res := NewMessageHandler(mockReq, mockNotify, mockDB)
	h, ok := res.(*messageHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB {
		t.Errorf("Dependencies were not stored as given.")
	}
	if h.utilHandler == nil {
		t.Errorf("Util handler must not be nil.")
	}
}

func Test_MessageHandler_Get_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &messageHandler{db: mockDB}
	ctx := context.Background()

	mockDB.EXPECT().MessageGet(ctx, testMessageID).Return(nil, errTestDB)

	res, err := h.Get(ctx, testMessageID)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_MessageHandler_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &messageHandler{db: mockDB}
	ctx := context.Background()

	filters := map[message.Field]any{message.FieldSessionID: testSessionID}
	mockDB.EXPECT().MessageList(ctx, uint64(20), "token-1", filters).Return(nil, errTestDB)

	res, err := h.List(ctx, 20, "token-1", filters)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_MessageHandler_Delete_errorPaths(t *testing.T) {
	tests := []struct {
		name      string
		deleteErr error
		getErr    error
		expectGet bool
	}{
		{name: "delete fails and no read follows", deleteErr: errTestDB},
		{name: "re-read fails", getErr: errTestDB, expectGet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &messageHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().MessageDelete(ctx, testMessageID).Return(tt.deleteErr)
			if tt.expectGet {
				mockDB.EXPECT().MessageGet(ctx, testMessageID).Return(nil, tt.getErr)
			}

			res, err := h.Delete(ctx, testMessageID)
			if !errors.Is(err, errTestDB) {
				t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_MessageHandler_Create_errorPaths(t *testing.T) {
	tests := []struct {
		name string

		sessionErr error
		createErr  error
		getErr     error

		expectCreate bool
		expectGet    bool
	}{
		{name: "session read fails and no message is stored", sessionErr: errTestDB},
		{name: "store fails and no re-read follows", createErr: errTestDB, expectCreate: true},
		{name: "re-read fails and no event is published", getErr: errTestDB, expectCreate: true, expectGet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			// strict notify mock without expectations: any publish fails the test
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &messageHandler{utilHandler: mockUtil, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			if tt.sessionErr != nil {
				mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(nil, tt.sessionErr)
			} else {
				mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(&session.Session{Identity: commonidentity.Identity{ID: testSessionID}}, nil)
			}
			if tt.expectCreate {
				mockUtil.EXPECT().UUIDCreate().Return(testMessageID)
				mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(tt.createErr)
			}
			if tt.expectGet {
				mockDB.EXPECT().MessageGet(ctx, testMessageID).Return(nil, tt.getErr)
			}

			res, err := h.Create(ctx, testCustomerID, testSessionID, message.DirectionInbound, testSenderID, "hello")
			if !errors.Is(err, errTestDB) {
				t.Errorf("Wrong error. expect: %v, got: %v", errTestDB, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_MessageHandler_Create_nilNotifyHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &messageHandler{utilHandler: mockUtil, db: mockDB}
	ctx := context.Background()

	stored := &message.Message{Identity: commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID}, Text: "hello"}
	mockDB.EXPECT().SessionGet(ctx, testSessionID).Return(&session.Session{Identity: commonidentity.Identity{ID: testSessionID}}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(testMessageID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, testMessageID).Return(stored, nil)

	res, err := h.Create(ctx, testCustomerID, testSessionID, message.DirectionInbound, testSenderID, "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != stored {
		t.Errorf("Wrong result. expect the stored message, got: %v", res)
	}
}
