package cachehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

type i1Payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func Test_setSerialize(t *testing.T) {
	t.Run("success stores json with 24h ttl", func(t *testing.T) {
		h, mr := newTestHandler(t)

		in := &i1Payload{Name: "example", Count: 3}
		if err := h.setSerialize(context.Background(), "i1-key", in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := mr.Get("i1-key")
		if err != nil {
			t.Fatalf("could not read stored value: %v", err)
		}
		if expect := `{"name":"example","count":3}`; got != expect {
			t.Errorf("wrong stored value. expect: %s, got: %s", expect, got)
		}
		if got := mr.TTL("i1-key"); got != 24*time.Hour {
			t.Errorf("wrong TTL. expect: %v, got: %v", 24*time.Hour, got)
		}
	})

	t.Run("marshal failure is returned and nothing is stored", func(t *testing.T) {
		h, mr := newTestHandler(t)

		if err := h.setSerialize(context.Background(), "i1-key", make(chan int)); err == nil {
			t.Fatalf("expected marshal error, got nil")
		}
		if mr.Exists("i1-key") {
			t.Errorf("key must not be stored when marshal fails")
		}
	})

	t.Run("redis failure is returned", func(t *testing.T) {
		h, mr := newTestHandler(t)
		mr.Close()

		if err := h.setSerialize(context.Background(), "i1-key", &i1Payload{Name: "x"}); err == nil {
			t.Errorf("expected redis error, got nil")
		}
	})
}

func Test_getSerialize(t *testing.T) {
	t.Run("success populates the target", func(t *testing.T) {
		h, mr := newTestHandler(t)
		if err := mr.Set("i1-key", `{"name":"example","count":7}`); err != nil {
			t.Fatalf("could not seed: %v", err)
		}

		res := &i1Payload{}
		if err := h.getSerialize(context.Background(), "i1-key", res); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expect := &i1Payload{Name: "example", Count: 7}
		if !reflect.DeepEqual(res, expect) {
			t.Errorf("wrong result. expect: %v, got: %v", expect, res)
		}
	})

	t.Run("round trip with setSerialize", func(t *testing.T) {
		h, _ := newTestHandler(t)
		in := &i1Payload{Name: "round", Count: 11}
		if err := h.setSerialize(context.Background(), "i1-rt", in); err != nil {
			t.Fatalf("set: %v", err)
		}
		res := &i1Payload{}
		if err := h.getSerialize(context.Background(), "i1-rt", res); err != nil {
			t.Fatalf("get: %v", err)
		}
		if !reflect.DeepEqual(res, in) {
			t.Errorf("wrong result. expect: %v, got: %v", in, res)
		}
	})

	t.Run("missing key returns redis nil error", func(t *testing.T) {
		h, _ := newTestHandler(t)
		res := &i1Payload{}
		err := h.getSerialize(context.Background(), "i1-missing", res)
		if err == nil || err.Error() != "redis: nil" {
			t.Errorf("expected redis: nil, got: %v", err)
		}
		if !reflect.DeepEqual(res, &i1Payload{}) {
			t.Errorf("target must be untouched, got: %v", res)
		}
	})

	t.Run("non json value returns unmarshal error", func(t *testing.T) {
		h, mr := newTestHandler(t)
		if err := mr.Set("i1-key", "not-json"); err != nil {
			t.Fatalf("could not seed: %v", err)
		}
		res := &i1Payload{}
		err := h.getSerialize(context.Background(), "i1-key", res)
		if err == nil || err.Error() == "redis: nil" {
			t.Errorf("expected json unmarshal error, got: %v", err)
		}
		if !reflect.DeepEqual(res, &i1Payload{}) {
			t.Errorf("target must be untouched, got: %v", res)
		}
	})
}

func Test_ProvisioningTokenSet_RedisError(t *testing.T) {
	h, mr := newTestHandler(t)
	mr.Close()

	err := h.ProvisioningTokenSet(context.Background(), "tok", uuid.FromStringOrNil("6a1cca9e-80fb-11f0-92cd-2f8ae91ffc6e"), time.Minute)
	if err == nil {
		t.Errorf("expected redis error, got nil")
	}
}

func Test_ProvisioningTokenGet_RedisError(t *testing.T) {
	h, mr := newTestHandler(t)
	mr.Close()

	res, err := h.ProvisioningTokenGet(context.Background(), "tok")
	if err == nil {
		t.Errorf("expected redis error, got nil")
	}
	if res != uuid.Nil {
		t.Errorf("expected uuid.Nil, got: %v", res)
	}
}
