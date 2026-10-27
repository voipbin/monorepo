package cachehandler

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// failNthSet is a client hook that fails the n-th SET command and lets every
// other command through.
type failNthSet struct {
	n     int32
	count atomic.Int32
}

var errCov90SetRefused = errors.New("set refused")

func (f *failNthSet) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if strings.EqualFold(cmd.Name(), "set") && f.count.Add(1) == f.n {
		return ctx, errCov90SetRefused
	}
	return ctx, nil
}

func (f *failNthSet) AfterProcess(context.Context, redis.Cmder) error { return nil }

func (f *failNthSet) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (f *failNthSet) AfterProcessPipeline(context.Context, []redis.Cmder) error { return nil }

func Test_NewHandler(t *testing.T) {
	password := "test-password-0001"

	res := NewHandler("cache.example.com:6379", password, 3)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("expected a *handler, got: %T", res)
	}
	defer func() { _ = h.Cache.Close() }()

	if h.Addr != "cache.example.com:6379" || h.Password != password || h.DB != 3 || h.Cache == nil {
		t.Errorf("the constructor did not keep the connection settings: %+v", h)
	}
}

// Test_AIcallSet_PartialFailure verifies that a failing write of any of the three
// keys is reported, so a caller never believes a half-written entry is cached.
func Test_AIcallSet_PartialFailure(t *testing.T) {
	ctx := context.Background()
	data := &aicall.AIcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000a1")}}

	for _, failing := range []int32{1, 2, 3} {
		h, mr := newCoverageTestHandler(t)
		h.Cache.AddHook(&failNthSet{n: failing})

		err := h.AIcallSet(ctx, data)
		if !errors.Is(err, errCov90SetRefused) {
			t.Errorf("set #%d failing: expected the refused write to be reported, got: %v", failing, err)
		}

		// the writes before the refused one are in place, the ones after it are not
		if got := len(mr.Keys()); got != int(failing)-1 {
			t.Errorf("set #%d failing: expected %d keys written, got %d", failing, failing-1, got)
		}
	}
}

func Test_Set_ServerUnavailable(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)
	mr.Close()

	if err := h.AISet(ctx, &ai.AI{}); err == nil {
		t.Errorf("AISet: expected an error from an unavailable server, got nil")
	}
	if _, err := h.ListenConversationAIcallIDsGet(ctx, uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000a2")); err == nil {
		t.Errorf("ListenConversationAIcallIDsGet: expected an error from an unavailable server, got nil")
	}
}

func Test_ListenConversationAIcallIDsGet_SkipsMalformedMembers(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)

	conversationID := uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000b1")
	good := uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000b2")
	if _, err := mr.SAdd(listenConversationKey(conversationID), good.String(), "not-a-uuid"); err != nil {
		t.Fatalf("could not seed the set: %v", err)
	}

	res, err := h.ListenConversationAIcallIDsGet(ctx, conversationID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0] != good {
		t.Errorf("expected only the well-formed member, got: %v", res)
	}
}

// Test_ListenLists_WrongType verifies that a key holding the wrong type is
// reported as an error rather than as an empty buffer.
func Test_ListenLists_WrongType(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)

	aicallID := uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000c1")
	mr.Set(listenPendingKey(aicallID), "a string, not a list")
	mr.Set(listenWindowKey(aicallID), "a string, not a list")

	if res, err := h.ListenPendingPopAll(ctx, aicallID); err == nil || res != nil {
		t.Errorf("ListenPendingPopAll: expected an error and no lines, got: %v, %v", res, err)
	}
	if res, err := h.ListenWindowGet(ctx, aicallID); err == nil || res != nil {
		t.Errorf("ListenWindowGet: expected an error and no lines, got: %v, %v", res, err)
	}
}

func Test_ListenPendingPopAll_Empty(t *testing.T) {
	ctx := context.Background()
	h, _ := newCoverageTestHandler(t)

	res, err := h.ListenPendingPopAll(ctx, uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000d1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("expected an empty, non-nil slice, got: %#v", res)
	}
}
