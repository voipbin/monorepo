package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

type errorPathMocks struct {
	util   *utilhandler.MockUtilHandler
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
}

func newErrorPathHandler(mc *gomock.Controller) (*messageHandler, *errorPathMocks) {
	m := &errorPathMocks{
		util:   utilhandler.NewMockUtilHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}

	h := &messageHandler{
		utilHandler:   m.util,
		db:            m.db,
		reqHandler:    m.req,
		notifyHandler: m.notify,
	}

	return h, m
}

func Test_Create_generatesIDAndFailsClosed(t *testing.T) {
	customerID := uuid.FromStringOrNil("755c7b90-197d-11ee-9cb0-a3ddbcba0c6f")
	generatedID := uuid.FromStringOrNil("0c6f1b7a-5b1c-4d6e-9d1f-2a3b4c5d6e7f")

	t.Run("nil id is replaced by a generated id", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		stored := &message.Message{Identity: commonidentity.Identity{ID: generatedID, CustomerID: customerID}}

		m.util.EXPECT().UUIDCreate().Return(generatedID)
		m.db.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *message.Message) error {
			if got.ID != generatedID {
				t.Errorf("Wrong id. expect: %s, got: %s", generatedID, got.ID)
			}
			if got.Type != message.TypeSMS || len(got.Medias) != 0 || got.Medias == nil {
				t.Errorf("Wrong defaults. type: %s, medias: %v", got.Type, got.Medias)
			}
			return nil
		})
		m.db.EXPECT().MessageGet(ctx, generatedID).Return(stored, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, stored)

		res, err := h.Create(ctx, uuid.Nil, customerID, nil, nil, message.ProviderNameTelnyx, "text", message.DirectionInbound)
		if err != nil || res != stored {
			t.Errorf("Wrong match. expect: stored message, got: res: %v, err: %v", res, err)
		}
	})

	t.Run("db create error is returned and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("create failed")
		m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errDB)

		res, err := h.Create(ctx, generatedID, customerID, nil, nil, message.ProviderNameTelnyx, "text", message.DirectionInbound)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})

	t.Run("get after create error is returned and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("get failed")
		m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().MessageGet(ctx, generatedID).Return(nil, errDB)

		res, err := h.Create(ctx, generatedID, customerID, nil, nil, message.ProviderNameTelnyx, "text", message.DirectionInbound)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})
}

func Test_List_pass(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, m := newErrorPathHandler(mc)
	ctx := context.Background()

	filters := map[message.Field]any{message.FieldCustomerID: uuid.FromStringOrNil("755c7b90-197d-11ee-9cb0-a3ddbcba0c6f")}
	expect := []*message.Message{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("6dd9d746-197d-11ee-a39d-0ffbf2a45563")}}}

	m.db.EXPECT().MessageList(ctx, "token", uint64(5), filters).Return(expect, nil)
	res, err := h.List(ctx, "token", 5, filters)
	if err != nil || len(res) != 1 || res[0] != expect[0] {
		t.Errorf("Wrong match. expect: %v, got: res: %v, err: %v", expect, res, err)
	}

	errDB := errors.New("list failed")
	m.db.EXPECT().MessageList(ctx, "token", uint64(5), filters).Return(nil, errDB)
	res, err = h.List(ctx, "token", 5, filters)
	if res != nil || !errors.Is(err, errDB) {
		t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
	}
}

func Test_Get_notFoundIsTyped(t *testing.T) {
	id := uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4")

	t.Run("found", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		expect := &message.Message{Identity: commonidentity.Identity{ID: id}}
		m.db.EXPECT().MessageGet(ctx, id).Return(expect, nil)

		res, err := h.Get(ctx, id)
		if err != nil || res != expect {
			t.Errorf("Wrong match. expect: %v, got: res: %v, err: %v", expect, res, err)
		}
	})

	t.Run("db not found becomes a typed not found error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().MessageGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}

		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("Wrong match. expect: *cerrors.VoipbinError, got: %T %v", err, err)
		}
		if ve.Status != cerrors.StatusNotFound || ve.Reason != "MESSAGE_NOT_FOUND" {
			t.Errorf("Wrong typed error. status: %v, reason: %s", ve.Status, ve.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect the typed error to keep the db error as its cause")
		}
	})

	t.Run("other db errors pass through untyped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("connection lost")
		m.db.EXPECT().MessageGet(ctx, id).Return(nil, errDB)

		res, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if res != nil || !errors.Is(err, errDB) || errors.As(err, &ve) {
			t.Errorf("Wrong match. expect: the plain db error, got: res: %v, err: %v", res, err)
		}
	})
}

func Test_Delete_errorPaths(t *testing.T) {
	id := uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4")
	customerID := uuid.FromStringOrNil("755c7b90-197d-11ee-9cb0-a3ddbcba0c6f")

	t.Run("deleted message is returned and the event is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		deleted := &message.Message{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}
		m.db.EXPECT().MessageDelete(ctx, id).Return(nil)
		m.db.EXPECT().MessageGet(ctx, id).Return(deleted, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageDeleted, deleted)

		res, err := h.Delete(ctx, id)
		if err != nil || res != deleted {
			t.Errorf("Wrong match. expect: %v, got: res: %v, err: %v", deleted, res, err)
		}
	})

	t.Run("delete error stops before the lookup", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("delete failed")
		m.db.EXPECT().MessageDelete(ctx, id).Return(errDB)

		res, err := h.Delete(ctx, id)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})

	t.Run("lookup error after delete publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("get failed")
		m.db.EXPECT().MessageDelete(ctx, id).Return(nil)
		m.db.EXPECT().MessageGet(ctx, id).Return(nil, errDB)

		res, err := h.Delete(ctx, id)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})
}

func Test_dbUpdateTargets_errorPaths(t *testing.T) {
	id := uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4")
	targets := []target.Target{{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"}, Status: target.StatusSent}}

	t.Run("update error stops before the lookup", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("update failed")
		m.db.EXPECT().MessageUpdateTargets(ctx, id, message.ProviderNameTelnyx, targets).Return(errDB)

		res, err := h.dbUpdateTargets(ctx, id, message.ProviderNameTelnyx, targets)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})

	t.Run("lookup error after update publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newErrorPathHandler(mc)
		ctx := context.Background()

		errDB := errors.New("get failed")
		m.db.EXPECT().MessageUpdateTargets(ctx, id, message.ProviderNameTelnyx, targets).Return(nil)
		m.db.EXPECT().MessageGet(ctx, id).Return(nil, errDB)

		res, err := h.dbUpdateTargets(ctx, id, message.ProviderNameTelnyx, targets)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
		}
	})
}

func Test_dbList_dbGet_dbDelete_dbErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, m := newErrorPathHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4")
	errDB := errors.New("db failure")

	m.db.EXPECT().MessageList(ctx, "", uint64(1), nil).Return(nil, errDB)
	if res, err := h.dbList(ctx, "", 1, nil); res != nil || !errors.Is(err, errDB) {
		t.Errorf("dbList: expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
	}

	m.db.EXPECT().MessageGet(ctx, id).Return(nil, errDB)
	if res, err := h.dbGet(ctx, id); res != nil || !errors.Is(err, errDB) {
		t.Errorf("dbGet: expect: nil and %v, got: res: %v, err: %v", errDB, res, err)
	}
}

// telnyxHookPayload builds a telnyx webhook body for the given event type and destinations.
func telnyxHookPayload(eventType string, destinations ...string) []byte {
	to := make([]string, 0, len(destinations))
	for _, d := range destinations {
		to = append(to, fmt.Sprintf(`{"phone_number":"%s","status":"webhook_delivered"}`, d))
	}

	return []byte(fmt.Sprintf(`{
  "data": {
    "event_type": "%s",
    "payload": {
      "direction": "inbound",
      "from": {"phone_number": "+15550100001"},
      "text": "hello",
      "to": [%s]
    }
  }
}`, eventType, strings.Join(to, ",")))
}

func Test_Hook_errorPaths(t *testing.T) {
	const hookURI = "https://hook.example.com/v1.0/hooks/telnyx"
	number := "+15550100002"

	tests := []struct {
		name string

		uri  string
		data []byte

		// mocks that must be called, in order; the strict gomock controller fails on anything else.
		setup func(ctx context.Context, m *errorPathMocks)

		expectErr bool
	}{
		{
			name:      "unknown hook uri",
			uri:       "https://hook.example.com/v1.0/hooks/unknown",
			data:      telnyxHookPayload("message.received", number),
			expectErr: true,
		},
		{
			name:      "hook uri with a suffix variant is not a telnyx hook",
			uri:       hookURI + "/",
			data:      telnyxHookPayload("message.received", number),
			expectErr: true,
		},
		{
			name:      "invalid json",
			uri:       hookURI,
			data:      []byte(`{bad`),
			expectErr: true,
		},
		{
			name:      "empty destination is rejected before the event type check",
			uri:       hookURI,
			data:      telnyxHookPayload("message.sent"),
			expectErr: true,
		},
		{
			name:      "sent event is ignored",
			uri:       hookURI,
			data:      telnyxHookPayload("message.sent", number),
			expectErr: false,
		},
		{
			name:      "finalized event is ignored",
			uri:       hookURI,
			data:      telnyxHookPayload("message.finalized", number),
			expectErr: false,
		},
		{
			name: "number lookup error",
			uri:  hookURI,
			data: telnyxHookPayload("message.received", number),
			setup: func(ctx context.Context, m *errorPathMocks) {
				m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), gomock.Any()).Return(nil, errors.New("lookup failed"))
			},
			expectErr: true,
		},
		{
			name: "no number found",
			uri:  hookURI,
			data: telnyxHookPayload("message.received", number),
			setup: func(ctx context.Context, m *errorPathMocks) {
				m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), gomock.Any()).Return([]nmnumber.Number{}, nil)
			},
			expectErr: true,
		},
		{
			name: "create error",
			uri:  hookURI,
			data: telnyxHookPayload("message.received", number),
			setup: func(ctx context.Context, m *errorPathMocks) {
				m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{Number: number}}, nil)
				m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("0c6f1b7a-5b1c-4d6e-9d1f-2a3b4c5d6e7f"))
				m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errors.New("create failed"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newErrorPathHandler(mc)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(ctx, m)
			}

			err := h.Hook(ctx, tt.uri, tt.data)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Hook_numberFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, m := newErrorPathHandler(mc)
	ctx := context.Background()

	// the first destination decides which number is looked up, and deleted numbers are excluded.
	expectFilters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  "+15550100002",
		nmnumber.FieldDeleted: false,
	}
	m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), expectFilters).Return(nil, errors.New("stop here"))

	if err := h.Hook(ctx, "https://hook.example.com/v1.0/hooks/telnyx", telnyxHookPayload("message.received", "+15550100002", "+15550100003")); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Send_errorPaths(t *testing.T) {
	id := uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7")
	customerID := uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9")
	source := func() *commonaddress.Address {
		return &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
	}
	destinations := func() []commonaddress.Address {
		return []commonaddress.Address{
			{Type: commonaddress.TypeTel, Target: "+15550100002"},
			{Type: commonaddress.TypeTel, Target: "+15550100003"},
		}
	}

	verified := func(ctx context.Context, m *errorPathMocks) {
		m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
			ID:                         customerID,
			IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
		}, nil)
	}

	tests := []struct {
		name string

		setup func(ctx context.Context, m *errorPathMocks)

		expectErrContains string
	}{
		{
			name: "balance check error",
			setup: func(ctx context.Context, m *errorPathMocks) {
				verified(ctx, m)
				m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeSMS, "", 2).Return(false, errors.New("billing unavailable"))
			},
			expectErrContains: "could not validate the customer's balance",
		},
		{
			name: "insufficient balance creates nothing",
			setup: func(ctx context.Context, m *errorPathMocks) {
				verified(ctx, m)
				m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeSMS, "", 2).Return(false, nil)
			},
			expectErrContains: "insufficient balance",
		},
		{
			name: "create error is returned before any provider is used",
			setup: func(ctx context.Context, m *errorPathMocks) {
				verified(ctx, m)
				m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeSMS, "", 2).Return(true, nil)
				m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errors.New("create failed"))
			},
			expectErrContains: "create failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newErrorPathHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.Send(ctx, id, customerID, source(), destinations(), "hello")
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}
