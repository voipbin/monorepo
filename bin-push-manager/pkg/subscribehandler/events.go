package subscribehandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"monorepo/bin-common-handler/models/sock"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tkmessage "monorepo/bin-talk-manager/models/message"
)

// processEventAMAgentDeleted handles the agent-manager's agent_deleted event: the devices and the notification
// settings of the agent are deleted. Both are tried, and the errors are returned together.
func (h *subscribeHandler) processEventAMAgentDeleted(ctx context.Context, m *sock.Event) error {
	a := &amagent.Agent{}
	if err := json.Unmarshal([]byte(m.Data), a); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	errDevice := h.deviceHandler.EventAgentDeleted(ctx, a)
	errSettings := h.settingsHandler.EventAgentDeleted(ctx, a)

	return errors.Join(errDevice, errSettings)
}

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event: the devices and the
// notification settings of the customer are deleted. Both are tried, and the errors are returned together.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *sock.Event) error {
	c := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), c); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	errDevice := h.deviceHandler.EventCustomerDeleted(ctx, c)
	errSettings := h.settingsHandler.EventCustomerDeleted(ctx, c)

	return errors.Join(errDevice, errSettings)
}

// processEventTKMessageCreated handles the talk-manager's chatmessage_created event.
func (h *subscribeHandler) processEventTKMessageCreated(ctx context.Context, m *sock.Event) error {
	msg := &tkmessage.Message{}
	if err := json.Unmarshal([]byte(m.Data), msg); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.eventHandler.EventChatMessageCreated(ctx, msg)
}

// processEventCVMessageCreated handles the conversation-manager's conversation_message_created event.
func (h *subscribeHandler) processEventCVMessageCreated(ctx context.Context, m *sock.Event) error {
	msg := &cvmessage.Message{}
	if err := json.Unmarshal([]byte(m.Data), msg); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.eventHandler.EventConversationMessageCreated(ctx, msg)
}

// processEventCMCaseNoteCreated handles the contact-manager's case_note_created event.
func (h *subscribeHandler) processEventCMCaseNoteCreated(ctx context.Context, m *sock.Event) error {
	n := &cmcasenote.CaseNote{}
	if err := json.Unmarshal([]byte(m.Data), n); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.eventHandler.EventCaseNoteCreated(ctx, n)
}

// processEventCMCaseAssigned handles the contact-manager's case_assigned event.
func (h *subscribeHandler) processEventCMCaseAssigned(ctx context.Context, m *sock.Event) error {
	e := &cmkase.CaseAssignedEvent{}
	if err := json.Unmarshal([]byte(m.Data), e); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.eventHandler.EventCaseAssigned(ctx, e)
}
