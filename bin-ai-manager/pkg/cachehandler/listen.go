package cachehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"
)

// Redis state backing the Insight AI's realtime call listening
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.2.4, §5.3,
// §5.4).
//
// These keys are deliberately NOT part of AIcallSet's snapshot-index scheme in
// handler.go. That scheme writes secondary keys pointing at a serialized entity
// and never invalidates a stale one when the indexed field changes; reusing it
// here would leave stale pointers and collide every non-listening AIcall on a
// shared nil-UUID key. These are purpose-built, explicitly-managed pointers
// with explicit lifecycles: written at listen start, removed at listen stop,
// and TTL'd as a backstop against a lost stop.
//
// Cache-loss behaviour is deliberate and stated: a Redis flush drops the
// resolver keys, so in-flight calls stop being listened to until the panel is
// reopened (which re-runs the listen-start path and repopulates). There is no
// DB fallback on a miss, because a DB fallback would put a query on the
// platform-wide transcript_created hot path -- exactly the cost this design
// removes.

// listenPendingPopMax bounds a single atomic drain of the pending buffer.
//
// go-redis v8's LPopCount requires a count argument, and the atomicity is the
// whole point: LLEN followed by a separate LPOP would reintroduce the race
// where a line pushed between the two calls is silently lost. 500 lines is far
// beyond any realistic debounce interval's worth of speech, so in practice this
// drains everything in one command.
const listenPendingPopMax = 500

func listenTranscribeKey(transcribeID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:transcribe:%s", transcribeID)
}

func listenPendingKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:pending:%s", aicallID)
}

func listenWindowKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:window:%s", aicallID)
}

func listenLockKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:lock:%s", aicallID)
}

func listenTurnsKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:turns:%s", aicallID)
}

func listenTurnPipecatcallIDKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:turnpcid:%s", aicallID)
}

func listenStartLockKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:startlock:%s", aicallID)
}

// listenConversationKey is the conversation -> listening AIcall ids resolver
// (design 2026-09-05 §5.2.2). Same shape and TTL discipline as
// listenTranscribeKey; keyed by conversation id because Message.CaseID is not
// populated on every write path, while Message.ConversationID always is.
func listenConversationKey(conversationID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:conversation:%s", conversationID)
}

// listenStartLockReleaseScript is the start lock's compare-and-delete.
//
// It MUST be one EVAL, not a GET followed by a DEL. A separate GET-then-DEL
// could observe our own token, then have this goroutine's TTL lapse and a
// second goroutine acquire the key, and then delete THAT goroutine's still-live
// lock -- which is precisely the clobbering the lock exists to prevent
// (design §5.2.2, review round 15 finding HIGH-1(b)).
const listenStartLockReleaseScript = `if redis.call("GET",KEYS[1])==ARGV[1] then return redis.call("DEL",KEYS[1]) else return 0 end`

// The four write scripts below all exist for ONE reason (review round 1
// finding MEDIUM-4): a write followed by a SEPARATE Expire is not atomic, and a
// failed Expire leaves the key with NO TTL at all. For the resolver set that
// defeats listenResolverTTL outright -- the entry, and with it this AIcall's
// subscription to a transcribe session's segments, would survive a lost cleanup
// forever rather than for the bounded twelve hours the TTL promises.
//
// EVAL rather than a pipeline, matching listenStartLockReleaseScript above --
// the one multi-command primitive this file already had. A TxPipeline would
// also be atomic, but mixing two idioms for the same guarantee in one file
// invites the next reader to assume the difference is meaningful.
//
// The first three return the EXPIRE result (1 on success), which callers
// ignore; what matters is that neither command can land without the other.
// listenIncrExpireScript is the one exception: its caller needs the counter's
// new value, so it returns the INCR result and discards EXPIRE's.
const (
	// listenSetAddExpireScript: SADD one member, then (re)arm the key's TTL.
	listenSetAddExpireScript = `redis.call("SADD",KEYS[1],ARGV[1]) return redis.call("EXPIRE",KEYS[1],ARGV[2])`

	// listenListPushExpireScript: RPUSH one line, then (re)arm the key's TTL.
	listenListPushExpireScript = `redis.call("RPUSH",KEYS[1],ARGV[1]) return redis.call("EXPIRE",KEYS[1],ARGV[2])`

	// listenListPushTrimExpireScript: RPUSH one line, trim the list back to a
	// bounded tail, then (re)arm the key's TTL. The trim joins the same script
	// rather than staying a third round trip -- an untrimmed window between two
	// commands is harmless, but there is no reason to keep the window open once
	// the atomic form is already being used here.
	listenListPushTrimExpireScript = `redis.call("RPUSH",KEYS[1],ARGV[1]) redis.call("LTRIM",KEYS[1],ARGV[3],-1) return redis.call("EXPIRE",KEYS[1],ARGV[2])`

	// listenIncrExpireScript: INCR the counter, then (re)arm the key's TTL,
	// returning the INCR result rather than EXPIRE's -- the turn cap needs the
	// count. Same atomicity requirement as the three above: a counter that lost
	// its TTL is a key nothing else ever reclaims, and it would keep an old
	// call's turn total alive against a later AIcall reusing the same id space.
	listenIncrExpireScript = `local n = redis.call("INCR",KEYS[1]) redis.call("EXPIRE",KEYS[1],ARGV[1]) return n`
)

// listenTTLSeconds renders a TTL for Redis's EXPIRE, which takes whole seconds.
//
// It floors at 1: EXPIRE with 0 (or a negative) DELETES the key immediately,
// so a sub-second TTL -- only reachable from a test or a misconfiguration --
// must never be allowed to turn a write into a delete.
func listenTTLSeconds(ttl time.Duration) int {
	if sec := int(ttl.Seconds()); sec > 0 {
		return sec
	}

	return 1
}

// ListenAIcallIDsGet returns every AIcall id currently listening to the given
// transcribe session.
//
// A SET, not a single value: N AIcalls can share one transcribe session (two
// Cases open on one call each get their own AIcall, and the second reuses the
// first's session). A single-valued key would let the second listener silently
// overwrite the first's mapping -- the first would stop receiving segments for
// the rest of the call, with no error and no metric.
//
// This is the ONE Redis round trip the platform-wide transcript_created hot
// path pays per final STT result. An empty result means "not a session we
// started" and is the overwhelmingly common outcome.
func (h *handler) ListenAIcallIDsGet(ctx context.Context, transcribeID uuid.UUID) ([]uuid.UUID, error) {
	tmp, err := h.Cache.SMembers(ctx, listenTranscribeKey(transcribeID)).Result()
	if err != nil {
		return nil, err
	}

	res := []uuid.UUID{}
	for _, m := range tmp {
		id := uuid.FromStringOrNil(m)
		if id == uuid.Nil {
			// A malformed member cannot address an AIcall; skip rather than
			// fail the whole resolution for the other listeners.
			continue
		}
		res = append(res, id)
	}

	return res, nil
}

// ListenAIcallIDAdd registers this AIcall as a listener on the transcribe
// session. Every listener adds only itself, so cleanup can remove only itself.
//
// The add and its TTL are ONE atomic EVAL -- see listenSetAddExpireScript for
// why a membership that outlives its TTL is the specific failure this prevents.
func (h *handler) ListenAIcallIDAdd(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenSetAddExpireScript,
		[]string{listenTranscribeKey(transcribeID)},
		aicallID.String(), listenTTLSeconds(ttl),
	).Err()
}

// ListenAIcallIDRemove removes only THIS AIcall's membership.
//
// Never DEL the key: another AIcall may still be listening to the same shared
// transcribe session, and deleting the whole key would cut it off silently.
// Redis removes the key itself once the set empties.
func (h *handler) ListenAIcallIDRemove(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID) error {
	return h.Cache.SRem(ctx, listenTranscribeKey(transcribeID), aicallID.String()).Err()
}

// ListenConversationAIcallIDsGet returns every AIcall id currently listening
// to the given conversation. A SET for the same reason as ListenAIcallIDsGet.
func (h *handler) ListenConversationAIcallIDsGet(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	tmp, err := h.Cache.SMembers(ctx, listenConversationKey(conversationID)).Result()
	if err != nil {
		return nil, err
	}

	res := []uuid.UUID{}
	for _, m := range tmp {
		id := uuid.FromStringOrNil(m)
		if id == uuid.Nil {
			continue
		}
		res = append(res, id)
	}

	return res, nil
}

// ListenConversationAIcallIDAdd registers aicallID as a listener of the
// conversation and (re)arms the key's TTL in one atomic EVAL.
func (h *handler) ListenConversationAIcallIDAdd(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenSetAddExpireScript,
		[]string{listenConversationKey(conversationID)},
		aicallID.String(), listenTTLSeconds(ttl),
	).Err()
}

// ListenConversationAIcallIDRemove drops one listener from the conversation's
// resolver set.
func (h *handler) ListenConversationAIcallIDRemove(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) error {
	return h.Cache.SRem(ctx, listenConversationKey(conversationID), aicallID.String()).Err()
}

// ListenConversationResolverTouch re-arms the resolver set's TTL without adding
// members; EXPIRE on a missing key is a no-op, so a membership a concurrent
// stop removed is never resurrected (design 2026-09-05 §5.2.2, code review
// round 4).
func (h *handler) ListenConversationResolverTouch(ctx context.Context, conversationID uuid.UUID, ttl time.Duration) error {
	return h.Cache.Expire(ctx, listenConversationKey(conversationID), time.Duration(listenTTLSeconds(ttl))*time.Second).Err()
}

// ListenConversationAIcallIDIsMember reports whether aicallID is registered as
// a listener of the conversation. Used by the conversation start's idempotency
// check (design 2026-09-05 §5.1.1 step 0).
func (h *handler) ListenConversationAIcallIDIsMember(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) (bool, error) {
	return h.Cache.SIsMember(ctx, listenConversationKey(conversationID), aicallID.String()).Result()
}

// ListenPendingLen returns the number of buffered, not-yet-evaluated lines.
// The conversation kind's RunListenTurn short-circuits on 0 BEFORE the Case RPC
// and BEFORE the turn counter, so a deferred flush that finds nothing costs
// neither (design 2026-09-05 §5.4).
func (h *handler) ListenPendingLen(ctx context.Context, aicallID uuid.UUID) (int64, error) {
	return h.Cache.LLen(ctx, listenPendingKey(aicallID)).Result()
}

// ListenPendingPush appends one transcript line to the not-yet-evaluated
// buffer.
//
// Push and TTL are ONE atomic EVAL, for the same reason ListenAIcallIDAdd's
// are: a buffer that lost its TTL is a key nothing will ever reclaim if its
// AIcall's stop path is missed.
func (h *handler) ListenPendingPush(ctx context.Context, aicallID uuid.UUID, line string, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenListPushExpireScript,
		[]string{listenPendingKey(aicallID)},
		line, listenTTLSeconds(ttl),
	).Err()
}

// ListenPendingPopAll atomically drains the pending buffer.
//
// One LPOP key count command (Redis >= 6.2), not LLEN followed by LPOP: the
// atomicity is what guarantees no concurrent appender's line is lost between a
// read and a trim. Returns an empty slice (not an error) when the buffer is
// empty -- redis.Nil is the normal "nothing there" signal for LPOP.
func (h *handler) ListenPendingPopAll(ctx context.Context, aicallID uuid.UUID) ([]string, error) {
	res, err := h.Cache.LPopCount(ctx, listenPendingKey(aicallID), listenPendingPopMax).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListenWindowPush appends one transcript line to the rolling window and trims
// it back to windowSize.
//
// A second list rather than a counter on the first: a line briefly present in
// the window but not yet popped from pending is harmless -- it is context
// either way.
//
// Push, trim and TTL are ONE atomic EVAL (listenListPushTrimExpireScript), so
// the window can neither grow past its bound nor lose its TTL because a later
// command in the sequence failed.
func (h *handler) ListenWindowPush(ctx context.Context, aicallID uuid.UUID, line string, windowSize int, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenListPushTrimExpireScript,
		[]string{listenWindowKey(aicallID)},
		line, listenTTLSeconds(ttl), -windowSize,
	).Err()
}

// ListenWindowGet returns the rolling window, oldest line first.
func (h *handler) ListenWindowGet(ctx context.Context, aicallID uuid.UUID) ([]string, error) {
	res, err := h.Cache.LRange(ctx, listenWindowKey(aicallID), 0, -1).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListenTurnTryLock reports whether this caller may run an evaluation turn now.
//
// SET NX EX: a leaky-bucket debounce, not a mutex. It works across replicas
// (both ai-manager pods share Redis), needs no timers and no per-AIcall
// goroutine, and self-heals on pod loss when the TTL expires. Losing the race
// is the normal case and is not an error -- the line stays buffered for the
// turn that did win.
func (h *handler) ListenTurnTryLock(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (bool, error) {
	return h.Cache.SetNX(ctx, listenLockKey(aicallID), "1", ttl).Result()
}

// ListenTurnCountIncr increments and returns this AIcall's evaluation-turn
// count. The hard cap it feeds is the backstop against a pathologically long
// call burning LLM spend indefinitely.
//
// The increment and its TTL are ONE atomic EVAL (listenIncrExpireScript), for
// the same reason every other write in this file is (review round 2 finding
// MEDIUM-3). This function was the last one left on the INCR-then-separate-
// EXPIRE idiom its four siblings were converted away from, and leaving one
// behind invited exactly the reading the scripts' own comment warns about --
// that the difference must mean something. It does not.
func (h *handler) ListenTurnCountIncr(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (int64, error) {
	res, err := h.Cache.Eval(ctx, listenIncrExpireScript,
		[]string{listenTurnsKey(aicallID)},
		listenTTLSeconds(ttl),
	).Int64()
	if err != nil {
		return 0, err
	}

	return res, nil
}

// ListenTurnPipecatcallIDAdd registers a pipecatcall id as a genuine listen
// evaluation turn, at the moment that id is minted.
//
// This is the POSITIVE signal ToolHandle needs. The tempting alternative --
// "this id is not the AIcall's currently-bound one, so it must be a listen
// turn" -- is wrong: an agent's own tool call can arrive after Send() has
// best-effort-interrupted its turn and rotated the bound id away, and would be
// indistinguishable from a listen turn. An id that was never SADD'd here is
// provably not a listen turn, whatever the AIcall row happens to say.
//
// Self-expiring: the entry only needs to outlive one turn, so it uses its own
// short TTL and needs no explicit cleanup -- which is exactly why the add and
// the TTL go out as ONE atomic EVAL. Nothing else ever deletes this key, so a
// membership that lost its TTL is a leak with no other reclaimer.
func (h *handler) ListenTurnPipecatcallIDAdd(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID, ttl time.Duration) error {
	return h.Cache.Eval(ctx, listenSetAddExpireScript,
		[]string{listenTurnPipecatcallIDKey(aicallID)},
		pipecatcallID.String(), listenTTLSeconds(ttl),
	).Err()
}

// ListenTurnPipecatcallIDIsMember reports whether the given pipecatcall id was
// registered as a listen evaluation turn for this AIcall.
func (h *handler) ListenTurnPipecatcallIDIsMember(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID) (bool, error) {
	return h.Cache.SIsMember(ctx, listenTurnPipecatcallIDKey(aicallID), pipecatcallID.String()).Result()
}

// ListenStartLockAcquire takes the per-AIcall listen-start lock (design §5.2.2).
//
// A thin SET NX EX wrapper -- the same Redis command ListenTurnTryLock above
// already uses -- but with a CALLER-SUPPLIED OWNERSHIP TOKEN rather than a
// constant value, and that difference is the whole point. The debounce lock's
// "anyone may release it" shape is safe only because stealing it merely delays
// a turn. Stealing THIS lock lets two goroutines clobber each other's DB and
// Redis state for a live, billed STT session, so it must be releasable only by
// the goroutine that took it.
//
// This function and ListenStartLockRelease are the lock's only two entry
// points, and they are the only two places the key format exists (design
// review round 16 finding LOW-6). Never call SetNX for this key from a handler.
//
// Returns false, nil when another goroutine already holds it. That is the
// normal, expected outcome of a second panel re-open during one long ring, not
// an error.
func (h *handler) ListenStartLockAcquire(ctx context.Context, aicallID uuid.UUID, token string, ttl time.Duration) (bool, error) {
	return h.Cache.SetNX(ctx, listenStartLockKey(aicallID), token, ttl).Result()
}

// ListenStartLockRelease releases the per-AIcall listen-start lock, but ONLY if
// this caller still holds it.
//
// Compare-and-delete against the caller's own token, atomically, in one EVAL
// (see listenStartLockReleaseScript). A token mismatch means this goroutine's
// TTL already lapsed and someone else legitimately acquired the key since --
// so the call is a deliberate NO-OP, not an error and not a delete. Deleting
// there would take a lock this goroutine no longer holds away from a goroutine
// that does.
//
// ALWAYS CALLED ON A CONTEXT DETACHED FROM THE ACQUIRING GOROUTINE'S OWN ctx
// (design §5.2.2, review round 16 finding MEDIUM-2). Acquire must respect the
// caller's deadline like any other RPC in the trigger path; Release deliberately
// must not, or the one case the TTL-vs-timeout margin exists for -- a goroutine
// reaching its own outer timeout while still finishing legitimate work -- is
// exactly the case where the release silently fails and strands the lock. The
// caller owns that detaching (the trigger path), not this function.
func (h *handler) ListenStartLockRelease(ctx context.Context, aicallID uuid.UUID, token string) error {
	return h.Cache.Eval(ctx, listenStartLockReleaseScript, []string{listenStartLockKey(aicallID)}, token).Err()
}

// ListenStateClear removes this AIcall's own per-AIcall listen keys.
//
// It deliberately does NOT touch ai:listen:transcribe:<id> -- that set may be
// shared with another listening AIcall, and only this AIcall's own membership
// may be removed (ListenAIcallIDRemove does that, separately, before this is
// called).
//
// It also deliberately does NOT delete ai:listen:turnpcid:<id>. Those entries
// are short-TTL and self-expiring, and leaving a stale one past a stop causes
// no incorrect behaviour: a tool call arriving late for an already-stopped
// listen turn still correctly resolves as a listen turn, which is exactly what
// it was.
func (h *handler) ListenStateClear(ctx context.Context, aicallID uuid.UUID) error {
	return h.Cache.Del(ctx,
		listenPendingKey(aicallID),
		listenWindowKey(aicallID),
		listenLockKey(aicallID),
		listenTurnsKey(aicallID),
	).Err()
}
