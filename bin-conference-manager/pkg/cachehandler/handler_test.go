package cachehandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
)

const (
	testConferenceID     = "7c1d2e30-0001-11ee-8a11-000000000001"
	testCustomerID       = "7c1d2e30-0002-11ee-8a11-000000000002"
	testConferencecallID = "7c1d2e30-0003-11ee-8a11-000000000003"
	testReferenceID      = "7c1d2e30-0004-11ee-8a11-000000000004"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h := &handler{
		Addr: mr.Addr(),
		Cache: redis.NewClient(&redis.Options{
			Addr:        mr.Addr(),
			MaxRetries:  -1,
			DialTimeout: 200 * time.Millisecond,
		}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func newTestConference() *conference.Conference {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	return &conference.Conference{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(testConferenceID),
			CustomerID: uuid.FromStringOrNil(testCustomerID),
		},
		Type:              conference.TypeConference,
		Status:            conference.StatusProgressing,
		Name:              "test conference",
		Data:              map[string]any{"key": "value"},
		ConferencecallIDs: []uuid.UUID{uuid.FromStringOrNil(testConferencecallID)},
		TMCreate:          &tmCreate,
	}
}

func newTestConferencecall() *conferencecall.Conferencecall {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	return &conferencecall.Conferencecall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(testConferencecallID),
			CustomerID: uuid.FromStringOrNil(testCustomerID),
		},
		ConferenceID:  uuid.FromStringOrNil(testConferenceID),
		ReferenceType: conferencecall.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil(testReferenceID),
		Status:        conferencecall.StatusJoined,
		TMCreate:      &tmCreate,
	}
}

func Test_NewHandler(t *testing.T) {
	mr := miniredis.RunT(t)

	// the credential is held by the fake server only and is not a real one
	authValue := "synthetic-test-value"
	mr.RequireAuth(authValue)

	h := NewHandler(mr.Addr(), authValue, 3)
	hh, ok := h.(*handler)
	if !ok {
		t.Fatalf("Wrong type. expect: *handler, got: %T", h)
	}
	t.Cleanup(func() { _ = hh.Cache.Close() })

	if hh.Addr != mr.Addr() || hh.Password != authValue || hh.DB != 3 {
		t.Errorf("Wrong fields. addr: %s, db: %d", hh.Addr, hh.DB)
	}

	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	ctx := context.Background()
	cf := newTestConference()
	if err := h.ConferenceSet(ctx, cf); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "conference:conference:" + testConferenceID
	if !mr.DB(3).Exists(key) {
		t.Errorf("Wrong match. expect: key in db 3")
	}
	if mr.DB(0).Exists(key) {
		t.Errorf("Wrong match. expect: no key in db 0")
	}
}

func Test_NewHandler_wrongPassword(t *testing.T) {
	mr := miniredis.RunT(t)

	serverValue := "synthetic-server-value"
	mr.RequireAuth(serverValue)

	clientValue := "synthetic-other-value"
	h := NewHandler(mr.Addr(), clientValue, 0)
	if hh, ok := h.(*handler); ok {
		t.Cleanup(func() { _ = hh.Cache.Close() })
	}

	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: auth error, got: ok")
	}
}

func Test_Connect_serverClosed(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server is closed, got: ok")
	}
}

func Test_ConferenceSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	cf := newTestConference()
	if err := h.ConferenceSet(ctx, cf); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "conference:conference:" + testConferenceID
	raw, err := mr.Get(key)
	if err != nil {
		t.Fatalf("Wrong match. expect the stored key, got: %v", err)
	}
	var stored conference.Conference
	if errUnmarshal := json.Unmarshal([]byte(raw), &stored); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: json, got: %v", errUnmarshal)
	}
	if !reflect.DeepEqual(&stored, cf) {
		t.Errorf("Wrong stored value.\nexpect: %v\ngot: %v", cf, &stored)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong ttl. expect: %v, got: %v", 24*time.Hour, ttl)
	}

	res, err := h.ConferenceGet(ctx, cf.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, cf) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", cf, res)
	}

	// setting the same id again replaces the cached value
	updated := newTestConference()
	updated.Status = conference.StatusTerminating
	if err := h.ConferenceSet(ctx, updated); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	res, err = h.ConferenceGet(ctx, cf.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, updated) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
	}
}

func Test_ConferenceSet_marshalError(t *testing.T) {
	h, mr := newTestHandler(t)

	cf := newTestConference()
	cf.Data = map[string]any{"bad": make(chan int)}

	err := h.ConferenceSet(context.Background(), cf)
	var typeErr *json.UnsupportedTypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("Wrong match. expect: *json.UnsupportedTypeError, got: %v", err)
	}
	if mr.Exists("conference:conference:" + testConferenceID) {
		t.Errorf("Wrong match. expect: no key written on marshal error")
	}
}

func Test_ConferencecallSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	cc := newTestConferencecall()
	if err := h.ConferencecallSet(ctx, cc); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "conference:conferencecall:" + testConferencecallID
	raw, err := mr.Get(key)
	if err != nil {
		t.Fatalf("Wrong match. expect the stored key, got: %v", err)
	}
	var stored conferencecall.Conferencecall
	if errUnmarshal := json.Unmarshal([]byte(raw), &stored); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: json, got: %v", errUnmarshal)
	}
	if !reflect.DeepEqual(&stored, cc) {
		t.Errorf("Wrong stored value.\nexpect: %v\ngot: %v", cc, &stored)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong ttl. expect: %v, got: %v", 24*time.Hour, ttl)
	}

	res, err := h.ConferencecallGet(ctx, cc.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, cc) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", cc, res)
	}

	// setting the same id again replaces the cached value
	updated := newTestConferencecall()
	updated.Status = conferencecall.StatusLeaved
	if err := h.ConferencecallSet(ctx, updated); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	res, err = h.ConferencecallGet(ctx, cc.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, updated) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
	}
}

func Test_Get_notFound(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(testConferenceID)

	tests := []struct {
		name string
		get  func() (any, error)
	}{
		{"conference", func() (any, error) { return h.ConferenceGet(ctx, id) }},
		{"conferencecall", func() (any, error) { return h.ConferencecallGet(ctx, id) }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.get()
			if !errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
			}
			if !isNilPointer(res) {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_Get_invalidJSON(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(testConferenceID)

	tests := []struct {
		name string
		key  string
		get  func() (any, error)
	}{
		{"conference", "conference:conference:" + testConferenceID, func() (any, error) { return h.ConferenceGet(ctx, id) }},
		{"conferencecall", "conference:conferencecall:" + testConferenceID, func() (any, error) { return h.ConferencecallGet(ctx, id) }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if errSet := mr.Set(tt.key, "not-json"); errSet != nil {
				t.Fatalf("Could not seed the key. err: %v", errSet)
			}

			res, err := tt.get()
			var syntaxErr *json.SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Errorf("Wrong match. expect: *json.SyntaxError, got: %v", err)
			}
			if !isNilPointer(res) {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_serverClosed(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	mr.Close()

	if err := h.ConferenceSet(ctx, newTestConference()); err == nil {
		t.Errorf("Wrong match. expect: error on ConferenceSet, got: ok")
	}
	if err := h.ConferencecallSet(ctx, newTestConferencecall()); err == nil {
		t.Errorf("Wrong match. expect: error on ConferencecallSet, got: ok")
	}

	res, err := h.ConferenceGet(ctx, uuid.FromStringOrNil(testConferenceID))
	if err == nil || errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: connection error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func isNilPointer(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
