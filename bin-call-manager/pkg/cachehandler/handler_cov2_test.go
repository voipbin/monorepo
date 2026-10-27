package cachehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/externalmedia"
)

func Test_getSerialize_cov2_corruptValue(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	id := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000e1")

	if err := mr.Set("external_media:"+id.String(), "{not-json"); err != nil {
		t.Fatalf("could not seed: %v", err)
	}

	res, err := h.ExternalMediaGet(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: %v", res)
	}
}

func Test_ExternalMediaSet_cov2_serverError(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)

	mr.SetError("LOADING simulated")
	err := h.ExternalMediaSet(ctx, &externalmedia.ExternalMedia{ID: uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000e2")})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
