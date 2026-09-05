package aicallhandler

import (
	"monorepo/bin-ai-manager/models/aicall"

	"github.com/gofrs/uuid"
)

// listenKind discriminates which kind of listen session an AIcall holds
// (design 2026-09-05 §5.2.1). An AIcall carries at most one listen pointer.
type listenKind string

const (
	listenKindNone         listenKind = ""
	listenKindCall         listenKind = "call"
	listenKindConversation listenKind = "conversation"
)

// listenKindLabelUnknown is the metric `kind` label value for sites that fire
// before a listen pointer exists or before the Case's reference type is known
// (design 2026-09-05 §5.13).
const listenKindLabelUnknown = "unknown"

// listenConversationIDFromMetadata reads the conversation this AIcall listens
// to, or uuid.Nil when the key is absent or malformed.
func listenConversationIDFromMetadata(c *aicall.AIcall) uuid.UUID {
	if c == nil || c.Metadata == nil {
		return uuid.Nil
	}

	tmp, ok := c.Metadata[aicall.MetaKeyListenConversationID].(string)
	if !ok {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(tmp)
}

// listenKindOf resolves the AIcall's listen kind from its metadata pointers.
func listenKindOf(c *aicall.AIcall) listenKind {
	if listenTranscribeIDFromMetadata(c) != uuid.Nil {
		return listenKindCall
	}
	if listenConversationIDFromMetadata(c) != uuid.Nil {
		return listenKindConversation
	}
	return listenKindNone
}

// label renders the kind as its metric label value; none maps to unknown.
func (k listenKind) label() string {
	if k == listenKindNone {
		return listenKindLabelUnknown
	}
	return string(k)
}

// listenTerminateNeedsStop is ProcessTerminate's gate: only a contact_case
// AIcall can be listening, and it is listening if either the call-kind column
// or either metadata pointer says so (design 2026-09-05 §5.7; the ListenCallID
// clause is kept as belt-and-braces for call rows).
func listenTerminateNeedsStop(c *aicall.AIcall) bool {
	if c == nil || c.ReferenceType != aicall.ReferenceTypeContactCase {
		return false
	}
	return c.ListenCallID != uuid.Nil || listenKindOf(c) != listenKindNone
}

const (
	listenTranscriptHeaderCall         = "Live call transcript so far:"
	listenTranscriptHeaderConversation = "Conversation so far:"
)

// listenTranscriptHeader picks the transcript block's first line by kind.
func listenTranscriptHeader(kind listenKind) string {
	if kind == listenKindConversation {
		return listenTranscriptHeaderConversation
	}
	return listenTranscriptHeaderCall
}
