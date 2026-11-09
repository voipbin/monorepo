package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/device"
)

var (
	testCustomerA = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testCustomerB = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d31")
	testAgentA    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testAgentB    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e53")
)

func newTestDevice(id, customerID, agentID uuid.UUID, token string) *device.Device {
	return &device.Device{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		AgentID:  agentID,
		Platform: device.PlatformAndroid,
		Token:    token,
	}
}

func newTestHandler(t *testing.T, mockUtil *utilhandler.MockUtilHandler) *handler {
	t.Helper()

	// every test starts from an empty table
	if _, err := dbTest.Exec("delete from push_devices"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return &handler{
		utilHandler: mockUtil,
		db:          dbTest,
	}
}

func Test_DeviceCreate_Get(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	now := time.Date(2026, 3, 1, 10, 0, 0, 123000000, time.UTC)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockUtil)
	ctx := context.Background()

	id := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
	d := newTestDevice(id, testCustomerA, testAgentA, "token-a")

	mockUtil.EXPECT().TimeNow().Return(&now)
	if err := h.DeviceCreate(ctx, d); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	byID, err := h.DeviceGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if byID.Token != "token-a" || byID.AgentID != testAgentA || byID.CustomerID != testCustomerA || byID.Platform != device.PlatformAndroid {
		t.Errorf("Wrong match. got: %v", byID)
	}
	if byID.TMCreate == nil || !byID.TMCreate.Equal(now) || byID.TMUpdate == nil || !byID.TMUpdate.Equal(now) {
		t.Errorf("Wrong match. the create and refresh time start at now. got: %v, %v", byID.TMCreate, byID.TMUpdate)
	}

	byToken, err := h.DeviceGetByToken(ctx, "token-a")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if byToken.ID != id {
		t.Errorf("Wrong match. expect: %s, got: %s", id, byToken.ID)
	}

	// the token is case sensitive
	if _, err := h.DeviceGetByToken(ctx, "TOKEN-A"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
	if _, err := h.DeviceGet(ctx, uuid.Must(uuid.NewV4())); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_DeviceTouch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	created := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	touched := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockUtil)
	ctx := context.Background()

	id := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
	mockUtil.EXPECT().TimeNow().Return(&created)
	if err := h.DeviceCreate(ctx, newTestDevice(id, testCustomerA, testAgentA, "token-a")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&touched)
	if err := h.DeviceTouch(ctx, id); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err := h.DeviceGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !got.TMUpdate.Equal(touched) || !got.TMCreate.Equal(created) {
		t.Errorf("Wrong match. the refresh time moves, the create time stays. got: %v, %v", got.TMCreate, got.TMUpdate)
	}
}

func Test_DeviceList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockUtil)
	ctx := context.Background()

	rows := []struct {
		id       string
		customer uuid.UUID
		agent    uuid.UUID
		token    string
		at       time.Time
	}{
		{"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b01", testCustomerA, testAgentA, "t1", time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)},
		{"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b02", testCustomerA, testAgentA, "t2", time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)},
		{"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b03", testCustomerA, testAgentB, "t3", time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)},
		{"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b04", testCustomerB, testAgentA, "t4", time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)},
	}
	for _, r := range rows {
		at := r.at
		mockUtil.EXPECT().TimeNow().Return(&at)
		if err := h.DeviceCreate(ctx, newTestDevice(uuid.FromStringOrNil(r.id), r.customer, r.agent, r.token)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	const farFuture = "9999-12-31T23:59:59.999999Z"

	tests := []struct {
		name    string
		size    uint64
		filters map[device.Field]any
		expect  []string
	}{
		{"one agent of one customer, newest first", 10, map[device.Field]any{device.FieldCustomerID: testCustomerA, device.FieldAgentID: testAgentA}, []string{"t2", "t1"}},
		{"the same agent id of another customer is not mixed in", 10, map[device.Field]any{device.FieldCustomerID: testCustomerB, device.FieldAgentID: testAgentA}, []string{"t4"}},
		{"size limit", 1, map[device.Field]any{device.FieldCustomerID: testCustomerA}, []string{"t3"}},
		{"nothing", 10, map[device.Field]any{device.FieldAgentID: uuid.Must(uuid.NewV4())}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.DeviceList(ctx, tt.size, farFuture, tt.filters)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			tokens := []string{}
			for _, d := range got {
				tokens = append(tokens, d.Token)
			}
			if len(tokens) != len(tt.expect) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expect, tokens)
			}
			for i := range tokens {
				if tokens[i] != tt.expect[i] {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, tokens)
				}
			}
		})
	}
}

func Test_DeviceDelete_variants(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockUtil)
	ctx := context.Background()

	idA := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b01")
	idB := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b02")
	idC := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b03")
	idD := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b04")

	create := func(id, customer, agent uuid.UUID, token string) {
		mockUtil.EXPECT().TimeNow().Return(&now)
		if err := h.DeviceCreate(ctx, newTestDevice(id, customer, agent, token)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
	exists := func(id uuid.UUID) bool {
		_, err := h.DeviceGet(ctx, id)
		return err == nil
	}

	create(idA, testCustomerA, testAgentA, "t1")
	create(idB, testCustomerA, testAgentB, "t2")
	create(idC, testCustomerB, testAgentA, "t3")
	create(idD, testCustomerB, testAgentB, "t4")

	// the owner of the token keeps it
	if err := h.DeviceDeleteByTokenNotOwner(ctx, "t1", testCustomerA, testAgentA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !exists(idA) {
		t.Errorf("Wrong match. the owner's device must stay")
	}

	// the same agent id in another customer is not the owner, and neither is another agent
	for _, tc := range []struct {
		customer, agent uuid.UUID
	}{{testCustomerB, testAgentA}, {testCustomerA, testAgentB}} {
		if err := h.DeviceDeleteByTokenNotOwner(ctx, "t1", tc.customer, tc.agent); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if exists(idA) {
			t.Errorf("Wrong match. a device of another owner must be deleted. owner: %v", tc)
		}
		create(idA, testCustomerA, testAgentA, "t1")
	}

	// only the device with that token is touched
	if !exists(idB) || !exists(idC) || !exists(idD) {
		t.Errorf("Wrong match. the other devices must stay")
	}

	if err := h.DeviceDelete(ctx, idB); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if exists(idB) {
		t.Errorf("Wrong match. the device must be deleted")
	}

	if err := h.DeviceDeleteByAgentID(ctx, testAgentA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if exists(idA) || exists(idC) || !exists(idD) {
		t.Errorf("Wrong match. all devices of the agent, and only those, must be deleted")
	}

	if err := h.DeviceDeleteByCustomerID(ctx, testCustomerB); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if exists(idD) {
		t.Errorf("Wrong match. all devices of the customer must be deleted")
	}
}

func Test_isDuplicate(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"duplicate entry", &mysql.MySQLError{Number: 1062}, true},
		{"wrapped duplicate entry", errors.Join(errors.New("x"), &mysql.MySQLError{Number: 1062}), true},
		{"another mysql error", &mysql.MySQLError{Number: 1146}, false},
		{"not a mysql error", errors.New("boom"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicate(tt.err); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_classifyCreateError(t *testing.T) {
	dup := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'fake-token-value' for key 'idx_push_devices_token'"}

	got := classifyCreateError(dup)
	if !errors.Is(got, ErrDuplicate) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrDuplicate, got)
	}
	if strings.Contains(got.Error(), "fake-token-value") {
		t.Errorf("Wrong match. the key value must not be in the error. got: %v", got)
	}

	other := errors.New("boom")
	got = classifyCreateError(other)
	if errors.Is(got, ErrDuplicate) || !errors.Is(got, other) {
		t.Errorf("Wrong match. expect a wrapped %v, got: %v", other, got)
	}
}

func Test_NewHandler(t *testing.T) {
	if h := NewHandler(dbTest); h == nil {
		t.Errorf("Wrong match. expect: a handler, got: nil")
	}
}
