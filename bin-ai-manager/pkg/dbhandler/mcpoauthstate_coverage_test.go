package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
)

func Test_McpOAuthState_Create_Get_Delete(t *testing.T) {
	resetTestDB(t)

	h := &handler{db: dbTest}
	ctx := context.Background()

	serverID := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000001")
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	expire := created.Add(10 * time.Minute)

	state := &mcpoauthstate.McpOAuthState{
		State:        "state-token-1",
		CustomerID:   uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001"),
		McpServerID:  &serverID,
		Vendor:       "vendor-a",
		PKCEVerifier: "verifier-1",
		TMCreate:     &created,
		TMExpire:     &expire,
	}
	if err := h.McpOAuthStateCreate(ctx, state); err != nil {
		t.Fatalf("create failed. err: %v", err)
	}

	got, err := h.McpOAuthStateGet(ctx, "state-token-1")
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if got.State != state.State || got.CustomerID != state.CustomerID || got.Vendor != "vendor-a" || got.PKCEVerifier != "verifier-1" {
		t.Errorf("row mismatch. got: %+v", got)
	}
	if got.McpServerID == nil || *got.McpServerID != serverID {
		t.Errorf("mcp server id mismatch. got: %v", got.McpServerID)
	}
	if got.TMExpire == nil || !got.TMExpire.Equal(expire) {
		t.Errorf("tm_expire mismatch. got: %v", got.TMExpire)
	}

	// the state is the primary key: the same token cannot be created twice.
	if errDup := h.McpOAuthStateCreate(ctx, state); errDup == nil {
		t.Errorf("expected a duplicate state token to be rejected")
	}

	// an unrelated row that must survive the delete of another token.
	bystander := *state
	bystander.State = "state-token-2"
	if errBystander := h.McpOAuthStateCreate(ctx, &bystander); errBystander != nil {
		t.Fatalf("bystander create failed. err: %v", errBystander)
	}

	if errDelete := h.McpOAuthStateDelete(ctx, "state-token-1"); errDelete != nil {
		t.Fatalf("delete failed. err: %v", errDelete)
	}
	if _, errGet := h.McpOAuthStateGet(ctx, "state-token-2"); errGet != nil {
		t.Errorf("deleting one token must not remove another, got: %v", errGet)
	}
	if _, errGet := h.McpOAuthStateGet(ctx, "state-token-1"); !errors.Is(errGet, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", errGet)
	}

	// deleting a token that does not exist is not an error.
	if errDelete := h.McpOAuthStateDelete(ctx, "state-token-1"); errDelete != nil {
		t.Errorf("deleting an absent token must be a no-op. err: %v", errDelete)
	}
}

func Test_McpOAuthStateDeleteExpired(t *testing.T) {
	resetTestDB(t)

	h := &handler{db: dbTest}
	ctx := context.Background()

	base := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")

	// three rows: already expired, expiring exactly at the cutoff, still valid.
	seed := []struct {
		state  string
		expire time.Time
	}{
		{"expired", base.Add(-time.Minute)},
		{"at-cutoff", base},
		{"valid", base.Add(time.Minute)},
	}
	for _, s := range seed {
		created := s.expire.Add(-10 * time.Minute)
		expire := s.expire
		if err := h.McpOAuthStateCreate(ctx, &mcpoauthstate.McpOAuthState{
			State:        s.state,
			CustomerID:   customerID,
			Vendor:       "vendor-a",
			PKCEVerifier: "verifier",
			TMCreate:     &created,
			TMExpire:     &expire,
		}); err != nil {
			t.Fatalf("seed %s failed. err: %v", s.state, err)
		}
	}

	// the cutoff is strictly "before": the row expiring exactly at it survives.
	// sqlite compares the stored text, so the cutoff uses the layout the sqlite
	// driver writes for a time value; a MySQL DATETIME compares by value.
	n, err := h.McpOAuthStateDeleteExpired(ctx, base.Format("2006-01-02 15:04:05.999999999-07:00"))
	if err != nil {
		t.Fatalf("sweep failed. err: %v", err)
	}
	if n != 1 {
		t.Errorf("expected exactly one expired row to be removed, got: %d", n)
	}

	if _, errGet := h.McpOAuthStateGet(ctx, "expired"); !errors.Is(errGet, ErrNotFound) {
		t.Errorf("the expired row must be gone, got: %v", errGet)
	}
	for _, keep := range []string{"at-cutoff", "valid"} {
		if _, errGet := h.McpOAuthStateGet(ctx, keep); errGet != nil {
			t.Errorf("row %s must survive the sweep, got: %v", keep, errGet)
		}
	}
}
