package channelhandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

var errCov = fmt.Errorf("cov error")

type covMocks struct {
	db    *dbhandler.MockDBHandler
	req   *requesthandler.MockRequestHandler
	cache *cachehandler.MockCacheHandler
}

func newCovHandler(t *testing.T) (*channelHandler, *covMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		db:    dbhandler.NewMockDBHandler(mc),
		req:   requesthandler.NewMockRequestHandler(mc),
		cache: cachehandler.NewMockCacheHandler(mc),
	}
	h := &channelHandler{
		reqHandler: m.req,
		db:         m.db,
		cache:      m.cache,
	}
	return h, m
}

func Test_GetChannelsForRecovery_cov(t *testing.T) {
	tests := []struct {
		name        string
		dbErr       error
		dbRes       []*channel.Channel
		expectError bool
	}{
		{name: "normal", dbRes: []*channel.Channel{{ID: "a1b2c3d4-0001-11f0-0000-000000000001"}}},
		{name: "db error", dbErr: errCov, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			now := time.Now()

			m.db.EXPECT().ChannelGetsForRecovery(ctx, "ast-1", channel.TypeCall, &now, &now, uint64(10)).Return(tt.dbRes, tt.dbErr)

			res, err := h.GetChannelsForRecovery(ctx, "ast-1", channel.TypeCall, &now, &now, 10)
			if (err != nil) != tt.expectError {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectError, err)
			}
			if !tt.expectError && len(res) != len(tt.dbRes) {
				t.Errorf("Wrong match. expect: %d, got: %d", len(tt.dbRes), len(res))
			}
		})
	}
}

func Test_Create_errors_cov(t *testing.T) {
	tests := []struct {
		name      string
		createErr error
		getErr    error
		healthErr error
		expectErr bool
	}{
		{name: "create error", createErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
		{name: "health error is ignored", healthErr: errCov, expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			id := "b2c3d4e5-0002-11f0-0000-000000000002"

			m.db.EXPECT().ChannelCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().ChannelGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().ChannelGet(ctx, id).Return(&channel.Channel{ID: id}, nil)
					m.req.EXPECT().CallV1ChannelHealth(ctx, id, defaultHealthDelay, 0).Return(tt.healthErr)
				}
			}

			res, err := h.Create(ctx, id, "ast-1", "PJSIP/test", channel.TypeCall, channel.TechPJSIP, "", "", "", "", ari.ChannelStateRing)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && (res == nil || res.ID != id) {
				t.Errorf("Wrong match. got: %v", res)
			}
		})
	}
}

func Test_Get_notFound_cov(t *testing.T) {
	h, m := newCovHandler(t)

	// the parent deadline is shorter than defaultExistTimeout, so Get gives up quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	m.db.EXPECT().ChannelGet(gomock.Any(), "c3d4e5f6-0003-11f0-0000-000000000003").Return(nil, errCov).AnyTimes()

	res, err := h.Get(ctx, "c3d4e5f6-0003-11f0-0000-000000000003")
	if res != nil || err == nil {
		t.Fatalf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong typed error. expect: *VoipbinError, got: %v", err)
	}
	if ve.Status != cerrors.StatusNotFound || ve.Reason != "CHANNEL_NOT_FOUND" {
		t.Errorf("Wrong typed error. expect: %v/CHANNEL_NOT_FOUND, got: %v/%v", cerrors.StatusNotFound, ve.Status, ve.Reason)
	}

	checkNoGetWithTimeoutGoroutine(t)
}

func Test_Delete_cov(t *testing.T) {
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
			id := "d4e5f6a7-0004-11f0-0000-000000000004"

			m.db.EXPECT().ChannelEndAndDelete(ctx, id, ari.ChannelCauseNormalClearing).Return(tt.endErr)
			if tt.endErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().ChannelGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().ChannelGet(ctx, id).Return(&channel.Channel{ID: id}, nil)
				}
			}

			res, err := h.Delete(ctx, id, ari.ChannelCauseNormalClearing)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res == nil {
				t.Errorf("Wrong match. expect: channel, got: nil")
			}

			// Hangup wraps Delete.
			if tt.endErr != nil {
				m.db.EXPECT().ChannelEndAndDelete(ctx, id, ari.ChannelCauseNormalClearing).Return(tt.endErr)
				if _, errHangup := h.Hangup(ctx, id, ari.ChannelCauseNormalClearing); errHangup == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_Hangup_cov(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	id := "e5f6a7b8-0005-11f0-0000-000000000005"

	m.db.EXPECT().ChannelEndAndDelete(ctx, id, ari.ChannelCauseNormalClearing).Return(nil)
	m.db.EXPECT().ChannelGet(ctx, id).Return(&channel.Channel{ID: id}, nil)

	res, err := h.Hangup(ctx, id, ari.ChannelCauseNormalClearing)
	if err != nil || res == nil || res.ID != id {
		t.Errorf("Wrong match. got: (%v, %v)", res, err)
	}
}

func Test_setters_dbError_cov(t *testing.T) {
	id := "f6a7b8c9-0006-11f0-0000-000000000006"

	tests := []struct {
		name string
		call func(h *channelHandler, ctx context.Context) error
		set  func(m *covMocks, ctx context.Context)
	}{
		{
			name: "SetDataItem",
			call: func(h *channelHandler, ctx context.Context) error { return h.SetDataItem(ctx, id, "k", "v") },
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetDataItem(ctx, id, "k", "v").Return(errCov)
			},
		},
		{
			name: "SetSIPTransport",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetSIPTransport(ctx, id, channel.SIPTransportUDP)
			},
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetSIPTransport(ctx, id, channel.SIPTransportUDP).Return(errCov)
			},
		},
		{
			name: "SetDirection",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetDirection(ctx, id, channel.DirectionIncoming)
			},
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetDirection(ctx, id, channel.DirectionIncoming).Return(errCov)
			},
		},
		{
			name: "SetType",
			call: func(h *channelHandler, ctx context.Context) error { return h.SetType(ctx, id, channel.TypeCall) },
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetType(ctx, id, channel.TypeCall).Return(errCov)
			},
		},
		{
			name: "setSIPCallID",
			call: func(h *channelHandler, ctx context.Context) error { return h.setSIPCallID(ctx, id, "sip-1") },
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetSIPCallID(ctx, id, "sip-1").Return(errCov)
			},
		},
		{
			name: "setSIPPai",
			call: func(h *channelHandler, ctx context.Context) error { return h.setSIPPai(ctx, id, "pai") },
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetDataItem(ctx, id, "sip_pai", "pai").Return(errCov)
			},
		},
		{
			name: "setSIPPrivacy",
			call: func(h *channelHandler, ctx context.Context) error { return h.setSIPPrivacy(ctx, id, "id") },
			set: func(m *covMocks, ctx context.Context) {
				m.db.EXPECT().ChannelSetDataItem(ctx, id, "sip_privacy", "id").Return(errCov)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			tt.set(m, ctx)

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_setters_ok_cov(t *testing.T) {
	id := "a7b8c9d0-0007-11f0-0000-000000000007"

	tests := []struct {
		name string
		call func(h *channelHandler, ctx context.Context) error
		set  func(m *covMocks, ctx context.Context, wg *sync.WaitGroup)
	}{
		{
			name: "SetDataItem",
			call: func(h *channelHandler, ctx context.Context) error { return h.SetDataItem(ctx, id, "k", "v") },
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				m.db.EXPECT().ChannelSetDataItem(ctx, id, "k", "v").Return(nil)
			},
		},
		{
			name: "SetType",
			call: func(h *channelHandler, ctx context.Context) error { return h.SetType(ctx, id, channel.TypeCall) },
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				m.db.EXPECT().ChannelSetType(ctx, id, channel.TypeCall).Return(nil)
			},
		},
		{
			name: "SetSIPTransport, follow-up lookup fails",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetSIPTransport(ctx, id, channel.SIPTransportUDP)
			},
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				wg.Add(1)
				m.db.EXPECT().ChannelSetSIPTransport(ctx, id, channel.SIPTransportUDP).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, id).DoAndReturn(func(context.Context, string) (*channel.Channel, error) {
					defer wg.Done()
					return nil, errCov
				})
			},
		},
		{
			name: "SetSIPTransport, follow-up lookup succeeds",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetSIPTransport(ctx, id, channel.SIPTransportUDP)
			},
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				wg.Add(1)
				m.db.EXPECT().ChannelSetSIPTransport(ctx, id, channel.SIPTransportUDP).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, id).DoAndReturn(func(context.Context, string) (*channel.Channel, error) {
					defer wg.Done()
					return &channel.Channel{ID: id, Direction: channel.DirectionIncoming, SIPTransport: channel.SIPTransportUDP}, nil
				})
			},
		},
		{
			name: "SetDirection, follow-up lookup fails",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetDirection(ctx, id, channel.DirectionIncoming)
			},
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				wg.Add(1)
				m.db.EXPECT().ChannelSetDirection(ctx, id, channel.DirectionIncoming).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, id).DoAndReturn(func(context.Context, string) (*channel.Channel, error) {
					defer wg.Done()
					return nil, errCov
				})
			},
		},
		{
			name: "SetDirection, follow-up lookup succeeds",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.SetDirection(ctx, id, channel.DirectionIncoming)
			},
			set: func(m *covMocks, ctx context.Context, wg *sync.WaitGroup) {
				wg.Add(1)
				m.db.EXPECT().ChannelSetDirection(ctx, id, channel.DirectionIncoming).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, id).DoAndReturn(func(context.Context, string) (*channel.Channel, error) {
					defer wg.Done()
					return &channel.Channel{ID: id, Direction: channel.DirectionIncoming, SIPTransport: channel.SIPTransportUDP}, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			var wg sync.WaitGroup
			tt.set(m, ctx, &wg)

			if err := tt.call(h, ctx); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
			wg.Wait()
		})
	}
}

func Test_UpdateState_cov(t *testing.T) {
	id := "b8c9d0e1-0008-11f0-0000-000000000008"

	tests := []struct {
		name      string
		state     ari.ChannelState
		setErr    error
		getErr    error
		expectErr bool
	}{
		{name: "up", state: ari.ChannelStateUp},
		{name: "ring", state: ari.ChannelStateRing},
		{name: "ringing", state: ari.ChannelStateRinging},
		{name: "unsupported state", state: ari.ChannelStateDown, expectErr: true},
		{name: "up db error", state: ari.ChannelStateUp, setErr: errCov, expectErr: true},
		{name: "ringing db error", state: ari.ChannelStateRinging, setErr: errCov, expectErr: true},
		{name: "get error", state: ari.ChannelStateUp, getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			switch tt.state {
			case ari.ChannelStateUp:
				m.db.EXPECT().ChannelSetStateAnswer(ctx, id, tt.state).Return(tt.setErr)
			case ari.ChannelStateRing, ari.ChannelStateRinging:
				m.db.EXPECT().ChannelSetStateRinging(ctx, id, tt.state).Return(tt.setErr)
			}
			if tt.setErr == nil && tt.state != ari.ChannelStateDown {
				if tt.getErr != nil {
					m.db.EXPECT().ChannelGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().ChannelGet(ctx, id).Return(&channel.Channel{ID: id, State: tt.state}, nil)
				}
			}

			res, err := h.UpdateState(ctx, id, tt.state)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res.State != tt.state {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.state, res.State)
			}
		})
	}
}

// Test_updaters_cov covers the "set, then get the updated channel" family.
func Test_updaters_cov(t *testing.T) {
	id := "c9d0e1f2-0009-11f0-0000-000000000009"

	tests := []struct {
		name   string
		call   func(h *channelHandler, ctx context.Context) (*channel.Channel, error)
		expect func(m *covMocks, ctx context.Context, setErr error)
	}{
		{
			name: "UpdateStasisName",
			call: func(h *channelHandler, ctx context.Context) (*channel.Channel, error) {
				return h.UpdateStasisName(ctx, id, "svc")
			},
			expect: func(m *covMocks, ctx context.Context, setErr error) {
				m.db.EXPECT().ChannelSetStasis(ctx, id, "svc").Return(setErr)
			},
		},
		{
			name: "UpdateBridgeID",
			call: func(h *channelHandler, ctx context.Context) (*channel.Channel, error) {
				return h.UpdateBridgeID(ctx, id, "br-1")
			},
			expect: func(m *covMocks, ctx context.Context, setErr error) {
				m.db.EXPECT().ChannelSetBridgeID(ctx, id, "br-1").Return(setErr)
			},
		},
		{
			name: "UpdatePlaybackID",
			call: func(h *channelHandler, ctx context.Context) (*channel.Channel, error) {
				return h.UpdatePlaybackID(ctx, id, "pb-1")
			},
			expect: func(m *covMocks, ctx context.Context, setErr error) {
				m.db.EXPECT().ChannelSetPlaybackID(ctx, id, "pb-1").Return(setErr)
			},
		},
		{
			name: "UpdateMuteDirection",
			call: func(h *channelHandler, ctx context.Context) (*channel.Channel, error) {
				return h.UpdateMuteDirection(ctx, id, channel.MuteDirectionBoth)
			},
			expect: func(m *covMocks, ctx context.Context, setErr error) {
				m.db.EXPECT().ChannelSetMuteDirection(ctx, id, channel.MuteDirectionBoth).Return(setErr)
			},
		},
		{
			name: "updateStasisInfo",
			call: func(h *channelHandler, ctx context.Context) (*channel.Channel, error) {
				return h.updateStasisInfo(ctx, id, channel.TypeCall, "svc", map[channel.StasisDataType]string{}, channel.DirectionIncoming)
			},
			expect: func(m *covMocks, ctx context.Context, setErr error) {
				m.db.EXPECT().ChannelSetStasisInfo(ctx, id, channel.TypeCall, "svc", map[channel.StasisDataType]string{}, channel.DirectionIncoming).Return(setErr)
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
		{name: "set error", setErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		for _, c := range cases {
			t.Run(tt.name+"/"+c.name, func(t *testing.T) {
				h, m := newCovHandler(t)
				ctx := context.Background()

				tt.expect(m, ctx, c.setErr)
				if c.setErr == nil {
					if c.getErr != nil {
						m.db.EXPECT().ChannelGet(ctx, id).Return(nil, c.getErr)
					} else {
						m.db.EXPECT().ChannelGet(ctx, id).Return(&channel.Channel{ID: id}, nil)
					}
				}

				res, err := tt.call(h, ctx)
				if (err != nil) != c.expectErr {
					t.Fatalf("Wrong match. expect error: %v, got: %v", c.expectErr, err)
				}
				if !c.expectErr && (res == nil || res.ID != id) {
					t.Errorf("Wrong match. got: %v", res)
				}
			})
		}
	}
}

func Test_UpdateSIPInfoByChannelVariable_cov(t *testing.T) {
	cn := &channel.Channel{ID: "d0e1f2a3-0010-11f0-0000-000000000010", AsteriskID: "ast-1"}
	meta := map[string]string{"k": "v"}

	tests := []struct {
		name string

		variable    string
		variableErr error
		cacheMeta   map[string]string
		cacheErr    error
		sipDataErr  error
		callIDErr   error
		getErr      error

		expectErr bool
	}{
		{name: "normal with metadata", variable: "sip-1", cacheMeta: meta},
		{name: "normal without metadata", variable: "sip-1"},
		{name: "cache error is ignored", variable: "sip-1", cacheErr: errCov},
		{name: "variable error, empty call id", variableErr: errCov},
		{name: "sip data error", variable: "sip-1", cacheMeta: meta, sipDataErr: errCov, expectErr: true},
		{name: "set call id error", variable: "sip-1", callIDErr: errCov, expectErr: true},
		{name: "get error", variable: "sip-1", getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.req.EXPECT().AstChannelVariableGet(ctx, cn.AsteriskID, cn.ID, "CHANNEL(pjsip,call-id)").Return(tt.variable, tt.variableErr)
			if tt.variable != "" {
				m.cache.EXPECT().KamailioMetadataGet(ctx, tt.variable).Return(tt.cacheMeta, tt.cacheErr)
				if len(tt.cacheMeta) > 0 {
					m.db.EXPECT().ChannelSetSIPData(ctx, cn.ID, tt.cacheMeta).Return(tt.sipDataErr)
				}
			}
			if tt.sipDataErr == nil {
				m.db.EXPECT().ChannelSetSIPCallID(ctx, cn.ID, tt.variable).Return(tt.callIDErr)
				if tt.callIDErr == nil {
					if tt.getErr != nil {
						m.db.EXPECT().ChannelGet(ctx, cn.ID).Return(nil, tt.getErr)
					} else {
						m.db.EXPECT().ChannelGet(ctx, cn.ID).Return(&channel.Channel{ID: cn.ID}, nil)
					}
				}
			}

			res, err := h.UpdateSIPInfoByChannelVariable(ctx, cn)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && (res == nil || res.ID != cn.ID) {
				t.Errorf("Wrong match. got: %v", res)
			}
		})
	}
}

func Test_UpdateSIPInfo_cov(t *testing.T) {
	id := "e1f2a3b4-0011-11f0-0000-000000000011"
	meta := map[string]string{"k": "v"}

	tests := []struct {
		name string

		cacheMeta  map[string]string
		cacheErr   error
		sipDataErr error
		callIDErr  error
		paiErr     error
		privacyErr error
		transErr   error
		getErr     error

		expectErr bool
	}{
		{name: "normal with metadata", cacheMeta: meta},
		{name: "cache error is ignored", cacheErr: errCov},
		{name: "sip data error", cacheMeta: meta, sipDataErr: errCov, expectErr: true},
		{name: "call id error", callIDErr: errCov, expectErr: true},
		{name: "pai error", paiErr: errCov, expectErr: true},
		{name: "privacy error", privacyErr: errCov, expectErr: true},
		{name: "transport error", transErr: errCov, expectErr: true},
		{name: "get error", getErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			var wg sync.WaitGroup

			m.cache.EXPECT().KamailioMetadataGet(ctx, "sip-1").Return(tt.cacheMeta, tt.cacheErr)
			if len(tt.cacheMeta) > 0 {
				m.db.EXPECT().ChannelSetSIPData(ctx, id, tt.cacheMeta).Return(tt.sipDataErr)
			}

			if tt.sipDataErr == nil {
				m.db.EXPECT().ChannelSetSIPCallID(ctx, id, "sip-1").Return(tt.callIDErr)
				if tt.callIDErr == nil {
					m.db.EXPECT().ChannelSetDataItem(ctx, id, "sip_pai", "pai").Return(tt.paiErr)
					if tt.paiErr == nil {
						m.db.EXPECT().ChannelSetDataItem(ctx, id, "sip_privacy", "none").Return(tt.privacyErr)
						if tt.privacyErr == nil {
							m.db.EXPECT().ChannelSetSIPTransport(ctx, id, channel.SIPTransportUDP).Return(tt.transErr)
							if tt.transErr == nil {
								// one lookup from the SetSIPTransport follow-up goroutine, one from the final get.
								wg.Add(2)
								m.db.EXPECT().ChannelGet(ctx, id).DoAndReturn(func(context.Context, string) (*channel.Channel, error) {
									defer wg.Done()
									if tt.getErr != nil {
										return nil, tt.getErr
									}
									return &channel.Channel{ID: id}, nil
								}).Times(2)
							}
						}
					}
				}
			}

			res, err := h.UpdateSIPInfo(ctx, id, "sip-1", "pai", "none", channel.SIPTransportUDP)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && (res == nil || res.ID != id) {
				t.Errorf("Wrong match. got: %v", res)
			}
			wg.Wait()
		})
	}
}
