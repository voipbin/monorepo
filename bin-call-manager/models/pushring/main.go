// Package pushring defines the push ring session of an incoming extension call.
//
// A session is the single owner of the ring window of a call that may be answered by the mobile app through a
// push notification. It lives in the cache only (no database table).
package pushring

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

// list of the timings of a session.
const (
	// Window is the ring window. It has the same value as the dial timeout of the legs.
	Window = 60 * time.Second

	// RedundantExpireDelay is the delay of the second, redundant expire request.
	RedundantExpireDelay = 90 * time.Second

	// KeyTTL is the cache ttl of the session and the ticket keys (window + 90 s), so the redundant expire request
	// still finds them and the ttl is the last cleanup when both requests are lost.
	KeyTTL = 150 * time.Second

	// LeaseTTL is the time a session call may hold the join lease.
	LeaseTTL = 15 * time.Second

	// TicketInUseTTL is the time a ticket stays in use before a call exists (or after the call vanished).
	TicketInUseTTL = 15 * time.Second

	// StartTimeout is the deadline of the whole start (the RPC timeout of the caller is 5 s).
	StartTimeout = 4 * time.Second
)

// JoinAddressFormat is the one-time join address given to the app. The ticket already holds the direct prefix.
const JoinAddressFormat = "sip:%s@sip.voipbin.net"

// State is the state of a session.
type State string

// list of the session states.
const (
	StateRinging    State = "ringing"     // the window is open.
	StatePushFailed State = "push_failed" // every device failed. The pushes are gone but the SIP gate stays open.
	StateClosed     State = "closed"      // the gate is closed. The keys stay until the cache ttl.
)

// TicketState is the state of a ticket.
type TicketState string

// list of the ticket states.
const (
	TicketStateIssued   TicketState = "issued"
	TicketStateInUse    TicketState = "in_use"
	TicketStateConsumed TicketState = "consumed"
)

// list of the kinds of the session lookup keys.
const (
	IndexConfbridge = "cb"     // confbridge id -> session id
	IndexMaster     = "master" // master call id -> session id
	IndexFlow       = "flow"   // flow id of a call of the session (the legs and the ticket calls) -> session id
)

// WinnerSIP is the winner value of a session that was answered by a SIP leg. The winner of the app is
// WinnerAppPrefix + device id.
const (
	WinnerSIP       = "sip"
	WinnerAppPrefix = "app:"
)

// Session is the push ring session.
type Session struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	AgentID     uuid.UUID
	ExtensionID uuid.UUID

	ConfbridgeID uuid.UUID
	MasterCallID uuid.UUID
	FlowID       uuid.UUID // the temporary flow that connect created. It identifies the legs of the session.

	ExpiresAt time.Time
	State     State

	LeaseOwner uuid.UUID
	LeaseUntil time.Time

	Winner        string
	CallerJoined  bool
	SIPRegistered bool

	// Tickets maps the ticket hash to the device id. It is used to delete the tickets of a session.
	Tickets map[string]uuid.UUID

	// TicketFlows maps the id of the flow of a ticket call to the ticket hash. The flow exists before the call does,
	// so the call is identified by its flow.
	TicketFlows map[uuid.UUID]string
}

// StartResult is the result of the start request.
type StartResult struct {
	Started       bool      `json:"started"`
	SessionID     uuid.UUID `json:"session_id,omitempty"`
	SIPRegistered bool      `json:"sip_registered,omitempty"`
}

// TicketClaim is the result of a ticket claim.
type TicketClaim struct {
	SessionID  uuid.UUID
	DeviceID   uuid.UUID
	TicketHash string
}

// list of the errors of the cache layer.
var (
	ErrSessionNotFound = errors.New("the push ring session was not found")
	ErrTicketNotFound  = errors.New("the push ticket was not found")
	ErrTicketRefused   = errors.New("the push ticket was refused")
)
