package taghandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/dbhandler"
)

func Test_List_Error(t *testing.T) {
	tests := []struct {
		name    string
		size    uint64
		token   string
		filters map[tag.Field]any
		dbErr   error
	}{
		{
			name:  "database_error",
			size:  10,
			token: "2020-04-18T03:22:17.995000Z",
			filters: map[tag.Field]any{
				tag.FieldCustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
				tag.FieldDeleted:    false,
			},
			dbErr: fmt.Errorf("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagList(ctx, tt.size, tt.token, tt.filters).Return(nil, tt.dbErr)

			_, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func Test_Get_Error(t *testing.T) {
	tests := []struct {
		name  string
		id    uuid.UUID
		dbErr error
	}{
		{
			name:  "tag_not_found",
			id:    uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
			dbErr: fmt.Errorf("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagGet(ctx, tt.id).Return(nil, tt.dbErr)

			_, err := h.Get(ctx, tt.id)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func Test_UpdateBasicInfo_Error(t *testing.T) {
	tests := []struct {
		name    string
		id      uuid.UUID
		tagName string
		detail  string
		dbErr   error
	}{
		{
			name:    "update_failed",
			id:      uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			tagName: "test name",
			detail:  "test detail",
			dbErr:   fmt.Errorf("update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagSetBasicInfo(ctx, tt.id, tt.tagName, tt.detail).Return(tt.dbErr)

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.tagName, tt.detail)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func Test_Create_Error(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		tagName    string
		detail     string
		dbErr      error
	}{
		{
			name:       "create_failed",
			customerID: uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			tagName:    "test name",
			detail:     "test detail",
			dbErr:      fmt.Errorf("create failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			responseUUID := uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41")

			mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
			mockDB.EXPECT().TagCreate(ctx, gomock.Any()).Return(tt.dbErr)

			_, err := h.Create(ctx, tt.customerID, tt.tagName, tt.detail)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func Test_Delete_Error(t *testing.T) {
	tests := []struct {
		name  string
		id    uuid.UUID
		dbErr error
	}{
		{
			name:  "delete_failed",
			id:    uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
			dbErr: fmt.Errorf("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagDelete(ctx, tt.id).Return(tt.dbErr)

			_, err := h.Delete(ctx, tt.id)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		})
	}
}

func Test_Create_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := tagHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyhandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310")
	responseUUID := uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41")

	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockDB.EXPECT().TagCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().TagGet(ctx, responseUUID).Return(nil, fmt.Errorf("get failed"))

	_, err := h.Create(ctx, customerID, "name", "detail")
	if err == nil {
		t.Errorf("Expected error when get fails after create, got nil")
	}
}

func Test_UpdateBasicInfo_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := tagHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyhandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed")

	mockDB.EXPECT().TagSetBasicInfo(ctx, id, "name", "detail").Return(nil)
	mockDB.EXPECT().TagGet(ctx, id).Return(nil, fmt.Errorf("get failed"))

	_, err := h.UpdateBasicInfo(ctx, id, "name", "detail")
	if err == nil {
		t.Errorf("Expected error when get fails after update, got nil")
	}
}

func Test_Delete_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := tagHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyhandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5")

	mockDB.EXPECT().TagDelete(ctx, id).Return(nil)
	mockDB.EXPECT().TagGet(ctx, id).Return(nil, fmt.Errorf("get failed"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Expected error when get fails after delete, got nil")
	}
}
