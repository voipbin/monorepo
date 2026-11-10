package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// draftDecisionNamespace is the UUIDv5 namespace of the decision claim id.
// The claim row id is derived from the draft id, so a second decision of the
// same draft collides on the primary key of the message table. Changing this
// value would make the claims of earlier decisions invisible, so it is never
// changed.
var draftDecisionNamespace = uuid.Must(uuid.FromString("2fcc8cd1-51ff-4100-886d-961edc8c1c54"))

const (
	draftDecisionExpiryDefaultSeconds = 600

	draftClaimType = "draft_decision"

	draftDeliveryUnconfirmedSuffix = " Delivery may be unconfirmed."
	draftRejectedMessage           = "The agent rejected this draft. The message was not sent."
)

// draftClaimContent is the content of the claim row written before a decision
// is carried out.
type draftClaimContent struct {
	Type     string                `json:"type"`
	DraftID  uuid.UUID             `json:"draft_id"`
	Decision message.DraftDecision `json:"decision"`
}

func draftDecisionClaimID(draftID uuid.UUID) uuid.UUID {
	return uuid.NewV5(draftDecisionNamespace, draftID.String())
}

func draftDecisionToolCallID(draftID uuid.UUID) string {
	return "dd_" + strings.ReplaceAll(draftID.String(), "-", "")
}

func draftNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNameAIManager, "DRAFT_NOT_FOUND", "The draft was not found.")
}

// DraftDecision carries out the agent's decision on an SMS draft card without
// involving the LLM. The request holds ids only; the recipient and the text
// are read from the stored draft row.
//
// Rows written for one decision (none is updated afterwards):
//   - a claim row (role notification), written first. Its id is derived from
//     the draft id, so approve and reject exclude each other and a repeated
//     decision is refused. Replay skips notification rows.
//   - after the send, an assistant tool-call row and a tool result row with
//     the synthetic tool call id dd_<draft id hex>, written one after the other.
//
// If the process stops after the claim and before the pair is written, only
// the claim remains; nothing is resent automatically.
func (h *aicallHandler) DraftDecision(ctx context.Context, aicallID uuid.UUID, draftID uuid.UUID, decision message.DraftDecision) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "DraftDecision",
		"aicall_id": aicallID,
		"draft_id":  draftID,
		"decision":  decision,
	})

	if decision != message.DraftDecisionApprove && decision != message.DraftDecisionReject {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_DECISION", "The decision must be approve or reject.")
	}

	c, err := h.Get(ctx, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	// every validation, including the expiry, runs before the claim, so a refused request never consumes the draft.
	draft, err := h.draftDecisionLoad(ctx, c, draftID, decision)
	if err != nil {
		return nil, err
	}

	if errClaim := h.draftDecisionClaim(ctx, c, draftID, decision); errClaim != nil {
		return nil, errClaim
	}

	args, err := json.Marshal(&fmaction.OptionMessageSend{
		Destinations: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: draft.To}},
		Text:         draft.Text,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the send arguments")
	}
	tool := &message.ToolCall{
		ID:   draftDecisionToolCallID(draftID),
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameSendMessage,
			Arguments: string(args),
		},
	}

	var result *messageContent
	if decision == message.DraftDecisionApprove {
		promAIcallToolExecuteTotal.WithLabelValues(labelForToolExecuteMetric(tool.Function.Name)).Inc()
		result = h.toolHandleMessageSend(ctx, c, tool)
		if result.Result == "failed" {
			result.Message += draftDeliveryUnconfirmedSuffix
		}
	} else {
		result = newToolResult(tool.ID)
		result.Result = "rejected"
		result.Message = draftRejectedMessage
	}

	activeAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
	if _, errCall := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "",
		messagehandler.WithActiveAIID(activeAIID)); errCall != nil {
		log.WithError(errCall).Errorf("Could not record the decision tool call row. The draft stays in the processing state. result: %s", result.Result)
		return nil, errors.Wrapf(errCall, "could not record the decision tool call")
	}
	res, err := h.toolCreateResultMessage(ctx, c, tool, result, activeAIID, message.OriginNone)
	if err != nil {
		log.WithError(err).Errorf("Could not record the decision result row. The draft stays in the processing state. result: %s", result.Result)
		return nil, errors.Wrapf(err, "could not record the decision result")
	}

	log.Infof("Decided the draft. result: %s", result.Result)
	return res, nil
}

// draftDecisionLoad reads the draft row and runs every check that must pass
// before the claim: the row belongs to the aicall and its customer, it is a
// tool row carrying an sms_draft block with a recipient, the recipient is a
// valid E.164 number (approve), and the draft is not expired. Every refusal
// about the row itself is the same NotFound, so ids cannot be probed. It
// returns the card block holding the recipient and the text.
func (h *aicallHandler) draftDecisionLoad(ctx context.Context, c *aicall.AIcall, draftID uuid.UUID, decision message.DraftDecision) (*CardBlock, error) {
	d, err := h.messageHandler.Get(ctx, draftID)
	if err != nil {
		var ve *cerrors.VoipbinError
		if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
			return nil, draftNotFound()
		}
		return nil, errors.Wrapf(err, "could not get the draft")
	}
	if d.AIcallID != c.ID || d.CustomerID != c.CustomerID || d.Role != message.RoleTool {
		return nil, draftNotFound()
	}

	var stored messageContent
	if errUnmarshal := json.Unmarshal([]byte(d.Content), &stored); errUnmarshal != nil {
		return nil, draftNotFound()
	}
	var draft *CardBlock
	for i := range stored.Blocks {
		if stored.Blocks[i].Type == message.BlockTypeSMSDraft && stored.Blocks[i].To != "" {
			draft = &stored.Blocks[i]
			break
		}
	}
	if draft == nil {
		return nil, draftNotFound()
	}

	if decision == message.DraftDecisionApprove && !draftToRegex.MatchString(draft.To) {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "DRAFT_INVALID", "The draft recipient is not a valid phone number.")
	}

	// expiry is evaluated lazily; a draft without a creation time counts as expired (fail closed).
	expiry := config.Get().AIcallDraftDecisionExpirySeconds
	if expiry <= 0 {
		expiry = draftDecisionExpiryDefaultSeconds
	}
	now := h.utilHandler.TimeNow()
	if d.TMCreate == nil || now == nil || now.Sub(*d.TMCreate) > time.Duration(expiry)*time.Second {
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameAIManager, "DRAFT_EXPIRED", "The draft expired and can no longer be decided.")
	}

	return draft, nil
}

// draftDecisionClaim writes the claim row. Its id is derived from the draft id,
// so the primary key conflict is the atomic exclusion of a second decision.
func (h *aicallHandler) draftDecisionClaim(ctx context.Context, c *aicall.AIcall, draftID uuid.UUID, decision message.DraftDecision) error {
	claim, err := json.Marshal(&draftClaimContent{Type: draftClaimType, DraftID: draftID, Decision: decision})
	if err != nil {
		return errors.Wrapf(err, "could not marshal the claim content")
	}

	if _, errCreate := h.messageHandler.Create(ctx, draftDecisionClaimID(draftID), c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(claim), nil, ""); errCreate != nil {
		if dbhandler.IsDuplicateKey(errCreate) {
			return cerrors.AlreadyExists(commonoutline.ServiceNameAIManager, "DRAFT_ALREADY_DECIDED", "The draft was already decided.")
		}
		return errors.Wrapf(errCreate, "could not record the decision claim")
	}

	return nil
}
