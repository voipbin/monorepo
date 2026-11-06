package message

import "monorepo/bin-message-manager/models/target"

// EventCopy returns a copy of the message that is safe to publish on the internal event bus: it has its own
// Targets slice and the internal provider keys of the targets are cleared, so the provider message ids do not
// leave the service. The receiver is not changed.
func (h *Message) EventCopy() *Message {
	res := *h

	if h.Targets != nil {
		res.Targets = make([]target.Target, len(h.Targets))
		copy(res.Targets, h.Targets)
		for i := range res.Targets {
			res.Targets[i].ProviderParts = 0
			res.Targets[i].ProviderReferenceID = ""
		}
	}

	// the message level provider reference id is not published either.
	res.ProviderReferenceID = ""

	return &res
}
