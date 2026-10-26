package accounthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-conversation-manager/internal/logtest"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

const (
	sentinelSecret = "SENTINELsecret123"
	sentinelToken  = "SENTINELtoken456"
)

func newMockUtilForCreate(mc *gomock.Controller, id uuid.UUID) *utilhandler.MockUtilHandler {
	m := utilhandler.NewMockUtilHandler(mc)
	m.EXPECT().UUIDCreate().Return(id).AnyTimes()
	return m
}

func newSecretAccount() *account.Account {
	return &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("5c1d3a52-5d7e-4e0c-9f6a-0a1b2c3d4e5f"),
			CustomerID: uuid.FromStringOrNil("6d2e4b63-6e8f-4f1d-8a7b-1b2c3d4e5f60"),
		},
		Type:   account.TypeSMS,
		Name:   "test name",
		Secret: sentinelSecret,
		Token:  sentinelToken,
	}
}

// assertPublishedWithoutSecret returns the matcher callback: the published value must be a stripped WebhookMessage that
// keeps the account id as its subscription address.
func assertPublishedWithoutSecret(t *testing.T, id uuid.UUID) func(context.Context, uuid.UUID, string, notifyhandler.WebhookEventMessage) {
	t.Helper()

	return func(_ context.Context, _ uuid.UUID, _ string, data notifyhandler.WebhookEventMessage) {
		if res := fmt.Sprintf("%T", data); res != "*account.WebhookMessage" {
			t.Errorf("Wrong published type. expect: *account.WebhookMessage, got: %s", res)
		}

		if res := data.EventSubscriptionID(); res != id.String() {
			t.Errorf("Wrong subscription id. expect: %s, got: %s", id, res)
		}

		b, err := json.Marshal(data)
		if err != nil {
			t.Errorf("Could not marshal the published value. err: %v", err)
			return
		}

		var m map[string]any
		if errUnmarshal := json.Unmarshal(b, &m); errUnmarshal != nil {
			t.Errorf("Could not unmarshal the published value. err: %v", errUnmarshal)
			return
		}
		for _, key := range []string{"secret", "token"} {
			if _, ok := m[key]; ok {
				t.Errorf("The published event has the %q key", key)
			}
		}
		if m["id"] != id.String() {
			t.Errorf("Wrong id in the published event. expect: %s, got: %v", id, m["id"])
		}
	}
}

func Test_Create_Update_Delete_publishedEventHasNoSecret(t *testing.T) {
	ctx := context.Background()
	ac := newSecretAccount()

	tests := []struct {
		name string
		call func(h *accountHandler, mockDB *dbhandler.MockDBHandler) error
		// eventType is the expected event type
		eventType string
	}{
		{
			name: "create",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) error {
				mockDB.EXPECT().AccountCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().AccountGet(gomock.Any(), gomock.Any()).Return(ac, nil)
				_, err := h.Create(ctx, ac.CustomerID, ac.Type, ac.Name, "", sentinelSecret, sentinelToken, uuid.Nil, nil)
				return err
			},
			eventType: account.EventTypeAccountCreated,
		},
		{
			name: "update",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) error {
				mockDB.EXPECT().AccountUpdate(ctx, ac.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, ac.ID).Return(ac, nil)
				_, err := h.Update(ctx, ac.ID, map[account.Field]any{account.FieldName: "new"})
				return err
			},
			eventType: account.EventTypeAccountUpdated,
		},
		{
			name: "delete",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) error {
				mockDB.EXPECT().AccountGet(ctx, ac.ID).Return(ac, nil).Times(2)
				mockDB.EXPECT().AccountDelete(ctx, ac.ID).Return(nil)
				_, err := h.Delete(ctx, ac.ID)
				return err
			},
			eventType: account.EventTypeAccountDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := newMockUtilForCreate(mc, ac.ID)

			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), ac.CustomerID, tt.eventType, gomock.Any()).
				Do(assertPublishedWithoutSecret(t, ac.ID))

			if err := tt.call(h, mockDB); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_logsHaveNoSecret(t *testing.T) {
	ctx := context.Background()
	ac := newSecretAccount()

	tests := []struct {
		name string
		call func(h *accountHandler, mockDB *dbhandler.MockDBHandler)
	}{
		{
			name: "get",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().AccountGet(ctx, ac.ID).Return(ac, nil)
				_, _ = h.Get(ctx, ac.ID)
			},
		},
		{
			name: "list with a secret in the filters",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) {
				filters := map[account.Field]any{account.FieldSecret: sentinelSecret}
				mockDB.EXPECT().AccountList(ctx, uint64(10), "", filters).Return([]*account.Account{ac}, nil)
				_, _ = h.List(ctx, "", 10, filters)
			},
		},
		{
			name: "update with a secret and a token in the fields",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) {
				fields := map[account.Field]any{account.FieldSecret: sentinelSecret, account.FieldToken: sentinelToken}
				mockDB.EXPECT().AccountUpdate(ctx, ac.ID, fields).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, ac.ID).Return(ac, nil)
				_, _ = h.Update(ctx, ac.ID, fields)
			},
		},
		{
			name: "delete",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().AccountGet(ctx, ac.ID).Return(ac, nil).Times(2)
				mockDB.EXPECT().AccountDelete(ctx, ac.ID).Return(nil)
				_, _ = h.Delete(ctx, ac.ID)
			},
		},
		{
			name: "create",
			call: func(h *accountHandler, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().AccountCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().AccountGet(gomock.Any(), gomock.Any()).Return(ac, nil)
				_, _ = h.Create(ctx, ac.CustomerID, ac.Type, ac.Name, "", sentinelSecret, sentinelToken, uuid.Nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logtest.Capture(t)

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			h := &accountHandler{
				utilHandler:   newMockUtilForCreate(mc, ac.ID),
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			tt.call(h, mockDB)

			logtest.AssertNoLeak(t, hook, sentinelSecret, sentinelToken)
		})
	}
}
