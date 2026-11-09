package devicehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/pkg/dbhandler"
)

var (
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testAgentID    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testDeviceID   = uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
	testNow        = time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
)

func newTestHandler(mc *gomock.Controller) (*deviceHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	return &deviceHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		now:         func() time.Time { return testNow },
	}, mockDB, mockUtil
}

func testDevice(id uuid.UUID, token string, tmUpdate time.Time) *device.Device {
	return &device.Device{
		Identity: commonidentity.Identity{ID: id, CustomerID: testCustomerID},
		AgentID:  testAgentID,
		Platform: device.PlatformAndroid,
		Token:    token,
		TMCreate: &tmUpdate,
		TMUpdate: &tmUpdate,
	}
}

func assertVoipbinError(t *testing.T, err error, reason string) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: a typed error, got: %v", err)
	}
	if ve.Reason != reason {
		t.Errorf("Wrong match. expect reason: %s, got: %s", reason, ve.Reason)
	}
}

func Test_Register_new(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil := newTestHandler(mc)
	ctx := context.Background()

	created := testDevice(testDeviceID, "token-a", testNow)

	mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil)
	mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, dbhandler.ErrNotFound)
	mockUtil.EXPECT().UUIDCreate().Return(testDeviceID)
	mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, d *device.Device) error {
		if d.ID != testDeviceID || d.CustomerID != testCustomerID || d.AgentID != testAgentID || d.Platform != device.PlatformAndroid || d.Token != "token-a" {
			t.Errorf("Wrong match. got: %v", d)
		}
		return nil
	})
	mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(created, nil)
	mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{created}, nil)

	res, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != created {
		t.Errorf("Wrong match. expect: %v, got: %v", created, res)
	}
}

func Test_Register_refresh(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTestHandler(mc)
	ctx := context.Background()

	cur := testDevice(testDeviceID, "token-a", testNow.Add(-time.Hour))
	refreshed := testDevice(testDeviceID, "token-a", testNow)

	mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil)
	mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(cur, nil)
	mockDB.EXPECT().DeviceTouch(ctx, testDeviceID).Return(nil)
	mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(refreshed, nil)
	mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{refreshed}, nil)

	res, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != testDeviceID || !res.TMUpdate.Equal(testNow) {
		t.Errorf("Wrong match. the same device with a new refresh time is expected. got: %v", res)
	}
}

func Test_Register_duplicateRace(t *testing.T) {
	t.Run("the race is lost once and the second try refreshes", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockUtil := newTestHandler(mc)
		ctx := context.Background()
		refreshed := testDevice(testDeviceID, "token-a", testNow)

		gomock.InOrder(
			// try 1: nobody has the token, then the insert loses the race
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
			mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, dbhandler.ErrNotFound),
			mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())),
			mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(dbhandler.ErrDuplicate),
			// try 2: it starts again from the first step, so a racer with another owner is deleted too
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
			mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(refreshed, nil),
			mockDB.EXPECT().DeviceTouch(ctx, testDeviceID).Return(nil),
			mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(refreshed, nil),
			mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{refreshed}, nil),
		)

		if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a"); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("the race is lost every time", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockUtil := newTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil).Times(maxRegisterTries)
		mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, dbhandler.ErrNotFound).Times(maxRegisterTries)
		mockUtil.EXPECT().UUIDCreate().Return(testDeviceID).Times(maxRegisterTries)
		mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(dbhandler.ErrDuplicate).Times(maxRegisterTries)

		if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a"); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_Register_ownerChangedBetweenSteps(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTestHandler(mc)
	ctx := context.Background()

	// try 1: another owner inserted the token after step 1, so step 2 sees a foreign row. It is not touched
	// and not returned: the registration starts again and the second try finds nothing to refresh.
	foreign := testDevice(uuid.Must(uuid.NewV4()), "token-a", testNow)
	foreign.AgentID = uuid.Must(uuid.NewV4())
	mine := testDevice(testDeviceID, "token-a", testNow)

	gomock.InOrder(
		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
		mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(foreign, nil),
		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
		mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(mine, nil),
		mockDB.EXPECT().DeviceTouch(ctx, testDeviceID).Return(nil),
		mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(mine, nil),
		mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{mine}, nil),
	)

	res, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a")
	if err != nil || res != mine {
		t.Errorf("Wrong match. expect: %v, got: %v, %v", mine, res, err)
	}
}

func Test_Register_rowRemovedAfterRefresh(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil := newTestHandler(mc)
	ctx := context.Background()
	cur := testDevice(testDeviceID, "token-a", testNow.Add(-time.Hour))
	created := testDevice(uuid.Must(uuid.NewV4()), "token-a", testNow)

	gomock.InOrder(
		// try 1: refreshed, then another owner's registration removed the row before it was read back
		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
		mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(cur, nil),
		mockDB.EXPECT().DeviceTouch(ctx, testDeviceID).Return(nil),
		mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(nil, dbhandler.ErrNotFound),
		// try 2: nothing is left, so it inserts
		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil),
		mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, dbhandler.ErrNotFound),
		mockUtil.EXPECT().UUIDCreate().Return(created.ID),
		mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(nil),
		mockDB.EXPECT().DeviceGet(ctx, created.ID).Return(created, nil),
		mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{created}, nil),
	)

	res, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a")
	if err != nil || res != created {
		t.Errorf("Wrong match. expect: %v, got: %v, %v", created, res, err)
	}
}

func Test_Register_dbErrors(t *testing.T) {
	boom := fmt.Errorf("boom")

	tests := []struct {
		name  string
		setup func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockUtil *utilhandler.MockUtilHandler)
	}{
		{"delete of the previous owner", func(ctx context.Context, mockDB *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(boom)
		}},
		{"get by token", func(ctx context.Context, mockDB *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil)
			mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, boom)
		}},
		{"touch", func(ctx context.Context, mockDB *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil)
			mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(testDevice(testDeviceID, "token-a", testNow), nil)
			mockDB.EXPECT().DeviceTouch(ctx, testDeviceID).Return(boom)
		}},
		{"create", func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockUtil *utilhandler.MockUtilHandler) {
			mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-a", testCustomerID, testAgentID).Return(nil)
			mockDB.EXPECT().DeviceGetByToken(ctx, "token-a").Return(nil, dbhandler.ErrNotFound)
			mockUtil.EXPECT().UUIDCreate().Return(testDeviceID)
			mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(boom)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockUtil := newTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, mockDB, mockUtil)

			// a database error is not retried and is not a duplicate
			if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-a"); !errors.Is(err, boom) {
				t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
			}
		})
	}
}

func Test_Register_validation(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		agentID    uuid.UUID
		platform   device.Platform
		token      string
		reason     string
	}{
		{"no customer", uuid.Nil, testAgentID, device.PlatformAndroid, "t", "INVALID_OWNER"},
		{"no agent", testCustomerID, uuid.Nil, device.PlatformAndroid, "t", "INVALID_OWNER"},
		{"unknown platform", testCustomerID, testAgentID, device.Platform("ios"), "t", "INVALID_PLATFORM"},
		{"empty token", testCustomerID, testAgentID, device.PlatformAndroid, "", "INVALID_TOKEN"},
		{"token above the limit", testCustomerID, testAgentID, device.PlatformAndroid, strings.Repeat("a", device.MaxTokenLength+1), "INVALID_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no database call is expected
			h, _, _ := newTestHandler(mc)

			_, err := h.Register(context.Background(), tt.customerID, tt.agentID, tt.platform, tt.token)
			assertVoipbinError(t, err, tt.reason)
		})
	}

	t.Run("a token at the limit is accepted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockUtil := newTestHandler(mc)
		ctx := context.Background()
		token := strings.Repeat("a", device.MaxTokenLength)
		d := testDevice(testDeviceID, token, testNow)

		mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, token, testCustomerID, testAgentID).Return(nil)
		mockDB.EXPECT().DeviceGetByToken(ctx, token).Return(nil, dbhandler.ErrNotFound)
		mockUtil.EXPECT().UUIDCreate().Return(testDeviceID)
		mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(d, nil)
		mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return([]*device.Device{d}, nil)

		if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, token); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_Register_trimOldest(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil := newTestHandler(mc)
	ctx := context.Background()

	created := testDevice(testDeviceID, "token-new", testNow)

	// 12 devices: the two with the oldest refresh time are deleted, whatever the list order is
	list := []*device.Device{created}
	oldIDs := []uuid.UUID{}
	for i := 0; i < 11; i++ {
		id := uuid.Must(uuid.NewV4())
		d := testDevice(id, fmt.Sprintf("token-%d", i), testNow.Add(-time.Duration(i+1)*time.Hour))
		list = append(list, d)
		if i >= 9 {
			oldIDs = append(oldIDs, id)
		}
	}

	mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-new", testCustomerID, testAgentID).Return(nil)
	mockDB.EXPECT().DeviceGetByToken(ctx, "token-new").Return(nil, dbhandler.ErrNotFound)
	mockUtil.EXPECT().UUIDCreate().Return(testDeviceID)
	mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(created, nil)
	mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return(list, nil)
	for _, id := range oldIDs {
		mockDB.EXPECT().DeviceDelete(ctx, id).Return(nil)
	}

	if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-new"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Register_trimFailureIsNotAnError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil := newTestHandler(mc)
	ctx := context.Background()
	created := testDevice(testDeviceID, "token-new", testNow)

	mockDB.EXPECT().DeviceDeleteByTokenNotOwner(ctx, "token-new", testCustomerID, testAgentID).Return(nil)
	mockDB.EXPECT().DeviceGetByToken(ctx, "token-new").Return(nil, dbhandler.ErrNotFound)
	mockUtil.EXPECT().UUIDCreate().Return(testDeviceID)
	mockDB.EXPECT().DeviceCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().DeviceGet(ctx, testDeviceID).Return(created, nil)
	mockDB.EXPECT().DeviceList(ctx, uint64(1000), listPageToken, gomock.Any()).Return(nil, fmt.Errorf("boom"))

	if _, err := h.Register(ctx, testCustomerID, testAgentID, device.PlatformAndroid, "token-new"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Get_Delete(t *testing.T) {
	d := testDevice(testDeviceID, "token-a", testNow)

	t.Run("get", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(d, nil)
		res, err := h.Get(context.Background(), testDeviceID)
		if err != nil || res != d {
			t.Errorf("Wrong match. expect: %v, got: %v, %v", d, res, err)
		}
	})

	t.Run("get not found is typed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(nil, dbhandler.ErrNotFound)
		_, err := h.Get(context.Background(), testDeviceID)
		assertVoipbinError(t, err, "DEVICE_NOT_FOUND")
	})

	t.Run("get database error passes through", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		boom := fmt.Errorf("boom")
		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(nil, boom)
		if _, err := h.Get(context.Background(), testDeviceID); !errors.Is(err, boom) {
			t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
		}
	})

	t.Run("delete returns the deleted device", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(d, nil)
		mockDB.EXPECT().DeviceDelete(gomock.Any(), testDeviceID).Return(nil)
		res, err := h.Delete(context.Background(), testDeviceID)
		if err != nil || res != d {
			t.Errorf("Wrong match. expect: %v, got: %v, %v", d, res, err)
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(nil, dbhandler.ErrNotFound)
		_, err := h.Delete(context.Background(), testDeviceID)
		assertVoipbinError(t, err, "DEVICE_NOT_FOUND")
	})

	t.Run("delete database error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)

		boom := fmt.Errorf("boom")
		mockDB.EXPECT().DeviceGet(gomock.Any(), testDeviceID).Return(d, nil)
		mockDB.EXPECT().DeviceDelete(gomock.Any(), testDeviceID).Return(boom)
		if _, err := h.Delete(context.Background(), testDeviceID); !errors.Is(err, boom) {
			t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
		}
	})
}

func Test_List_dropsExpired(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTestHandler(mc)
	ctx := context.Background()

	live := testDevice(uuid.Must(uuid.NewV4()), "live", testNow.Add(-time.Hour))
	edge := testDevice(uuid.Must(uuid.NewV4()), "edge", testNow.Add(-device.Expiry))
	expired := testDevice(uuid.Must(uuid.NewV4()), "expired", testNow.Add(-device.Expiry-time.Second))

	filters := map[device.Field]any{device.FieldAgentID: testAgentID}
	mockDB.EXPECT().DeviceList(ctx, uint64(10), "token", filters).Return([]*device.Device{live, expired, edge}, nil)

	res, err := h.List(ctx, 10, "token", filters)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[0] != live || res[1] != edge {
		t.Errorf("Wrong match. expect: [live edge], got: %v", res)
	}

	boom := fmt.Errorf("boom")
	mockDB.EXPECT().DeviceList(ctx, uint64(10), "token", filters).Return(nil, boom)
	if _, err := h.List(ctx, 10, "token", filters); !errors.Is(err, boom) {
		t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
	}
}

func Test_Events(t *testing.T) {
	boom := fmt.Errorf("boom")

	t.Run("customer deleted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)
		ctx := context.Background()

		c := &cmcustomer.Customer{}
		c.ID = testCustomerID

		mockDB.EXPECT().DeviceDeleteByCustomerID(ctx, testCustomerID).Return(nil)
		if err := h.EventCustomerDeleted(ctx, c); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		mockDB.EXPECT().DeviceDeleteByCustomerID(ctx, testCustomerID).Return(boom)
		if err := h.EventCustomerDeleted(ctx, c); !errors.Is(err, boom) {
			t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
		}
	})

	t.Run("agent deleted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newTestHandler(mc)
		ctx := context.Background()

		a := &amagent.Agent{}
		a.ID = testAgentID

		mockDB.EXPECT().DeviceDeleteByAgentID(ctx, testAgentID).Return(nil)
		if err := h.EventAgentDeleted(ctx, a); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		mockDB.EXPECT().DeviceDeleteByAgentID(ctx, testAgentID).Return(boom)
		if err := h.EventAgentDeleted(ctx, a); !errors.Is(err, boom) {
			t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
		}
	})
}

func Test_NewDeviceHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	if h := NewDeviceHandler(dbhandler.NewMockDBHandler(mc)); h == nil {
		t.Errorf("Wrong match. expect: a handler, got: nil")
	}
}
