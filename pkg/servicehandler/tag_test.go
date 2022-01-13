package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestTagCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		user    *user.User
		tagName string
		detail  string

		response  *amtag.Tag
		expectRes *tag.Tag
	}{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"test1 name",
			"test1 detail",

			&amtag.Tag{
				ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			},
			&tag.Tag{
				ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AMV1TagCreate(gomock.Any(), tt.user.ID, tt.tagName, tt.detail).Return(tt.response, nil)

			res, err := h.TagCreate(tt.user, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func TestTagGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		user  *user.User
		size  uint64
		token string

		response  []amtag.Tag
		expectRes []*tag.Tag
	}{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*tag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
		},
		{
			"2 results",
			&user.User{
				ID: 1,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
				},
			},
			[]*tag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1TagGets(gomock.Any(), tt.user.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.TagGets(tt.user, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestTagGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		user  *user.User
		tagID uuid.UUID

		response  *amtag.Tag
		expectRes *tag.Tag
	}{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&amtag.Tag{
				ID:     uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				UserID: 1,
			},
			&tag.Tag{
				ID:     uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.response, nil)

			res, err := h.TagGet(tt.user, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestTagDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		user  *user.User
		tagID uuid.UUID

		resTagGet *amtag.Tag
		expectRes *tag.Tag
	}{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&amtag.Tag{
				ID:     uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				UserID: 1,
			},
			&tag.Tag{
				ID:     uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().AMV1TagDelete(gomock.Any(), tt.tagID).Return(nil)

			if err := h.TagDelete(tt.user, tt.tagID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTagUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		user    *user.User
		tagID   uuid.UUID
		tagName string
		detail  string

		resTagGet *amtag.Tag
		expectRes *tag.Tag
	}{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
			"test1",
			"detail",

			&amtag.Tag{
				ID:     uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				UserID: 1,
			},
			&tag.Tag{
				ID:     uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().AMV1TagUpdate(gomock.Any(), tt.tagID, tt.tagName, tt.detail).Return(nil)

			if err := h.TagUpdate(tt.user, tt.tagID, tt.tagName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
