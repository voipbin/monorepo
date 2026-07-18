package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

// getExecuteMode reads the conversation's Owner snapshot and returns the dispatch mode.
// See docs/plans/2026-04-30-assignable-conversation-design.md §3.1: callers MUST NOT re-fetch
// the Conversation in the dispatch path; the snapshot already loaded by the inbound handler is authoritative.
func (h *conversationHandler) getExecuteMode(cv *conversation.Conversation) ExecuteMode {
	if cv.OwnerType == commonidentity.OwnerTypeAgent && cv.OwnerID != uuid.Nil {
		return ExecuteModeAgent
	}
	return ExecuteModeFlow
}

// caseIDHint reads the single case-linking hint off a Conversation's
// Metadata.ContactCaseID (contact-case-management design §4.3), for
// callers building MessageCreateArgs. Returns nil when Metadata is nil
// or ContactCaseID is unset -- the overwhelmingly common case, since
// most Conversations are never linked to a Case.
//
// Deliberately NOT read by getExecuteMode above or any agent/flow
// dispatch decision: this hint carries case-linking information for
// bin-contact-manager's Case get-or-create only, and must never
// influence how a message is routed to an agent or a flow (that
// decision is governed exclusively by Conversation.Owner, per
// getExecuteMode's own doc comment). See
// Test_CaseIDHint_NeverReadByExecuteMode for the explicit negative
// test the design doc requires.
func caseIDHint(cv *conversation.Conversation) *uuid.UUID {
	if cv == nil || cv.Metadata == nil {
		return nil
	}
	return cv.Metadata.ContactCaseID
}

// runExecuteModeAgent handles inbound messages on conversations owned by an agent.
// The agent UI learns of new messages via the existing `message_created` event filtered on cv.OwnerID.
// No new event is published; no flow is triggered. Logging only.
func (h *conversationHandler) runExecuteModeAgent(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "runExecuteModeAgent",
		"conversation_id": cv.ID,
		"message_id":      m.ID,
		"owner_id":        cv.OwnerID,
	})
	log.Infof("Conversation owned by agent. Skipping flow trigger.")
	return nil
}

// runExecuteModeFlow dispatches by conversation type. Each per-type runner fetches the type-specific
// flow source (account for LINE/WhatsApp, number for SMS, widget for Webchat) and calls executeActiveflow with the resolved flow id.
func (h *conversationHandler) runExecuteModeFlow(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	switch cv.Type {
	case conversation.TypeLine:
		return h.runExecuteModeFlowLine(ctx, cv, m)
	case conversation.TypeMessage:
		return h.runExecuteModeFlowMessage(ctx, cv, m)
	case conversation.TypeWhatsApp:
		return h.runExecuteModeFlowWhatsApp(ctx, cv, m)
	case conversation.TypeWebchat:
		return h.runExecuteModeFlowWebchat(ctx, cv, m)
	default:
		logrus.WithFields(logrus.Fields{
			"func":            "runExecuteModeFlow",
			"conversation_id": cv.ID,
			"type":            cv.Type,
		}).Infof("Unsupported conversation type for flow execution. Skipping.")
		return nil
	}
}

// runExecuteModeFlowLine fetches the account by cv.AccountID and triggers an activeflow
// using account.MessageFlowID. Returns nil without side effects when AccountID is uuid.Nil.
func (h *conversationHandler) runExecuteModeFlowLine(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	if cv.AccountID == uuid.Nil {
		return nil
	}
	ac, errGet := h.accountHandler.Get(ctx, cv.AccountID)
	if errGet != nil {
		return errors.Wrapf(errGet, "could not get account. account_id: %s", cv.AccountID)
	}
	if errExecute := h.executeActiveflow(ctx, cv, m, ac.MessageFlowID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute activeflow. account_id: %s", ac.ID)
	}
	return nil
}

// runExecuteModeFlowMessage fetches the number by cv.Self.Target and triggers an activeflow
// using number.MessageFlowID.
func (h *conversationHandler) runExecuteModeFlowMessage(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	num, errGet := h.NumberGet(ctx, cv.Self.Target)
	if errGet != nil {
		return errors.Wrapf(errGet, "could not get number. number: %s", cv.Self.Target)
	}
	if errExecute := h.executeActiveflow(ctx, cv, m, num.MessageFlowID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute activeflow. number_id: %s", num.ID)
	}
	return nil
}

// runExecuteModeFlowWhatsApp fetches the account by cv.AccountID and triggers an activeflow
// using account.MessageFlowID. Returns nil without side effects when AccountID is uuid.Nil.
func (h *conversationHandler) runExecuteModeFlowWhatsApp(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	if cv.AccountID == uuid.Nil {
		return nil
	}
	ac, errGet := h.accountHandler.Get(ctx, cv.AccountID)
	if errGet != nil {
		return errors.Wrapf(errGet, "could not get account. account_id: %s", cv.AccountID)
	}
	if errExecute := h.executeActiveflow(ctx, cv, m, ac.MessageFlowID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute activeflow. account_id: %s", ac.ID)
	}
	return nil
}

// runExecuteModeFlowWebchat fetches the Widget by cv.Self.Target (the
// Widget's own UUID, per messageEventReceivedWebchat's Self=Widget.ID
// address construction) and triggers an activeflow using
// Widget.MessageFlowID. Mirrors runExecuteModeFlowLine/WhatsApp's shape
// exactly, except the flow source is a Widget (via WebchatV1WidgetGet)
// rather than an Account -- webchat conversations never have an
// AccountID. See design doc
// 2026-07-18-webchat-message-flow-owner-migration-design.md §3: this
// moves MessageFlowID's Flow-trigger ownership here from
// bin-webchat-manager, matching every other channel.
func (h *conversationHandler) runExecuteModeFlowWebchat(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	widgetID, errParse := uuid.FromString(cv.Self.Target)
	if errParse != nil {
		return errors.Wrapf(errParse, "invalid widget id in conversation self target: %s", cv.Self.Target)
	}
	w, errGet := h.reqHandler.WebchatV1WidgetGet(ctx, widgetID)
	if errGet != nil {
		return errors.Wrapf(errGet, "could not get widget. widget_id: %s", widgetID)
	}
	if errExecute := h.executeActiveflow(ctx, cv, m, w.MessageFlowID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute activeflow. widget_id: %s", w.ID)
	}
	return nil
}
