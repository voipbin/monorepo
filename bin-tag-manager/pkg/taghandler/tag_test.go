package taghandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseTags []*tag.Tag
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			size:       10,
			token:      "2020-04-18 03:22:17.995000",

			responseTags: []*tag.Tag{
				{
					ID: uuid.FromStringOrNil("a0c95b3e-2a00-11ee-a3cd-3307849aa505"),
				},
			},
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

			mockDB.EXPECT().TagGets(ctx, tt.customerID, tt.size, tt.token).Return(tt.responseTags, nil)
			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseTags, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTags, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTag *tag.Tag
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),

			responseTag: &tag.Tag{
				ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
			},
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

			mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, nil)
			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseTag, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTag, res)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		tagName string
		detail  string

		responseTag *tag.Tag
	}{
		{
			name: "normal",

			id:      uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			tagName: "test name",
			detail:  "test detail",

			responseTag: &tag.Tag{
				ID: uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			},
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

			mockDB.EXPECT().TagSetBasicInfo(ctx, tt.id, tt.tagName, tt.detail).Return(nil)
			mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, nil)
			mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagUpdated, tt.responseTag)
			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseTag, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTag, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		tagName    string
		detail     string

		responseUUID uuid.UUID
		responseTag  *tag.Tag
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			tagName:    "test name",
			detail:     "test detail",

			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			responseTag: &tag.Tag{
				ID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			},
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
			mockDB.EXPECT().TagCreate(ctx, &tag.Tag{
				ID:         tt.responseUUID,
				CustomerID: tt.customerID,
				Name:       tt.tagName,
				Detail:     tt.detail,
			}).Return(nil)
			mockDB.EXPECT().TagGet(ctx, tt.responseUUID).Return(tt.responseTag, nil)
			mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagCreated, tt.responseTag)

			res, err := h.Create(ctx, tt.customerID, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseTag, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTag, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTag *tag.Tag

		expectNewTagIds [][]uuid.UUID
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),

			responseTag: &tag.Tag{
				ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
			},

			expectNewTagIds: [][]uuid.UUID{
				{
					uuid.FromStringOrNil("bf85e8e4-2a4b-11ee-82ac-c3736324422f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TagGet(ctx, tt.id).Return(tt.responseTag, nil)
			mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagDeleted, tt.responseTag)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseTag, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTag, res)
			}
		})
	}
}
