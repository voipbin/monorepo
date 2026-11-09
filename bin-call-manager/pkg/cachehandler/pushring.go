package cachehandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-call-manager/models/pushring"
)

// push ring session keys.
//
//	call:pushring:<session id>            hash   the session. fields: see the const list below.
//	call:pushring:cb:<confbridge id>      string session id
//	call:pushring:master:<call id>        string session id
//	call:pushring:flow:<flow id>          string session id (the temporary flow of the legs and one flow per ticket call)
//	call:pushticket:<sha256 of ticket>    hash   session_id, device_id, state, in_use_call_id, in_use_until
//
// Every state change of a session is one Lua script, so the state, the lease and the winner are always checked and
// changed together. Times are unix milliseconds passed by the caller.
const (
	pushRingKeyPrefix       = "call:pushring:"
	pushRingTicketKeyPrefix = "call:pushticket:"

	pushRingFieldCustomerID    = "customer_id"
	pushRingFieldAgentID       = "agent_id"
	pushRingFieldExtensionID   = "extension_id"
	pushRingFieldConfbridgeID  = "confbridge_id"
	pushRingFieldMasterCallID  = "master_call_id"
	pushRingFieldFlowID        = "flow_id"
	pushRingFieldExpiresAt     = "expires_at"
	pushRingFieldState         = "state"
	pushRingFieldLeaseOwner    = "lease_owner"
	pushRingFieldLeaseUntil    = "lease_until"
	pushRingFieldWinner        = "winner"
	pushRingFieldCallerJoined  = "caller_joined"
	pushRingFieldSIPRegistered = "sip_registered"

	// pushRingFieldTicketPrefix is "t:<ticket hash>" -> device id.
	pushRingFieldTicketPrefix = "t:"
	// pushRingFieldTicketFlowPrefix is "tf:<flow id of the ticket call>" -> ticket hash.
	pushRingFieldTicketFlowPrefix = "tf:"
)

func pushRingSessionKey(id uuid.UUID) string { return pushRingKeyPrefix + id.String() }

func pushRingIndexKey(kind string, id uuid.UUID) string {
	return fmt.Sprintf("%s%s:%s", pushRingKeyPrefix, kind, id)
}

func pushRingTicketKey(hash string) string { return pushRingTicketKeyPrefix + hash }

// pushRingLeaseTake takes the join lease. ARGV: call id, now, lease until.
var pushRingLeaseTake = redis.NewScript(`
local k = KEYS[1]
local st = redis.call('HGET', k, 'state')
if st ~= 'ringing' and st ~= 'push_failed' then return 0 end
local w = redis.call('HGET', k, 'winner')
if w and w ~= '' then return 0 end
local now = tonumber(ARGV[2])
local exp = tonumber(redis.call('HGET', k, 'expires_at'))
if not exp or now >= exp then return 0 end
local owner = redis.call('HGET', k, 'lease_owner')
local untilMs = tonumber(redis.call('HGET', k, 'lease_until')) or 0
if owner and owner ~= '' and owner ~= ARGV[1] and untilMs > now then return 0 end
redis.call('HSET', k, 'lease_owner', ARGV[1], 'lease_until', ARGV[3])
return 1
`)

// pushRingLeaseRelease releases the lease of the given call. ARGV: call id.
var pushRingLeaseRelease = redis.NewScript(`
local k = KEYS[1]
local owner = redis.call('HGET', k, 'lease_owner')
if not owner or owner ~= ARGV[1] then return 0 end
local w = redis.call('HGET', k, 'winner')
if w and w ~= '' then return 0 end
redis.call('HSET', k, 'lease_owner', '', 'lease_until', '0')
return 1
`)

// pushRingConfirm sets the winner. The lease owner confirms; a call whose lease expired takes the lease again with the
// same checks. ARGV: call id, now, winner.
var pushRingConfirm = redis.NewScript(`
local k = KEYS[1]
local st = redis.call('HGET', k, 'state')
if st ~= 'ringing' and st ~= 'push_failed' then return 0 end
local w = redis.call('HGET', k, 'winner')
if w and w ~= '' then return 0 end
local now = tonumber(ARGV[2])
local exp = tonumber(redis.call('HGET', k, 'expires_at'))
if not exp or now >= exp then return 0 end
local owner = redis.call('HGET', k, 'lease_owner')
local untilMs = tonumber(redis.call('HGET', k, 'lease_until')) or 0
if owner and owner ~= '' and owner ~= ARGV[1] and untilMs > now then return 0 end
redis.call('HSET', k, 'lease_owner', ARGV[1], 'winner', ARGV[3])
return 1
`)

// pushRingCallerJoined marks the caller as joined while the gate is open.
var pushRingCallerJoined = redis.NewScript(`
local k = KEYS[1]
local st = redis.call('HGET', k, 'state')
if st ~= 'ringing' and st ~= 'push_failed' then return 0 end
redis.call('HSET', k, 'caller_joined', '1')
return 1
`)

// pushRingFailed moves a ringing session to push_failed.
var pushRingFailed = redis.NewScript(`
local k = KEYS[1]
if redis.call('HGET', k, 'state') ~= 'ringing' then return 0 end
redis.call('HSET', k, 'state', 'push_failed')
return 1
`)

// pushRingClose closes the session once and returns the session as it was closed.
var pushRingClose = redis.NewScript(`
local k = KEYS[1]
local st = redis.call('HGET', k, 'state')
if not st or st == 'closed' then return {} end
redis.call('HSET', k, 'state', 'closed')
return redis.call('HGETALL', k)
`)

// pushRingTicketClaim marks a ticket in use. It returns -1 when the ticket does not exist, 0 when it is refused.
// ARGV: now, in use until.
var pushRingTicketClaim = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return -1 end
local k = KEYS[2]
if redis.call('HGET', k, 'state') ~= 'ringing' then return 0 end
local w = redis.call('HGET', k, 'winner')
if w and w ~= '' then return 0 end
if redis.call('HGET', k, 'caller_joined') ~= '1' then return 0 end
local now = tonumber(ARGV[1])
local exp = tonumber(redis.call('HGET', k, 'expires_at'))
if not exp or now >= exp then return 0 end
local ts = redis.call('HGET', KEYS[1], 'state')
if ts == 'in_use' then
  local u = tonumber(redis.call('HGET', KEYS[1], 'in_use_until')) or 0
  if u > now then return 0 end
elseif ts ~= 'issued' then
  return 0
end
redis.call('HSET', KEYS[1], 'state', 'in_use', 'in_use_until', ARGV[2], 'in_use_flow_id', '')
return 1
`)

// pushRingTicketReturn returns the in use ticket of the call that runs the given flow to issued while no winner exists.
// ARGV: flow id, ticket key prefix.
var pushRingTicketReturn = redis.NewScript(`
local h = redis.call('HGET', KEYS[1], 'tf:' .. ARGV[1])
if not h then return 0 end
local w = redis.call('HGET', KEYS[1], 'winner')
if w and w ~= '' then return 0 end
local tk = ARGV[2] .. h
if redis.call('HGET', tk, 'state') == 'in_use' and redis.call('HGET', tk, 'in_use_flow_id') == ARGV[1] then
  redis.call('HSET', tk, 'state', 'issued', 'in_use_flow_id', '', 'in_use_until', '0')
  return 1
end
return 0
`)

func unixMilli(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.FormatInt(t.UnixMilli(), 10)
}

// PushRingSessionCreate creates the session, its lookup keys and one ticket per entry of s.Tickets.
func (h *handler) PushRingSessionCreate(ctx context.Context, s *pushring.Session) error {
	key := pushRingSessionKey(s.ID)

	values := map[string]interface{}{
		pushRingFieldCustomerID:    s.CustomerID.String(),
		pushRingFieldAgentID:       s.AgentID.String(),
		pushRingFieldExtensionID:   s.ExtensionID.String(),
		pushRingFieldConfbridgeID:  s.ConfbridgeID.String(),
		pushRingFieldMasterCallID:  s.MasterCallID.String(),
		pushRingFieldFlowID:        s.FlowID.String(),
		pushRingFieldExpiresAt:     unixMilli(s.ExpiresAt),
		pushRingFieldState:         string(s.State),
		pushRingFieldLeaseOwner:    "",
		pushRingFieldLeaseUntil:    "0",
		pushRingFieldWinner:        "",
		pushRingFieldCallerJoined:  "0",
		pushRingFieldSIPRegistered: "0",
	}
	if s.SIPRegistered {
		values[pushRingFieldSIPRegistered] = "1"
	}
	for hash, deviceID := range s.Tickets {
		values[pushRingFieldTicketPrefix+hash] = deviceID.String()
	}

	pipe := h.Cache.TxPipeline()
	pipe.HSet(ctx, key, values)
	pipe.Expire(ctx, key, pushring.KeyTTL)
	pipe.Set(ctx, pushRingIndexKey(pushring.IndexConfbridge, s.ConfbridgeID), s.ID.String(), pushring.KeyTTL)
	pipe.Set(ctx, pushRingIndexKey(pushring.IndexMaster, s.MasterCallID), s.ID.String(), pushring.KeyTTL)
	pipe.Set(ctx, pushRingIndexKey(pushring.IndexFlow, s.FlowID), s.ID.String(), pushring.KeyTTL)
	for hash, deviceID := range s.Tickets {
		tk := pushRingTicketKey(hash)
		pipe.HSet(ctx, tk, map[string]interface{}{
			"session_id":     s.ID.String(),
			"device_id":      deviceID.String(),
			"state":          string(pushring.TicketStateIssued),
			"in_use_flow_id": "",
			"in_use_until":   "0",
		})
		pipe.Expire(ctx, tk, pushring.KeyTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(err, "could not create the push ring session")
	}

	return nil
}

// PushRingSessionGet returns the session.
func (h *handler) PushRingSessionGet(ctx context.Context, id uuid.UUID) (*pushring.Session, error) {
	m, err := h.Cache.HGetAll(ctx, pushRingSessionKey(id)).Result()
	if err != nil {
		return nil, err
	}

	return parsePushRingSession(id, m)
}

func parsePushRingSession(id uuid.UUID, m map[string]string) (*pushring.Session, error) {
	if len(m) == 0 {
		return nil, pushring.ErrSessionNotFound
	}

	res := &pushring.Session{
		ID:            id,
		CustomerID:    uuid.FromStringOrNil(m[pushRingFieldCustomerID]),
		AgentID:       uuid.FromStringOrNil(m[pushRingFieldAgentID]),
		ExtensionID:   uuid.FromStringOrNil(m[pushRingFieldExtensionID]),
		ConfbridgeID:  uuid.FromStringOrNil(m[pushRingFieldConfbridgeID]),
		MasterCallID:  uuid.FromStringOrNil(m[pushRingFieldMasterCallID]),
		FlowID:        uuid.FromStringOrNil(m[pushRingFieldFlowID]),
		State:         pushring.State(m[pushRingFieldState]),
		LeaseOwner:    uuid.FromStringOrNil(m[pushRingFieldLeaseOwner]),
		Winner:        m[pushRingFieldWinner],
		CallerJoined:  m[pushRingFieldCallerJoined] == "1",
		SIPRegistered: m[pushRingFieldSIPRegistered] == "1",
		Tickets:       map[string]uuid.UUID{},
		TicketFlows:   map[uuid.UUID]string{},
	}
	if ms, err := strconv.ParseInt(m[pushRingFieldExpiresAt], 10, 64); err == nil {
		res.ExpiresAt = time.UnixMilli(ms)
	}
	if ms, err := strconv.ParseInt(m[pushRingFieldLeaseUntil], 10, 64); err == nil && ms > 0 {
		res.LeaseUntil = time.UnixMilli(ms)
	}
	for field, value := range m {
		if strings.HasPrefix(field, pushRingFieldTicketPrefix) {
			res.Tickets[strings.TrimPrefix(field, pushRingFieldTicketPrefix)] = uuid.FromStringOrNil(value)
		}
		if strings.HasPrefix(field, pushRingFieldTicketFlowPrefix) {
			res.TicketFlows[uuid.FromStringOrNil(strings.TrimPrefix(field, pushRingFieldTicketFlowPrefix))] = value
		}
	}

	return res, nil
}

// PushRingSessionIDGet returns the id of the session found through the lookup key of the given kind
// (pushring.IndexConfbridge, IndexMaster or IndexCall). A missing key is pushring.ErrSessionNotFound.
func (h *handler) PushRingSessionIDGet(ctx context.Context, kind string, id uuid.UUID) (uuid.UUID, error) {
	tmp, err := h.Cache.Get(ctx, pushRingIndexKey(kind, id)).Result()
	if err == redis.Nil {
		return uuid.Nil, pushring.ErrSessionNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.FromStringOrNil(tmp), nil
}

// PushRingSessionIDsGet returns the session ids found through the master call index and the flow index in one
// round trip. A missing key gives uuid.Nil.
func (h *handler) PushRingSessionIDsGet(ctx context.Context, masterCallID, flowID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	res, err := h.Cache.MGet(ctx, pushRingIndexKey(pushring.IndexMaster, masterCallID), pushRingIndexKey(pushring.IndexFlow, flowID)).Result()
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	var byMaster, byFlow uuid.UUID
	if v, ok := res[0].(string); ok {
		byMaster = uuid.FromStringOrNil(v)
	}
	if v, ok := res[1].(string); ok {
		byFlow = uuid.FromStringOrNil(v)
	}

	return byMaster, byFlow, nil
}

// PushRingLeaseTake takes the join lease for the call.
func (h *handler) PushRingLeaseTake(ctx context.Context, sessionID, callID uuid.UUID, now, until time.Time) (bool, error) {
	res, err := pushRingLeaseTake.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}, callID.String(), unixMilli(now), unixMilli(until)).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

// PushRingLeaseRelease releases the lease the call holds.
func (h *handler) PushRingLeaseRelease(ctx context.Context, sessionID, callID uuid.UUID) error {
	return pushRingLeaseRelease.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}, callID.String()).Err()
}

// PushRingConfirm sets the winner. It returns false when the call may not win.
func (h *handler) PushRingConfirm(ctx context.Context, sessionID, callID uuid.UUID, now time.Time, winner string) (bool, error) {
	res, err := pushRingConfirm.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}, callID.String(), unixMilli(now), winner).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

// PushRingCallerJoined marks the caller of the session as joined.
func (h *handler) PushRingCallerJoined(ctx context.Context, sessionID uuid.UUID) error {
	return pushRingCallerJoined.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}).Err()
}

// PushRingFailed moves the session to push_failed. It returns true when the state changed.
func (h *handler) PushRingFailed(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	res, err := pushRingFailed.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

// PushRingClose closes the session. It returns the closed session, or nil when the session is missing or was closed
// before, so the close runs once.
func (h *handler) PushRingClose(ctx context.Context, sessionID uuid.UUID) (*pushring.Session, error) {
	tmp, err := pushRingClose.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}).Slice()
	if err != nil {
		return nil, err
	}
	if len(tmp) == 0 {
		return nil, nil
	}

	m := map[string]string{}
	for i := 0; i+1 < len(tmp); i += 2 {
		k, _ := tmp[i].(string)
		v, _ := tmp[i+1].(string)
		m[k] = v
	}

	return parsePushRingSession(sessionID, m)
}

// PushRingTicketClaim marks the ticket in use. It returns pushring.ErrTicketNotFound when no such ticket exists
// (the caller falls through to the other lookups) and pushring.ErrTicketRefused when the ticket exists but may not be used.
func (h *handler) PushRingTicketClaim(ctx context.Context, ticketHash string, now, inUseUntil time.Time) (*pushring.TicketClaim, error) {
	tk := pushRingTicketKey(ticketHash)

	m, err := h.Cache.HMGet(ctx, tk, "session_id", "device_id").Result()
	if err != nil {
		return nil, err
	}
	sessionIDStr, _ := m[0].(string)
	deviceIDStr, _ := m[1].(string)
	if sessionIDStr == "" {
		return nil, pushring.ErrTicketNotFound
	}
	sessionID := uuid.FromStringOrNil(sessionIDStr)

	res, err := pushRingTicketClaim.Run(ctx, h.Cache, []string{tk, pushRingSessionKey(sessionID)}, unixMilli(now), unixMilli(inUseUntil)).Int()
	if err != nil {
		return nil, err
	}
	switch res {
	case -1:
		return nil, pushring.ErrTicketNotFound
	case 1:
		return &pushring.TicketClaim{
			SessionID:  sessionID,
			DeviceID:   uuid.FromStringOrNil(deviceIDStr),
			TicketHash: ticketHash,
		}, nil
	}

	return nil, pushring.ErrTicketRefused
}

// pushRingTicketBind binds the ticket to its flow only while the ticket and the session keys exist and returns 0
// otherwise. A deleted ticket is not created again, so no key is left without a TTL. KEYS: ticket key, session key. ARGV: flow id, ticket
// hash, flow index key, session id, index TTL (seconds), session field prefix.
var pushRingTicketBind = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 or redis.call('EXISTS', KEYS[2]) == 0 then return 0 end
redis.call('HSET', KEYS[1], 'in_use_flow_id', ARGV[1])
redis.call('HSET', KEYS[2], ARGV[6] .. ARGV[1], ARGV[2])
redis.call('SET', ARGV[3], ARGV[4], 'EX', tonumber(ARGV[5]))
return 1
`)

// pushRingTicketMarkConsumed marks the ticket consumed only while the ticket key exists. KEYS: ticket key.
var pushRingTicketMarkConsumed = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
redis.call('HSET', KEYS[1], 'state', ARGV[1])
return 1
`)

// PushRingTicketBind binds the in use ticket to the temporary flow of its call and indexes the flow, so the call
// is identified before it runs its first action and the hangup hook finds the ticket. A ticket or session that was
// deleted in the meantime is not created again and the bind returns pushring.ErrTicketRefused.
func (h *handler) PushRingTicketBind(ctx context.Context, sessionID, flowID uuid.UUID, ticketHash string) error {
	res, err := pushRingTicketBind.Run(ctx, h.Cache,
		[]string{pushRingTicketKey(ticketHash), pushRingSessionKey(sessionID)},
		flowID.String(), ticketHash, pushRingIndexKey(pushring.IndexFlow, flowID), sessionID.String(),
		int(pushring.KeyTTL.Seconds()), pushRingFieldTicketFlowPrefix,
	).Int()
	if err != nil {
		return errors.Wrap(err, "could not bind the push ticket to the flow")
	}
	if res == 0 {
		// the ticket or the session is gone: the call must not run without the gate, so the bind is refused.
		return pushring.ErrTicketRefused
	}

	return nil
}

// PushRingTicketReturn returns the in use ticket of the given ticket flow to issued while no winner exists.
func (h *handler) PushRingTicketReturn(ctx context.Context, sessionID, flowID uuid.UUID) error {
	return pushRingTicketReturn.Run(ctx, h.Cache, []string{pushRingSessionKey(sessionID)}, flowID.String(), pushRingTicketKeyPrefix).Err()
}

// PushRingTicketConsume marks the ticket of the given ticket flow as consumed.
func (h *handler) PushRingTicketConsume(ctx context.Context, sessionID, flowID uuid.UUID) error {
	hash, err := h.Cache.HGet(ctx, pushRingSessionKey(sessionID), pushRingFieldTicketFlowPrefix+flowID.String()).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	return pushRingTicketMarkConsumed.Run(ctx, h.Cache, []string{pushRingTicketKey(hash)}, string(pushring.TicketStateConsumed)).Err()
}

// PushRingTicketsDelete deletes the tickets of the session except the ticket of the given device (uuid.Nil keeps none).
func (h *handler) PushRingTicketsDelete(ctx context.Context, s *pushring.Session, exceptDeviceID uuid.UUID) error {
	keys := []string{}
	for hash, deviceID := range s.Tickets {
		if exceptDeviceID != uuid.Nil && deviceID == exceptDeviceID {
			continue
		}
		keys = append(keys, pushRingTicketKey(hash))
	}
	if len(keys) == 0 {
		return nil
	}

	return h.Cache.Del(ctx, keys...).Err()
}
