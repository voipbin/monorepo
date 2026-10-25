package bridgehandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
)

func Test_Create_errors_cov(t *testing.T) {
	id := "6c1d2e3f-0002-11f0-8000-000000000002"

	tests := []struct {
		name      string
		createErr error
		getErr    error
		expectErr bool
	}{
		{name: "create error", createErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().BridgeCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				m.db.EXPECT().BridgeGet(ctx, id).Return(nil, tt.getErr)
			}

			res, err := h.Create(ctx, "ast-1", id, "name", bridge.TypeMixing, bridge.TechSimple, "", "", "", "", bridge.ReferenceTypeCall, uuid.Nil)
			if res != nil || err == nil {
				t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
			}
		})
	}
}

func Test_Get_notFound_cov(t *testing.T) {
	h, m := newCovHandler(t)
	id := "6c1d2e3f-0003-11f0-8000-000000000003"

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	m.db.EXPECT().BridgeGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

	res, err := h.Get(ctx, id)
	if res != nil || err == nil {
		t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
	}
	checkNoGetWithTimeoutGoroutine(t)
}

func Test_Delete_cov(t *testing.T) {
	id := "6c1d2e3f-0004-11f0-8000-000000000004"

	tests := []struct {
		name      string
		endErr    error
		getErr    error
		expectErr bool
	}{
		{name: "normal"},
		{name: "end error", endErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().BridgeEnd(ctx, id).Return(tt.endErr)
			if tt.endErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().BridgeGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().BridgeGet(ctx, id).Return(&bridge.Bridge{ID: id}, nil)
				}
			}

			res, err := h.Delete(ctx, id)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: (%v, %v)", tt.expectErr, res, err)
			}
		})
	}
}

func Test_AddRemoveChannelID_cov(t *testing.T) {
	id := "6c1d2e3f-0005-11f0-8000-000000000005"
	channelID := "6c1d2e3f-0006-11f0-8000-000000000006"

	calls := []struct {
		name string
		call func(h *bridgeHandler, ctx context.Context) (*bridge.Bridge, error)
		set  func(m *covMocks, ctx context.Context, err error)
	}{
		{
			name: "AddChannelID",
			call: func(h *bridgeHandler, ctx context.Context) (*bridge.Bridge, error) {
				return h.AddChannelID(ctx, id, channelID)
			},
			set: func(m *covMocks, ctx context.Context, err error) {
				m.db.EXPECT().BridgeAddChannelID(ctx, id, channelID).Return(err)
			},
		},
		{
			name: "RemoveChannelID",
			call: func(h *bridgeHandler, ctx context.Context) (*bridge.Bridge, error) {
				return h.RemoveChannelID(ctx, id, channelID)
			},
			set: func(m *covMocks, ctx context.Context, err error) {
				m.db.EXPECT().BridgeRemoveChannelID(ctx, id, channelID).Return(err)
			},
		},
	}

	cases := []struct {
		name      string
		setErr    error
		getErr    error
		expectErr bool
	}{
		{name: "normal"},
		{name: "db error", setErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
	}

	for _, c := range calls {
		for _, tt := range cases {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				h, m := newCovHandler(t)
				ctx := context.Background()

				c.set(m, ctx, tt.setErr)
				if tt.setErr == nil {
					if tt.getErr != nil {
						m.db.EXPECT().BridgeGet(ctx, id).Return(nil, tt.getErr)
					} else {
						m.db.EXPECT().BridgeGet(ctx, id).Return(&bridge.Bridge{ID: id}, nil)
					}
				}

				res, err := c.call(h, ctx)
				if (err != nil) != tt.expectErr {
					t.Fatalf("Wrong match. expect error: %v, got: (%v, %v)", tt.expectErr, res, err)
				}
			})
		}
	}
}

func Test_getWithTimeout_lateResult_cov(t *testing.T) {
	h, m := newCovHandler(t)
	id := "6c1d2e3f-0004-11f0-8000-000000000004"

	m.db.EXPECT().BridgeGet(gomock.Any(), id).DoAndReturn(func(c context.Context, _ string) (*bridge.Bridge, error) {
		<-c.Done()
		return &bridge.Bridge{}, nil
	}).Times(1)

	res, err := h.getWithTimeout(context.Background(), id, 20*time.Millisecond)
	if res != nil || err == nil {
		t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
	}
	checkNoGetWithTimeoutGoroutine(t)
}
