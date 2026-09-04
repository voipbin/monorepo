package aicallhandler

import (
	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
)

// isListenableCallStatus reports whether a call is in a state transcribe-manager
// will accept. It mirrors transcribehandler.isValidReference's own set exactly;
// diverging would mean starting a transcribe that is then refused.
func isListenableCallStatus(status cmcall.Status) bool {
	return status == cmcall.StatusDialing || status == cmcall.StatusRinging || status == cmcall.StatusProgressing
}

// listenTranscribeIDFromMetadata reads the listen transcribe id off the AIcall's
// metadata, returning uuid.Nil when absent or unparseable.
func listenTranscribeIDFromMetadata(c *aicall.AIcall) uuid.UUID {
	if c.Metadata == nil {
		return uuid.Nil
	}

	tmp, ok := c.Metadata[aicall.MetaKeyListenTranscribeID].(string)
	if !ok {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(tmp)
}

// listenOwnsTranscribeFromMetadata reports whether this AIcall started the
// transcribe session it is listening to, and may therefore stop it.
func listenOwnsTranscribeFromMetadata(c *aicall.AIcall) bool {
	if c.Metadata == nil {
		return false
	}

	owns, ok := c.Metadata[aicall.MetaKeyListenOwnsTranscribe].(bool)
	if !ok {
		return false
	}

	return owns
}
