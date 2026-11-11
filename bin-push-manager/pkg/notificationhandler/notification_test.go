package notificationhandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/pushhandler"
)

var (
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testAgentID    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testNow        = time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
)

const fakeToken = "fake-device-token-AbCdEf-0123456789"

func testDevice(token string, tmUpdate time.Time) *device.Device {
	return &device.Device{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: testCustomerID},
		AgentID:  testAgentID,
		Platform: device.PlatformAndroid,
		Token:    token,
		TMCreate: &tmUpdate,
		TMUpdate: &tmUpdate,
	}
}

func newTestHandler(mc *gomock.Controller) (*notificationHandler, *dbhandler.MockDBHandler, *pushhandler.MockSender) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSender := pushhandler.NewMockSender(mc)

	return &notificationHandler{
		db:     mockDB,
		sender: mockSender,
		now:    func() time.Time { return testNow },
	}, mockDB, mockSender
}

var listFilters = map[device.Field]any{
	device.FieldCustomerID: testCustomerID,
	device.FieldAgentID:    testAgentID,
}

func Test_Send_allDevices(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockSender := newTestHandler(mc)
	ctx := context.Background()

	ok1 := testDevice("tok-ok-1", testNow)
	ok2 := testDevice("tok-ok-2", testNow)
	gone := testDevice("tok-gone", testNow)
	failing := testDevice("tok-fail", testNow)
	old := testDevice("tok-old", testNow.Add(-device.Expiry-time.Second))
	data := map[string]string{"call_id": "c1"}

	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{ok1, gone, old, failing, ok2}, nil)
	mockSender.EXPECT().Send(gomock.Any(), "tok-ok-1", notification.KindCallCancelled, data).Return(nil)
	mockSender.EXPECT().Send(gomock.Any(), "tok-ok-2", notification.KindCallCancelled, data).Return(nil)
	mockSender.EXPECT().Send(gomock.Any(), "tok-gone", notification.KindCallCancelled, data).Return(pushhandler.ErrUnregistered)
	mockSender.EXPECT().Send(gomock.Any(), "tok-fail", notification.KindCallCancelled, data).Return(fmt.Errorf("transient"))
	// the unregistered token deletes its device. The transient failure keeps its device.
	mockDB.EXPECT().DeviceDelete(ctx, gone.ID).Return(nil)

	res, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallCancelled, data)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	// the expired device is not addressed at all
	if res.DeviceCount != 4 || res.SentCount != 2 {
		t.Errorf("Wrong match. expect: {4 2}, got: %v", res)
	}
}

// The metric labels and the per-device limit are part of the contract (dashboard, design).
func Test_Send_metricsAndLimit(t *testing.T) {
	if deliveryTimeout != 5*time.Second {
		t.Errorf("Wrong match. expect: 5s, got: %v", deliveryTimeout)
	}

	tests := []struct {
		name   string
		err    error
		result string
	}{
		{"sent", nil, resultSent},
		{"unregistered", pushhandler.ErrUnregistered, resultUnregistered},
		{"failed", fmt.Errorf("transient"), resultFailed},
		{"not configured", pushhandler.ErrNotConfigured, resultNotConfigured},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockSender := newTestHandler(mc)
			ctx := context.Background()
			d := testDevice(fakeToken, testNow)

			mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{d}, nil)
			mockSender.EXPECT().Send(gomock.Any(), fakeToken, notification.KindCallIncoming, gomock.Any()).Return(tt.err)
			if tt.err == pushhandler.ErrUnregistered {
				mockDB.EXPECT().DeviceDelete(ctx, d.ID).Return(nil)
			}

			counter := func(result string) float64 {
				return testutil.ToFloat64(promNotificationTotal.WithLabelValues(string(notification.KindCallIncoming), result))
			}
			before := map[string]float64{}
			for _, r := range []string{resultSent, resultUnregistered, resultFailed, resultNotConfigured} {
				before[r] = counter(r)
			}

			if _, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			for r, b := range before {
				expect := 0.0
				if r == tt.result {
					expect = 1
				}
				if got := counter(r) - b; got != expect {
					t.Errorf("Wrong match. result: %s, expect: %v, got: %v", r, expect, got)
				}
			}
		})
	}
}

func Test_Send_oneDevice(t *testing.T) {
	other := uuid.Must(uuid.NewV4())

	tests := []struct {
		name         string
		dev          func() *device.Device
		dbErr        error
		expectSend   bool
		expectDevice int
		expectErr    bool
	}{
		{"the device of the agent", func() *device.Device { return testDevice(fakeToken, testNow) }, nil, true, 1, false},
		{"a device of another agent is not addressed", func() *device.Device {
			d := testDevice(fakeToken, testNow)
			d.AgentID = other
			return d
		}, nil, false, 0, false},
		{"a device of another customer is not addressed", func() *device.Device {
			d := testDevice(fakeToken, testNow)
			d.CustomerID = other
			return d
		}, nil, false, 0, false},
		{"an expired device is not addressed", func() *device.Device { return testDevice(fakeToken, testNow.Add(-device.Expiry-time.Second)) }, nil, false, 0, false},
		{"a missing device is not an error", func() *device.Device { return nil }, dbhandler.ErrNotFound, false, 0, false},
		{"a database error is an error", func() *device.Device { return nil }, fmt.Errorf("boom"), false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockSender := newTestHandler(mc)
			ctx := context.Background()
			d := tt.dev()
			id := uuid.Must(uuid.NewV4())
			if d != nil {
				id = d.ID
			}

			mockDB.EXPECT().DeviceGet(ctx, id).Return(d, tt.dbErr)
			if tt.expectSend {
				mockSender.EXPECT().Send(gomock.Any(), fakeToken, notification.KindCallIncoming, gomock.Any()).Return(nil)
			}

			res, err := h.Send(ctx, testCustomerID, testAgentID, id, notification.KindCallIncoming, map[string]string{"join_address": "a1"})
			if tt.expectErr != (err != nil) {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err == nil && (res.DeviceCount != tt.expectDevice || res.SentCount != tt.expectDevice) {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectDevice, res)
			}
		})
	}
}

func Test_Send_noDevices(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTestHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{}, nil)
	res, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.DeviceCount != 0 || res.SentCount != 0 {
		t.Errorf("Wrong match. expect: {0 0}, got: %v", res)
	}

	boom := fmt.Errorf("boom")
	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return(nil, boom)
	if _, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil); !errors.Is(err, boom) {
		t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
	}
}

func Test_Send_validation(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		agentID    uuid.UUID
		kind       notification.Kind
		data       map[string]string
	}{
		{"no customer", uuid.Nil, testAgentID, notification.KindCallIncoming, nil},
		{"no agent", testCustomerID, uuid.Nil, notification.KindCallIncoming, nil},
		{"unknown kind", testCustomerID, testAgentID, notification.Kind("unknown_kind"), nil},
		{"the kind key belongs to push-manager", testCustomerID, testAgentID, notification.KindCallIncoming, map[string]string{notification.DataKeyKind: "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// neither the database nor the sender is called
			h, _, _ := newTestHandler(mc)
			if _, err := h.Send(context.Background(), tt.customerID, tt.agentID, uuid.Nil, tt.kind, tt.data); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_Send_notConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockSender := newTestHandler(mc)
	ctx := context.Background()
	d := testDevice(fakeToken, testNow)

	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{d}, nil)
	mockSender.EXPECT().Send(gomock.Any(), fakeToken, notification.KindCallIncoming, gomock.Any()).Return(pushhandler.ErrNotConfigured)

	before := testutil.ToFloat64(promNotificationTotal.WithLabelValues(string(notification.KindCallIncoming), resultNotConfigured))
	res, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	// the device stays, nothing is deleted (no DeviceDelete expectation) and the result counts as not sent
	if res.DeviceCount != 1 || res.SentCount != 0 {
		t.Errorf("Wrong match. expect: {1 0}, got: %v", res)
	}
	after := testutil.ToFloat64(promNotificationTotal.WithLabelValues(string(notification.KindCallIncoming), resultNotConfigured))
	if after-before != 1 {
		t.Errorf("Wrong match. the not_configured counter must grow by 1. got: %v", after-before)
	}
}

func Test_Send_deliveryTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockSender := newTestHandler(mc)
	ctx := context.Background()
	d := testDevice(fakeToken, testNow)

	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{d}, nil)
	mockSender.EXPECT().Send(gomock.Any(), fakeToken, notification.KindCallIncoming, gomock.Any()).DoAndReturn(
		func(c context.Context, _ string, _ notification.Kind, _ map[string]string) error {
			dl, ok := c.Deadline()
			if !ok {
				t.Errorf("Wrong match. the delivery must have a deadline")
			} else if left := time.Until(dl); left > deliveryTimeout || left < deliveryTimeout-time.Second {
				t.Errorf("Wrong match. expect about %v, got: %v", deliveryTimeout, left)
			}
			return context.DeadlineExceeded
		})

	res, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.SentCount != 0 {
		t.Errorf("Wrong match. expect: 0, got: %v", res)
	}
}

func Test_Send_deliversInParallel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockSender := newTestHandler(mc)
	ctx := context.Background()

	devices := []*device.Device{testDevice("a", testNow), testDevice("b", testNow), testDevice("c", testNow)}
	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return(devices, nil)

	// all three deliveries must be in flight at once, or the barrier never opens
	var wg sync.WaitGroup
	wg.Add(len(devices))
	opened := make(chan struct{})
	go func() { wg.Wait(); close(opened) }()
	mockSender.EXPECT().Send(gomock.Any(), gomock.Any(), notification.KindCallIncoming, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ notification.Kind, _ map[string]string) error {
			wg.Done()
			select {
			case <-opened:
				return nil
			case <-time.After(2 * time.Second):
				return fmt.Errorf("deliveries are not parallel")
			}
		}).Times(len(devices))

	res, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil)
	if err != nil || res.SentCount != 3 {
		t.Errorf("Wrong match. expect: 3 sent, got: %v, %v", res, err)
	}
}

func Test_Send_neverLogsTheToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var logs bytes.Buffer
	logrus.SetOutput(&logs)
	defer logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.DebugLevel)

	h, mockDB, mockSender := newTestHandler(mc)
	ctx := context.Background()
	d := testDevice(fakeToken, testNow)

	mockDB.EXPECT().DeviceList(ctx, uint64(listSize), listPageToken, listFilters).Return([]*device.Device{d}, nil)
	mockSender.EXPECT().Send(gomock.Any(), fakeToken, notification.KindCallIncoming, gomock.Any()).Return(fmt.Errorf("transient"))

	if _, err := h.Send(ctx, testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(logs.String(), fakeToken) {
		t.Errorf("Wrong match. the token must not be in the log. got: %s", logs.String())
	}
}

func Test_NewNotificationHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	if h := NewNotificationHandler(dbhandler.NewMockDBHandler(mc), pushhandler.NewMockSender(mc)); h == nil {
		t.Errorf("Wrong match. expect: a handler, got: nil")
	}
}
