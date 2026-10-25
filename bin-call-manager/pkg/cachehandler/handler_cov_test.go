package cachehandler

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func newCovHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	h := NewHandler(mr.Addr(), "", 0).(*handler)
	t.Cleanup(func() { _ = h.Cache.Close() })
	return h, mr
}

func Test_setSerialize_getSerialize_cov(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000a1")

	tests := []struct {
		name string
		key  string
		set  func(h *handler) error
		get  func(h *handler) (interface{}, error)
		want interface{}
	}{
		{
			name: "channel",
			key:  "channel:ch-001",
			set:  func(h *handler) error { return h.ChannelSet(ctx, &channel.Channel{ID: "ch-001", AsteriskID: "ast-1"}) },
			get:  func(h *handler) (interface{}, error) { return h.ChannelGet(ctx, "ch-001") },
			want: &channel.Channel{ID: "ch-001", AsteriskID: "ast-1"},
		},
		{
			name: "bridge",
			key:  "bridge:br-001",
			set:  func(h *handler) error { return h.BridgeSet(ctx, &bridge.Bridge{ID: "br-001", Name: "n1"}) },
			get:  func(h *handler) (interface{}, error) { return h.BridgeGet(ctx, "br-001") },
			want: &bridge.Bridge{ID: "br-001", Name: "n1"},
		},
		{
			name: "call",
			key:  "call:00000000-0000-0000-0000-0000000000a1",
			set: func(h *handler) error {
				return h.CallSet(ctx, &call.Call{Identity: commonidentity.Identity{ID: id}, ChannelID: "ch-1"})
			},
			get:  func(h *handler) (interface{}, error) { return h.CallGet(ctx, id) },
			want: &call.Call{Identity: commonidentity.Identity{ID: id}, ChannelID: "ch-1"},
		},
		{
			name: "confbridge",
			key:  "confbridge:00000000-0000-0000-0000-0000000000a1",
			set: func(h *handler) error {
				return h.ConfbridgeSet(ctx, &confbridge.Confbridge{Identity: commonidentity.Identity{ID: id}, BridgeID: "br-9"})
			},
			get:  func(h *handler) (interface{}, error) { return h.ConfbridgeGet(ctx, id) },
			want: &confbridge.Confbridge{Identity: commonidentity.Identity{ID: id}, BridgeID: "br-9"},
		},
		{
			name: "recording",
			key:  "recording:00000000-0000-0000-0000-0000000000a1",
			set: func(h *handler) error {
				return h.RecordingSet(ctx, &recording.Recording{Identity: commonidentity.Identity{ID: id}, Filenames: []string{"a.wav"}})
			},
			get:  func(h *handler) (interface{}, error) { return h.RecordingGet(ctx, id) },
			want: &recording.Recording{Identity: commonidentity.Identity{ID: id}, Filenames: []string{"a.wav"}},
		},
		{
			name: "groupcall",
			key:  "call:groupcall:00000000-0000-0000-0000-0000000000a1",
			set: func(h *handler) error {
				return h.GroupcallSet(ctx, &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Source: nil})
			},
			get:  func(h *handler) (interface{}, error) { return h.GroupcallGet(ctx, id) },
			want: &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}},
		},
		{
			name: "call app amd",
			key:  "callapplication:amd:ch-77",
			set: func(h *handler) error {
				return h.CallAppAMDSet(ctx, "ch-77", &callapplication.AMD{CallID: id, MachineHandle: callapplication.AMDMachineHandleHangup, Async: true})
			},
			get:  func(h *handler) (interface{}, error) { return h.CallAppAMDGet(ctx, "ch-77") },
			want: &callapplication.AMD{CallID: id, MachineHandle: callapplication.AMDMachineHandleHangup, Async: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newCovHandler(t)

			// miss before set
			if _, err := tt.get(h); !errors.Is(err, redis.Nil) {
				t.Errorf("miss: expected redis.Nil, got %v", err)
			}

			if err := tt.set(h); err != nil {
				t.Fatalf("set failed: %v", err)
			}
			if !mr.Exists(tt.key) {
				t.Fatalf("key %q was not written", tt.key)
			}
			if ttl := mr.TTL(tt.key); ttl != 24*time.Hour {
				t.Errorf("ttl: expected 24h, got %v", ttl)
			}

			got, err := tt.get(h)
			if err != nil {
				t.Fatalf("get failed: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("round trip mismatch.\nexpect: %+v\ngot: %+v", tt.want, got)
			}

			// expired entry is a miss again
			mr.FastForward(24*time.Hour + time.Second)
			if _, err := tt.get(h); !errors.Is(err, redis.Nil) {
				t.Errorf("after ttl: expected redis.Nil, got %v", err)
			}

			// a stored value that is not JSON makes the getter fail
			if err := mr.Set(tt.key, "not-json"); err != nil {
				t.Fatalf("seed failed: %v", err)
			}
			if _, err := tt.get(h); err == nil || errors.Is(err, redis.Nil) {
				t.Errorf("invalid json: expected decode error, got %v", err)
			}

			// server failure propagates from the setter
			mr.Close()
			if err := tt.set(h); err == nil {
				t.Errorf("closed server: expected error from set")
			}
		})
	}
}

func Test_AsteriskAddressInternalGet_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)

	if _, err := h.AsteriskAddressInternalGet(ctx, "ast-1"); !errors.Is(err, redis.Nil) {
		t.Errorf("miss: expected redis.Nil, got %v", err)
	}

	if err := mr.Set("asterisk.ast-1.address-internal", "203.0.113.10"); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	got, err := h.AsteriskAddressInternalGet(ctx, "ast-1")
	if err != nil || got != "203.0.113.10" {
		t.Errorf("expected 203.0.113.10, got %q err %v", got, err)
	}
}

func Test_KamailioMetadataGet_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)

	mr.HSet("kamailio:sip-call-1", "from", "616263", "plain", "zz-not-hex")
	got, err := h.KamailioMetadataGet(ctx, "sip-call-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{"from": "abc", "plain": "zz-not-hex"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expect: %v, got: %v", want, got)
	}

	// unknown call id returns an empty map
	got, err = h.KamailioMetadataGet(ctx, "sip-call-none")
	if err != nil || len(got) != 0 {
		t.Errorf("expected empty map, got %v err %v", got, err)
	}

	mr.Close()
	if _, err := h.KamailioMetadataGet(ctx, "sip-call-1"); err == nil {
		t.Errorf("closed server: expected error")
	}
}

func Test_CallRecoveryClaim_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	id := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000b1")

	ok, err := h.CallRecoveryClaim(ctx, id, 2*time.Minute)
	if err != nil || !ok {
		t.Fatalf("first claim: expected true, got %v err %v", ok, err)
	}
	if ttl := mr.TTL("call:recovery:00000000-0000-0000-0000-0000000000b1"); ttl != 2*time.Minute {
		t.Errorf("ttl: expected 2m, got %v", ttl)
	}

	ok, err = h.CallRecoveryClaim(ctx, id, 2*time.Minute)
	if err != nil || ok {
		t.Errorf("second claim: expected false, got %v err %v", ok, err)
	}

	mr.FastForward(2*time.Minute + time.Second)
	ok, err = h.CallRecoveryClaim(ctx, id, 2*time.Minute)
	if err != nil || !ok {
		t.Errorf("after ttl: expected true, got %v err %v", ok, err)
	}

	mr.Close()
	if _, err := h.CallRecoveryClaim(ctx, id, time.Minute); err == nil {
		t.Errorf("closed server: expected error")
	}
}

func Test_ExternalMedia_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	id := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000c1")
	refID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000c2")
	em := &externalmedia.ExternalMedia{ID: id, ReferenceID: refID, LocalIP: "203.0.113.20"}

	if _, err := h.ExternalMediaGet(ctx, id); !errors.Is(err, redis.Nil) {
		t.Errorf("miss: expected redis.Nil, got %v", err)
	}
	if _, err := h.ExternalMediaGetByReferenceID(ctx, refID); !errors.Is(err, redis.Nil) {
		t.Errorf("miss by reference: expected redis.Nil, got %v", err)
	}
	// delete of an unknown id fails at the lookup
	if err := h.ExternalMediaDelete(ctx, id); !errors.Is(err, redis.Nil) {
		t.Errorf("delete unknown: expected redis.Nil, got %v", err)
	}

	if err := h.ExternalMediaSet(ctx, em); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	for _, key := range []string{
		"external_media:00000000-0000-0000-0000-0000000000c1",
		"external_media:reference_id:00000000-0000-0000-0000-0000000000c2",
	} {
		if !mr.Exists(key) {
			t.Errorf("key %q was not written", key)
		}
	}

	got, err := h.ExternalMediaGet(ctx, id)
	if err != nil || !reflect.DeepEqual(got, em) {
		t.Errorf("get: expect %+v, got %+v err %v", em, got, err)
	}
	got, err = h.ExternalMediaGetByReferenceID(ctx, refID)
	if err != nil || !reflect.DeepEqual(got, em) {
		t.Errorf("get by reference: expect %+v, got %+v err %v", em, got, err)
	}

	if err := h.ExternalMediaDelete(ctx, id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if mr.Exists("external_media:00000000-0000-0000-0000-0000000000c1") ||
		mr.Exists("external_media:reference_id:00000000-0000-0000-0000-0000000000c2") {
		t.Errorf("delete must remove both keys, left: %v", mr.Keys())
	}

	mr.Close()
	if err := h.ExternalMediaSet(ctx, em); err == nil {
		t.Errorf("closed server: expected error from set")
	}
}

func Test_ExternalMediaDelete_second_key_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	id := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000c1")
	refID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000c2")

	if err := h.ExternalMediaSet(ctx, &externalmedia.ExternalMedia{ID: id, ReferenceID: refID}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	// the first Del reaches the server, then the connection is lost: the
	// lookup succeeds, the deletes fail.
	mr.SetError("LOADING simulated")
	err := h.ExternalMediaDelete(ctx, id)
	if err == nil {
		t.Errorf("expected error while the server rejects commands")
	}
}

func Test_OutboundConfig_cov(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	customerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000d1")
	key := "outbound_config:00000000-0000-0000-0000-0000000000d1"
	cfg := &outboundconfig.OutboundConfig{ID: uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000d2"), CustomerID: customerID}

	if _, err := h.OutboundConfigGet(ctx, customerID); !errors.Is(err, redis.Nil) {
		t.Errorf("miss: expected redis.Nil, got %v", err)
	}

	if err := h.OutboundConfigSet(ctx, customerID, cfg); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if ttl := mr.TTL(key); ttl != 30*time.Minute {
		t.Errorf("ttl: expected 30m, got %v", ttl)
	}
	got, err := h.OutboundConfigGet(ctx, customerID)
	if err != nil || !reflect.DeepEqual(got, cfg) {
		t.Errorf("get: expect %+v, got %+v err %v", cfg, got, err)
	}

	if err := h.OutboundConfigDelete(ctx, customerID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if mr.Exists(key) {
		t.Errorf("delete must remove the key")
	}

	if err := h.OutboundConfigSetNotFound(ctx, customerID); err != nil {
		t.Fatalf("set not found failed: %v", err)
	}
	if ttl := mr.TTL(key); ttl != time.Minute {
		t.Errorf("sentinel ttl: expected 1m, got %v", ttl)
	}
	got, err = h.OutboundConfigGet(ctx, customerID)
	if got != nil || err != nil {
		t.Errorf("sentinel: expected (nil, nil), got %+v err %v", got, err)
	}
	mr.FastForward(time.Minute + time.Second)
	if _, err := h.OutboundConfigGet(ctx, customerID); !errors.Is(err, redis.Nil) {
		t.Errorf("sentinel after ttl: expected redis.Nil, got %v", err)
	}

	// corrupt value
	if err := mr.Set(key, "not-json"); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if got, err := h.OutboundConfigGet(ctx, customerID); got != nil || err == nil || errors.Is(err, redis.Nil) {
		t.Errorf("corrupt: expected decode error, got %+v err %v", got, err)
	}

	mr.Close()
	if err := h.OutboundConfigSet(ctx, customerID, cfg); err == nil {
		t.Errorf("closed server: expected error from set")
	}
	if err := h.OutboundConfigSetNotFound(ctx, customerID); err == nil {
		t.Errorf("closed server: expected error from set not found")
	}
	if err := h.OutboundConfigDelete(ctx, customerID); err == nil {
		t.Errorf("closed server: expected error from delete")
	}
}

func Test_NewHandler_Connect_cov(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mr := miniredis.RunT(t)
		h := NewHandler(mr.Addr(), "", 0)
		if err := h.Connect(); err != nil {
			t.Errorf("expected connect to succeed, got %v", err)
		}
	})

	t.Run("closed port", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen failed: %v", err)
		}
		addr := l.Addr().String()
		_ = l.Close()

		h := NewHandler(addr, "", 0)
		if err := h.Connect(); err == nil {
			t.Errorf("expected connect to fail")
		}
	})

	t.Run("fields", func(t *testing.T) {
		h := NewHandler("203.0.113.5:6379", "", 3).(*handler)
		if h.Addr != "203.0.113.5:6379" || h.DB != 3 || h.Cache == nil {
			t.Errorf("unexpected handler: %+v", h)
		}
	})
}
