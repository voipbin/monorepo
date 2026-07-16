package sessionhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

func Test_SessionHandler_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseSession *session.Session
		expectRes       *session.Session
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			responseSession: &session.Session{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
			expectRes: &session.Session{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &sessionHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().SessionGet(ctx, tt.id).Return(tt.responseSession, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SessionHandler_List(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &sessionHandler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          mockDB,
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
	}
	ctx := context.Background()

	expectRes := []*session.Session{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")}},
	}

	mockDB.EXPECT().SessionList(ctx, uint64(10), "", map[session.Field]any{}).Return(expectRes, nil)

	res, err := h.List(ctx, 10, "", map[session.Field]any{})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_SessionHandler_Delete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &sessionHandler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          mockDB,
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	expectRes := &session.Session{Identity: commonidentity.Identity{ID: id}}

	mockDB.EXPECT().SessionDelete(ctx, id).Return(nil)
	mockDB.EXPECT().SessionGet(ctx, id).Return(expectRes, nil)

	res, err := h.Delete(ctx, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_SessionHandler_End(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &sessionHandler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          mockDB,
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	expectRes := &session.Session{
		Identity: commonidentity.Identity{ID: id},
		Status:   session.StatusEnded,
	}

	mockDB.EXPECT().SessionUpdate(ctx, id, map[session.Field]any{
		session.FieldStatus: session.StatusEnded,
	}).Return(nil)
	mockDB.EXPECT().SessionGet(ctx, id).Return(expectRes, nil)

	res, err := h.End(ctx, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}
