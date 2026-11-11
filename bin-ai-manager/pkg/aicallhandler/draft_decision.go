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

// DraftDecision carries out the agent's decision on an SMS, email or call draft card
// without involving the LLM. The request holds ids only; the recipients and the
// text are read from the stored draft row, and the block type of that row
// (sms_draft, email_draft or call_draft) selects the send_message, send_email or create_call handler.
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

	tool, err := draftDecisionBuildTool(draft, draftID)
	if err != nil {
		return nil, err
	}

	result := h.draftDecisionExecute(ctx, c, tool, draft, decision)

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
// tool row carrying an sms_draft block with a recipient or an email_draft block
// with recipients, the stored draft still passes validation (approve: the E.164
// rule for SMS, validateEmailDraft for email), and the draft is not expired. Every refusal
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

	// UseNumber: the numbers in a call draft's action options must keep their exact
	// digits, as toolHandleCreateCall decodes them.
	var stored messageContent
	dec := json.NewDecoder(strings.NewReader(d.Content))
	dec.UseNumber()
	if errUnmarshal := dec.Decode(&stored); errUnmarshal != nil {
		return nil, draftNotFound()
	}
	draft := draftDecisionFindBlock(stored.Blocks)
	if draft == nil {
		return nil, draftNotFound()
	}

	if decision == message.DraftDecisionApprove {
		if errValid := draftDecisionShapeValid(draft); errValid != nil {
			return nil, errValid
		}
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

	// the flow binding is read after the expiry check, so an expired draft costs no read,
	// and before the claim, so a read error does not consume the draft.
	if decision == message.DraftDecisionApprove && draft.Type == message.BlockTypeCallDraft {
		ok, errBind := h.callDraftFlowBindingValid(ctx, c, draft.Call)
		if errBind != nil {
			return nil, errors.Wrapf(errBind, "could not read the flow of the call draft")
		}
		if !ok {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "DRAFT_INVALID", draftCallInvalidMessage)
		}
	}

	return draft, nil
}

// draftDecisionFindBlock returns the first card block that is a decidable draft:
// an sms_draft with a recipient, an email_draft with recipients or a call_draft
// with a call. It returns nil when there is none.
func draftDecisionFindBlock(blocks []CardBlock) *CardBlock {
	for i := range blocks {
		switch {
		case blocks[i].Type == message.BlockTypeSMSDraft && blocks[i].To != "":
			return &blocks[i]
		case blocks[i].Type == message.BlockTypeEmailDraft && len(blocks[i].Recipients) > 0:
			return &blocks[i]
		case blocks[i].Type == message.BlockTypeCallDraft && blocks[i].Call != nil:
			return &blocks[i]
		}
	}
	return nil
}

// draftDecisionShapeValid runs, on approve, the validation of the draft type
// again: the E.164 rule for SMS, validateEmailDraft for email and
// validateCallDraftShape for a call.
func draftDecisionShapeValid(draft *CardBlock) error {
	switch draft.Type {
	case message.BlockTypeEmailDraft:
		if !draftEmailStillValid(draft) {
			return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "DRAFT_INVALID", draftEmailInvalidMessage)
		}
	case message.BlockTypeCallDraft:
		if !callDraftShapeStillValid(draft.Call) {
			return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "DRAFT_INVALID", draftCallInvalidMessage)
		}
	default:
		if !draftToRegex.MatchString(draft.To) {
			return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "DRAFT_INVALID", "The draft recipient is not a valid phone number.")
		}
	}
	return nil
}

// draftDecisionBuildTool rebuilds, from the stored draft, the tool call that
// the decision runs: send_message, send_email or create_call with the stored
// arguments. The block type selects it.
func draftDecisionBuildTool(draft *CardBlock, draftID uuid.UUID) (*message.ToolCall, error) {
	var funcName message.FunctionCallName
	var args []byte
	var err error
	switch draft.Type {
	case message.BlockTypeEmailDraft:
		funcName = message.FunctionCallNameSendEmail
		destinations := make([]commonaddress.Address, 0, len(draft.Recipients))
		for _, r := range draft.Recipients {
			destinations = append(destinations, commonaddress.Address{Type: commonaddress.TypeEmail, Target: r})
		}
		args, err = json.Marshal(&fmaction.OptionEmailSend{
			Destinations: destinations,
			Subject:      draft.Subject,
			Content:      draft.Content,
		})
	case message.BlockTypeCallDraft:
		funcName = message.FunctionCallNameCreateCall
		args, err = callDraftCreateArguments(draft.Call)
	default:
		funcName = message.FunctionCallNameSendMessage
		args, err = json.Marshal(&fmaction.OptionMessageSend{
			Destinations: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: draft.To}},
			Text:         draft.Text,
		})
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the send arguments")
	}

	return &message.ToolCall{
		ID:   draftDecisionToolCallID(draftID),
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      funcName,
			Arguments: string(args),
		},
	}, nil
}

// draftDecisionExecute carries out the decision and returns the tool result. A
// reject runs nothing. An approve runs the handler of the draft type in code
// without an LLM turn: the metric label is the handler that ran.
func (h *aicallHandler) draftDecisionExecute(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall, draft *CardBlock, decision message.DraftDecision) *messageContent {
	if decision != message.DraftDecisionApprove {
		result := newToolResult(tool.ID)
		result.Result = "rejected"
		switch draft.Type {
		case message.BlockTypeEmailDraft:
			result.Message = draftEmailRejectedMessage
		case message.BlockTypeCallDraft:
			result.Message = draftCallRejectedMessage
		default:
			result.Message = draftRejectedMessage
		}
		return result
	}

	promAIcallToolExecuteTotal.WithLabelValues(labelForToolExecuteMetric(tool.Function.Name)).Inc()

	switch draft.Type {
	case message.BlockTypeEmailDraft:
		result := h.toolHandleEmailSend(ctx, c, tool)
		if result.Result == "failed" {
			result.Message += draftDeliveryUnconfirmedSuffix
		} else {
			// the send is asynchronous: email-manager answers once the email exists and a
			// provider failure afterwards is never reported, so a success is not a delivery.
			result.Message = draftEmailAcceptedMessage
		}
		return result

	case message.BlockTypeCallDraft:
		// the handler's result is kept as is: on success the created call ids, on failure
		// no call was created (an error is returned only when every destination failed),
		// so there is nothing unconfirmed to add.
		return h.toolHandleCreateCall(ctx, c, tool)

	default:
		result := h.toolHandleMessageSend(ctx, c, tool)
		if result.Result == "failed" {
			result.Message += draftDeliveryUnconfirmedSuffix
		}
		return result
	}
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

// draftEmailStillValid reports whether a stored email draft passes validation
// again: the same rules as at creation, and the stored recipients must already be
// in their normalized form (a row that was changed after creation is refused).
func draftEmailStillValid(draft *CardBlock) bool {
	normalized, err := validateEmailDraft(draft.Recipients, draft.Subject, draft.Content)
	if err != nil || len(normalized) != len(draft.Recipients) {
		return false
	}
	for i := range normalized {
		if normalized[i] != draft.Recipients[i] {
			return false
		}
	}
	return true
}
