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

func Test_dbList(t *testing.T) {
	tests := []struct {
		name         string
		size         uint64
		token        string
		filters      map[tag.Field]any
		responseTags []*tag.Tag
		dbErr        error
		expectError  bool
	}{
		{
			name:  "successful_list",
			size:  10,
			token: "2020-04-18T03:22:17.995000Z",
			filters: map[tag.Field]any{
				tag.FieldCustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			responseTags: []*tag.Tag{
				{},
			},
			dbErr:       nil,
			expectError: false,
		},
		{
			name:  "db_error",
			size:  10,
			token: "2020-04-18T03:22:17.995000Z",
			filters: map[tag.Field]any{
				tag.FieldCustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			responseTags: nil,
			dbErr:        fmt.Errorf("database error"),
			expectError:  true,
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

			mockDB.EXPECT().TagList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseTags, tt.dbErr)

			res, err := h.dbList(ctx, tt.size, tt.token, tt.filters)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && len(res) != len(tt.responseTags) {
				t.Errorf("Wrong result length. expect: %d, got: %d", len(tt.responseTags), len(res))
			}
		})
	}
}

func Test_dbGet(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		responseTag *tag.Tag
		dbErr       error
		expectError bool
	}{
		{
			name:        "successful_get",
			id:          uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
			responseTag: &tag.Tag{},
			dbErr:       nil,
			expectError: false,
		},
		{
			name:        "db_error",
			id:          uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
			responseTag: nil,
			dbErr:       fmt.Errorf("not found"),
			expectError: true,
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

			mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, tt.dbErr)

			res, err := h.dbGet(ctx, tt.id)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && res == nil {
				t.Errorf("Expected tag, got nil")
			}
		})
	}
}

func Test_dbUpdateInfo(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		tagName     string
		detail      string
		updateErr   error
		getErr      error
		responseTag *tag.Tag
		expectError bool
	}{
		{
			name:        "successful_update",
			id:          uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			tagName:     "test name",
			detail:      "test detail",
			updateErr:   nil,
			getErr:      nil,
			responseTag: &tag.Tag{},
			expectError: false,
		},
		{
			name:        "update_error",
			id:          uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			tagName:     "test name",
			detail:      "test detail",
			updateErr:   fmt.Errorf("update failed"),
			getErr:      nil,
			responseTag: nil,
			expectError: true,
		},
		{
			name:        "get_error_after_update",
			id:          uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			tagName:     "test name",
			detail:      "test detail",
			updateErr:   nil,
			getErr:      fmt.Errorf("get failed"),
			responseTag: nil,
			expectError: true,
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

			mockDB.EXPECT().TagSetBasicInfo(ctx, tt.id, tt.tagName, tt.detail).Return(tt.updateErr)

			if tt.updateErr == nil {
				mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, tt.getErr)
				if tt.getErr == nil {
					mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagUpdated, tt.responseTag)
				}
			}

			res, err := h.dbUpdateInfo(ctx, tt.id, tt.tagName, tt.detail)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && res == nil {
				t.Errorf("Expected tag, got nil")
			}
		})
	}
}

func Test_dbCreate(t *testing.T) {
	tests := []struct {
		name         string
		customerID   uuid.UUID
		tagName      string
		detail       string
		responseUUID uuid.UUID
		createErr    error
		getErr       error
		responseTag  *tag.Tag
		expectError  bool
	}{
		{
			name:         "successful_create",
			customerID:   uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			tagName:      "test name",
			detail:       "test detail",
			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			createErr:    nil,
			getErr:       nil,
			responseTag:  &tag.Tag{},
			expectError:  false,
		},
		{
			name:         "create_error",
			customerID:   uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			tagName:      "test name",
			detail:       "test detail",
			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			createErr:    fmt.Errorf("create failed"),
			getErr:       nil,
			responseTag:  nil,
			expectError:  true,
		},
		{
			name:         "get_error_after_create",
			customerID:   uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			tagName:      "test name",
			detail:       "test detail",
			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			createErr:    nil,
			getErr:       fmt.Errorf("get failed"),
			responseTag:  nil,
			expectError:  true,
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().TagCreate(ctx, gomock.Any()).Return(tt.createErr)

			if tt.createErr == nil {
				mockDB.EXPECT().TagGet(ctx, tt.responseUUID).Return(tt.responseTag, tt.getErr)
				if tt.getErr == nil {
					mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagCreated, tt.responseTag)
				}
			}

			res, err := h.dbCreate(ctx, tt.customerID, tt.tagName, tt.detail)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && res == nil {
				t.Errorf("Expected tag, got nil")
			}
		})
	}
}

func Test_dbDelete(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		deleteErr   error
		getErr      error
		responseTag *tag.Tag
		expectError bool
	}{
		{
			name:        "successful_delete",
			id:          uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
			deleteErr:   nil,
			getErr:      nil,
			responseTag: &tag.Tag{},
			expectError: false,
		},
		{
			name:        "delete_error",
			id:          uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
			deleteErr:   fmt.Errorf("delete failed"),
			getErr:      nil,
			responseTag: nil,
			expectError: true,
		},
		{
			name:        "get_error_after_delete",
			id:          uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
			deleteErr:   nil,
			getErr:      fmt.Errorf("get failed"),
			responseTag: nil,
			expectError: true,
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

			mockDB.EXPECT().TagDelete(ctx, tt.id).Return(tt.deleteErr)

			if tt.deleteErr == nil {
				mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, tt.getErr)
				if tt.getErr == nil {
					mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagDeleted, tt.responseTag)
				}
			}

			res, err := h.dbDelete(ctx, tt.id)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && res == nil {
				t.Errorf("Expected tag, got nil")
			}
		})
	}
}
