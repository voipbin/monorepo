package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConfbridgeParticipantClaim(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	id := uuid.FromStringOrNil("5b2d9c63-2222-11ef-b001-000000000001")
	other := uuid.FromStringOrNil("5b2d9c63-2222-11ef-b001-000000000002")
	key := "confbridge:participant_add:5b2d9c63-2222-11ef-b001-000000000001"

	ok, err := h.ConfbridgeParticipantClaim(ctx, id, 90*time.Second)
	if err != nil || !ok {
		t.Fatalf("first claim: expected true, got %v err %v", ok, err)
	}
	if ttl := mr.TTL(key); ttl != 90*time.Second {
		t.Errorf("ttl: expected 90s, got %v", ttl)
	}

	// a second request on the same confbridge is refused, another confbridge is independent
	ok, err = h.ConfbridgeParticipantClaim(ctx, id, 90*time.Second)
	if err != nil || ok {
		t.Errorf("second claim: expected false, got %v err %v", ok, err)
	}
	ok, err = h.ConfbridgeParticipantClaim(ctx, other, 90*time.Second)
	if err != nil || !ok {
		t.Errorf("other confbridge: expected true, got %v err %v", ok, err)
	}

	// the claim can be taken again after the ttl
	mr.FastForward(90*time.Second + time.Second)
	ok, err = h.ConfbridgeParticipantClaim(ctx, id, 90*time.Second)
	if err != nil || !ok {
		t.Errorf("after ttl: expected true, got %v err %v", ok, err)
	}

	// the claim can be taken again after the release; releasing a missing claim is not an error
	if errRelease := h.ConfbridgeParticipantRelease(ctx, id); errRelease != nil {
		t.Errorf("release: expected ok, got %v", errRelease)
	}
	if mr.Exists(key) {
		t.Errorf("release: the key still exists")
	}
	if errRelease := h.ConfbridgeParticipantRelease(ctx, id); errRelease != nil {
		t.Errorf("release of a missing claim: expected ok, got %v", errRelease)
	}
	ok, err = h.ConfbridgeParticipantClaim(ctx, id, 90*time.Second)
	if err != nil || !ok {
		t.Errorf("after release: expected true, got %v err %v", ok, err)
	}

	// a cache failure is reported, never reported as a granted claim
	mr.Close()
	ok, err = h.ConfbridgeParticipantClaim(ctx, id, time.Minute)
	if err == nil || ok {
		t.Errorf("closed server: expected false with an error, got %v err %v", ok, err)
	}
	if err := h.ConfbridgeParticipantRelease(ctx, id); err == nil {
		t.Errorf("closed server: expected a release error")
	}
}
