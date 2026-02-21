package accesskeyhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_List(t *testing.T) {

	tests := []struct {
		name    string
		size    uint64
		token   string
		filters map[accesskey.Field]any

		result []*accesskey.Accesskey
	}{
		{
			"normal",
			10,
			"",
			map[accesskey.Field]any{
				accesskey.FieldDeleted: false,
			},

			[]*accesskey.Accesskey{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &accesskeyHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AccesskeyList(gomock.Any(), tt.size, tt.token, tt.filters).Return(tt.result, nil)
			_, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ListByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		size       uint64
		token      string
		customerID uuid.UUID

		responseAccesskeys []*accesskey.Accesskey
		expectFilter       map[accesskey.Field]any
		expectRes          []*accesskey.Accesskey
	}{
		{
			name: "normal",

			size:       100,
			token:      "",
			customerID: uuid.FromStringOrNil("7fdfe8d6-ab12-11ef-9387-339121720d4f"),

			responseAccesskeys: []*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("8032c2ae-ab12-11ef-b470-3772725e480e"),
				},
				{
					ID: uuid.FromStringOrNil("805a864a-ab12-11ef-9dfd-376fdcb46270"),
				},
			},
			expectFilter: map[accesskey.Field]any{
				accesskey.FieldCustomerID: uuid.FromStringOrNil("7fdfe8d6-ab12-11ef-9387-339121720d4f"),
			},
			expectRes: []*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("8032c2ae-ab12-11ef-b470-3772725e480e"),
				},
				{
					ID: uuid.FromStringOrNil("805a864a-ab12-11ef-9dfd-376fdcb46270"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccesskeyList(ctx, gomock.Any(), "", tt.expectFilter).Return(tt.responseAccesskeys, nil)

			res, err := h.GetsByCustomerID(ctx, tt.size, tt.token, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {
	expireTime := time.Date(2024, 4, 4, 7, 15, 59, 233415000, time.UTC)

	tests := []struct {
		name string

		customerID uuid.UUID
		userName   string
		detail     string
		expire     time.Duration

		responseUUID       uuid.UUID
		responseRandomPart string
		responseExpire     *time.Time
		responseTokenHash  string
		expectAccesskey    *accesskey.Accesskey
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("58d43704-a75e-11ef-b9b7-279abaf5dda3"),
			userName:   "test1",
			detail:     "detail1",
			expire:     time.Duration(time.Hour * 24 * 365),

			responseUUID:       uuid.FromStringOrNil("5947fe5a-a75e-11ef-8595-878f92d49c95"),
			responseRandomPart: "abcdefghijklmnopqrstuvwxyz012345",
			responseExpire:     &expireTime,
			responseTokenHash:  "fakehash64chars0000000000000000000000000000000000000000000000000",
			expectAccesskey: &accesskey.Accesskey{
				ID:         uuid.FromStringOrNil("5947fe5a-a75e-11ef-8595-878f92d49c95"),
				CustomerID: uuid.FromStringOrNil("58d43704-a75e-11ef-b9b7-279abaf5dda3"),
				Name:       "test1",
				Detail:     "detail1",
				TokenHash:  "fakehash64chars0000000000000000000000000000000000000000000000000",
				TokenPrefix: "vb_abcdefgh",
				TMExpire:   &expireTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			token := defaultTokenPrefix + tt.responseRandomPart

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeNowAdd(tt.expire).Return(tt.responseExpire)
			mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return(tt.responseRandomPart, nil)
			mockUtil.EXPECT().HashSHA256Hex(token).Return(tt.responseTokenHash)
			mockDB.EXPECT().AccesskeyCreate(ctx, tt.expectAccesskey).Return(nil)
			mockDB.EXPECT().AccesskeyGet(ctx, tt.responseUUID).Return(&accesskey.Accesskey{}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, accesskey.EventTypeAccesskeyCreated, gomock.Any()).Do(
				func(_ context.Context, _ string, data any) {
					ak, ok := data.(*accesskey.Accesskey)
					if !ok {
						t.Errorf("Expected *accesskey.Accesskey, got: %T", data)
						return
					}
					if ak.RawToken != "" {
						t.Errorf("Event should not contain RawToken, got: %s", ak.RawToken)
					}
				},
			).Return()

			res, err := h.Create(ctx, tt.customerID, tt.userName, tt.detail, tt.expire)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			// Verify RawToken is set for one-time return
			if res.RawToken != token {
				t.Errorf("Wrong RawToken. expect: %s, got: %s", token, res.RawToken)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAccesskey *accesskey.Accesskey
		expectRes         *accesskey.Accesskey
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("3c998326-ab11-11ef-a2f1-9359a11fc578"),
			responseAccesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("3c998326-ab11-11ef-a2f1-9359a11fc578"),
			},
			expectRes: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("3c998326-ab11-11ef-a2f1-9359a11fc578"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccesskeyGet(ctx, tt.id).Return(tt.responseAccesskey, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GetByToken(t *testing.T) {

	tests := []struct {
		name string

		token string

		responseTokenHash  string
		responseAccesskeys []*accesskey.Accesskey
		expectFilter       map[accesskey.Field]any
		expectRes          *accesskey.Accesskey
	}{
		{
			name: "normal",

			token: "8043e3fa-ab11-11ef-ba54-cf942545cefe",

			responseTokenHash: "fakehash64chars0000000000000000000000000000000000000000000000000",
			responseAccesskeys: []*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("8061b60a-ab11-11ef-8cd0-4721783d6664"),
				},
			},
			expectFilter: map[accesskey.Field]any{
				accesskey.FieldTokenHash: "fakehash64chars0000000000000000000000000000000000000000000000000",
			},
			expectRes: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("8061b60a-ab11-11ef-8cd0-4721783d6664"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().HashSHA256Hex(tt.token).Return(tt.responseTokenHash)
			mockDB.EXPECT().AccesskeyList(ctx, gomock.Any(), "", tt.expectFilter).Return(tt.responseAccesskeys, nil)

			res, err := h.GetByToken(ctx, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GetByToken_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}
	ctx := context.Background()

	mockUtil.EXPECT().HashSHA256Hex("vb_nonexistenttoken").Return("somehash")
	mockDB.EXPECT().AccesskeyList(ctx, gomock.Any(), "", gomock.Any()).Return([]*accesskey.Accesskey{}, nil)

	_, err := h.GetByToken(ctx, "vb_nonexistenttoken")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_GetByToken_MultipleMatches(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}
	ctx := context.Background()

	mockUtil.EXPECT().HashSHA256Hex("vb_duplicatetoken").Return("somehash")
	mockDB.EXPECT().AccesskeyList(ctx, gomock.Any(), "", gomock.Any()).Return([]*accesskey.Accesskey{
		{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")},
		{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")},
	}, nil)

	_, err := h.GetByToken(ctx, "vb_duplicatetoken")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseAccesskey *accesskey.Accesskey
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("b5be1332-a75d-11ef-8871-ef7e4ff7506a"),

			responseAccesskey: &accesskey.Accesskey{
				ID:       uuid.FromStringOrNil("b5be1332-a75d-11ef-8871-ef7e4ff7506a"),
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &accesskeyHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccesskeyGet(ctx, tt.id).Return(tt.responseAccesskey, nil)
			mockDB.EXPECT().AccesskeyDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AccesskeyGet(ctx, tt.id).Return(tt.responseAccesskey, nil)
			mockNotify.EXPECT().PublishEvent(ctx, accesskey.EventTypeAccesskeyDeleted, tt.responseAccesskey).Return()

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerName string
		detail       string

		expectFields map[accesskey.Field]any
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("459d01b2-a75d-11ef-80a3-03acd83f53cc"),
			customerName: "name new",
			detail:       "detail new",

			expectFields: map[accesskey.Field]any{
				accesskey.FieldName:   "name new",
				accesskey.FieldDetail: "detail new",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AccesskeyUpdate(gomock.Any(), tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().AccesskeyGet(gomock.Any(), gomock.Any()).Return(&accesskey.Accesskey{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), accesskey.EventTypeAccesskeyUpdated, gomock.Any()).Return()

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.customerName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_Delete_AlreadyDeleted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("b5be1332-a75d-11ef-8871-ef7e4ff7506a")
	tmDelete := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	responseAccesskey := &accesskey.Accesskey{
		ID:       id,
		TMDelete: &tmDelete,
	}

	mockDB.EXPECT().AccesskeyGet(ctx, id).Return(responseAccesskey, nil)

	res, err := h.Delete(ctx, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res.TMDelete == nil {
		t.Errorf("Wrong match. expect: already deleted accesskey, got: %v", res)
	}
}

func Test_List_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	filters := map[accesskey.Field]any{
		accesskey.FieldDeleted: false,
	}

	mockDB.EXPECT().AccesskeyList(ctx, uint64(10), "", filters).Return(nil, fmt.Errorf("database error"))
	_, err := h.List(ctx, 10, "", filters)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Get_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		db: mockDB,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("invalid-id")

	mockDB.EXPECT().AccesskeyGet(ctx, id).Return(nil, fmt.Errorf("not found"))
	_, err := h.Get(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Create_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("test-customer-id")
	id := uuid.FromStringOrNil("test-id")

	mockUtil.EXPECT().UUIDCreate().Return(id)
	mockUtil.EXPECT().TimeNowAdd(gomock.Any()).Return(nil)
	mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return("randompart000000000000000000000000", nil)
	mockUtil.EXPECT().HashSHA256Hex(gomock.Any()).Return("fakehash")

	mockDB.EXPECT().AccesskeyCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	_, err := h.Create(ctx, customerID, "test", "detail", time.Hour)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Delete_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		db: mockDB,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("test-id")

	mockDB.EXPECT().AccesskeyGet(ctx, id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_UpdateBasicInfo_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &accesskeyHandler{
		db: mockDB,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("test-id")

	mockDB.EXPECT().AccesskeyUpdate(ctx, id, gomock.Any()).Return(fmt.Errorf("database error"))

	_, err := h.UpdateBasicInfo(ctx, id, "new name", "new detail")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
