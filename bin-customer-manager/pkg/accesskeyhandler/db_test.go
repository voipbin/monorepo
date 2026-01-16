package accesskeyhandler

import (
	"context"
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

	tests := []struct {
		name string

		customerID uuid.UUID
		userName   string
		detail     string
		expire     time.Duration

		responseUUID    uuid.UUID
		responseToken   string
		responseExpire  string
		expectAccesskey *accesskey.Accesskey
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("58d43704-a75e-11ef-b9b7-279abaf5dda3"),
			userName:   "test1",
			detail:     "detail1",
			expire:     time.Duration(time.Hour * 24 * 365),

			responseUUID:   uuid.FromStringOrNil("5947fe5a-a75e-11ef-8595-878f92d49c95"),
			responseToken:  "test_token",
			responseExpire: "2024-04-04 07:15:59.233415",
			expectAccesskey: &accesskey.Accesskey{
				ID:         uuid.FromStringOrNil("5947fe5a-a75e-11ef-8595-878f92d49c95"),
				CustomerID: uuid.FromStringOrNil("58d43704-a75e-11ef-b9b7-279abaf5dda3"),
				Name:       "test1",
				Detail:     "detail1",
				Token:      "test_token",
				TMExpire:   "2024-04-04 07:15:59.233415",
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTimeAdd(tt.expire).Return(tt.responseExpire)
			mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return(tt.responseToken, nil)
			mockDB.EXPECT().AccesskeyCreate(ctx, tt.expectAccesskey).Return(nil)
			mockDB.EXPECT().AccesskeyGet(ctx, tt.responseUUID).Return(&accesskey.Accesskey{}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, accesskey.EventTypeAccesskeyCreated, gomock.Any()).Return()

			_, err := h.Create(ctx, tt.customerID, tt.userName, tt.detail, tt.expire)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
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

		responseAccesskeys []*accesskey.Accesskey
		expectFilter       map[accesskey.Field]any
		expectRes          *accesskey.Accesskey
	}{
		{
			name: "normal",

			token: "8043e3fa-ab11-11ef-ba54-cf942545cefe",

			responseAccesskeys: []*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("8061b60a-ab11-11ef-8cd0-4721783d6664"),
				},
			},
			expectFilter: map[accesskey.Field]any{
				accesskey.FieldToken: "8043e3fa-ab11-11ef-ba54-cf942545cefe",
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
				TMDelete: dbhandler.DefaultTimeStamp,
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
