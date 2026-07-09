package message

import (
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

type Message struct {
	identity.Identity

	// referenced pipecatcall info
	// this is used for simplifying queries.
	PipecatcallID            uuid.UUID                 `json:"pipecatcall_id,omitempty"`
	PipecatcallReferenceType pipecatcall.ReferenceType `json:"pipecatcall_reference_type,omitempty"`
	PipecatcallReferenceID   uuid.UUID                 `json:"pipecatcall_reference_id,omitempty"`
	ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`

	// InReplyToMessageID is the ID of the user-authored message (as passed to
	// SendMessage) that this bot response answers. Snapshotted once per LLM
	// generation (on the first bot-llm-text token) from
	// Session.SetPendingInReplyToMessageID / SnapshotPendingInReplyToMessageID, so every intermediate and final event
	// for that generation carries the same value. Zero UUID if the session had
	// no pending inbound message ID at generation start (e.g. voice calls,
	// which do not go through the same-aicall-reuse SendMessage path this
	// field exists to disambiguate). See VOIP-1234 design doc §4-1.
	InReplyToMessageID uuid.UUID `json:"in_reply_to_message_id,omitempty"`

	Text     string `json:"text,omitempty"`
	Sequence int    `json:"sequence,omitempty"`
}
