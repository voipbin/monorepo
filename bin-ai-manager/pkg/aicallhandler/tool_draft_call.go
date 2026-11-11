package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/actioncatalog"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// draft_call limits (docs/plans/2026-11-11-insight-assistant-call-design.md,
// section 4). There is no policy limit on the call arguments (the customer's AI
// prompt controls them); the limits below are technical. The variables limits
// mirror flow-manager's (activeflowhandler/main.go): flow-manager drops the
// whole variables map without an error when one is exceeded, so the draft
// refuses such a map instead of showing variables that would not be applied.
const (
	draftCallTitleDefault      = "Call draft"
	draftCallDisplayedMessage  = "Displayed a call draft for review."
	draftCallRejectedMessage   = "The agent rejected this draft. The call was not placed."
	draftCallInvalidMessage    = "The draft destinations, flow, actions or variables are not valid."
	draftCallVariablesMaxKeys  = 100
	draftCallVariablesMaxValue = 32 * 1024
	draftCallVariablesMaxTotal = 64 * 1024
	draftCallVariablesReserved = "voipbin."
)

// CallDraft is the call carried by a "call_draft" card block: the arguments of
// the create_call tool as the agent saw them, plus labels read from the flow at
// draft time. Exactly one of FlowID and Actions is set.
//
// The Flow* fields other than FlowID and FlowTMUpdate are labels for the
// approver, not proof of trust: an ordinary agent can build a temporary flow
// with any name. FlowTMUpdate binds the approval to the flow as it was shown
// (empty when the flow has never been updated).
type CallDraft struct {
	Destinations []commonaddress.Address `json:"destinations"`
	Source       *commonaddress.Address  `json:"source,omitempty"`
	Anonymous    string                  `json:"anonymous,omitempty"`
	Variables    map[string]string       `json:"variables,omitempty"`

	FlowID           string                  `json:"flow_id,omitempty"`
	FlowName         string                  `json:"flow_name,omitempty"`
	FlowActionTypes  []string                `json:"flow_action_types,omitempty"`
	FlowPersistent   *bool                   `json:"flow_persistent,omitempty"`
	FlowDestinations []commonaddress.Address `json:"flow_destinations,omitempty"`
	FlowTMUpdate     string                  `json:"flow_tm_update,omitempty"`

	Actions []fmaction.Action `json:"actions,omitempty"`
}

// toolHandleDraftCall handles the draft_call tool. It records a "call_draft"
// card block in the session's own message stream and places no call: nothing is
// called until the agent approves the card through the draft-decision path. The
// id of the stored tool result row is the draft id.
//
// The result message never contains the draft body. Logs carry counts only,
// never a number.
func (h *aicallHandler) toolHandleDraftCall(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleDraftCall",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool draft_call.")

	res := newToolResult(tool.ID)

	draft, title, err := parseCallDraftArguments(tool.Function.Arguments)
	if err != nil {
		fillFailed(res, err)
		return res
	}
	if err := validateCallDraftShape(draft); err != nil {
		fillFailed(res, err)
		return res
	}

	if draft.FlowID != "" {
		if err := h.callDraftFillFlow(ctx, c, draft); err != nil {
			fillFailed(res, err)
			return res
		}
	}

	res.Blocks = []CardBlock{
		{
			Type:  message.BlockTypeCallDraft,
			Title: title,
			Call:  draft,
		},
	}
	fillSuccess(res, "card", "", draftCallDisplayedMessage)

	stored, errMarshal := json.Marshal(res)
	if errMarshal != nil {
		res = newToolResult(tool.ID)
		fillFailed(res, errMarshal)
		return res
	}
	if len(stored) > draftStoredMaxBytes {
		res = newToolResult(tool.ID)
		fillFailed(res, fmt.Errorf("the draft is too large to store: %d bytes after encoding, the limit is %d. Shorten the draft and draft again", len(stored), draftStoredMaxBytes))
		return res
	}

	log.Debugf("Created call draft. destinations: %d, actions: %d, variables: %d, flow: %v", len(draft.Destinations), len(draft.Actions), len(draft.Variables), draft.FlowID != "")

	return res
}

// parseCallDraftArguments decodes the draft_call arguments the way
// toolHandleCreateCall decodes create_call's (UseNumber, so a number in an
// action option keeps its exact digits) and returns the draft and the card
// title. It checks only what decoding needs; validateCallDraftShape checks the
// rest and runs again at approval.
func parseCallDraftArguments(arguments string) (*CallDraft, string, error) {
	var args struct {
		FlowID       uuid.UUID               `json:"flow_id"`
		Actions      []fmaction.Action       `json:"actions,omitempty"`
		Source       *commonaddress.Address  `json:"source,omitempty"`
		Destinations []commonaddress.Address `json:"destinations"`
		Anonymous    string                  `json:"anonymous,omitempty"`
		Variables    map[string]any          `json:"variables,omitempty"`
		Title        string                  `json:"title"`
	}
	dec := json.NewDecoder(strings.NewReader(arguments))
	dec.UseNumber()
	if err := dec.Decode(&args); err != nil {
		return nil, "", err
	}

	draft := &CallDraft{
		Destinations: args.Destinations,
		Source:       args.Source,
		Anonymous:    args.Anonymous,
		Variables:    fmvariable.NewVariablesFromMap(args.Variables),
		Actions:      args.Actions,
	}
	if args.FlowID != uuid.Nil {
		draft.FlowID = args.FlowID.String()
	}

	title := strings.TrimSpace(args.Title)
	if title == "" {
		title = draftCallTitleDefault
	}
	return draft, truncateCardText(title, cardTitleMaxLen), nil
}

// validateCallDraftShape checks a call draft. It refuses, it never cuts. The
// same function runs when the draft is created and when it is approved, so a
// stored draft is decided by the rules it was created under.
func validateCallDraftShape(d *CallDraft) error {
	if err := validateCallDraftTarget(d); err != nil {
		return err
	}
	if err := validateCallDraftActions(d.Actions); err != nil {
		return err
	}
	return validateCallDraftVariables(d.Variables)
}

// validateCallDraftTarget checks flow_id XOR actions and the destinations.
func validateCallDraftTarget(d *CallDraft) error {
	hasFlow := d.FlowID != ""
	if hasFlow {
		if id := uuid.FromStringOrNil(d.FlowID); id == uuid.Nil {
			return fmt.Errorf("flow_id must be a valid UUID")
		}
	}
	hasActions := len(d.Actions) > 0
	switch {
	case hasFlow && hasActions:
		return fmt.Errorf("provide either flow_id or actions, not both")
	case !hasFlow && !hasActions:
		return fmt.Errorf("either flow_id or actions is required")
	}

	if len(d.Destinations) == 0 {
		return fmt.Errorf("at least one destination is required")
	}
	for _, dest := range d.Destinations {
		if dest.Target == "" {
			return fmt.Errorf("destination target must not be empty")
		}
	}
	return nil
}

// validateCallDraftActions refuses an action type that flow-manager does not
// know. flow-manager validates the rest when the flow is created at approval.
func validateCallDraftActions(actions []fmaction.Action) error {
	known := map[string]bool{}
	for _, t := range actioncatalog.ActionTypeEnum() {
		known[t] = true
	}
	for i, a := range actions {
		if !known[string(a.Type)] {
			return fmt.Errorf("actions[%d] has an unknown type %q. Use describe_action to look up the action types", i, a.Type)
		}
	}
	return nil
}

// validateCallDraftVariables refuses the variables that flow-manager would
// drop. A key that is empty, not trimmed or reserved, more than the key limit,
// a value or the total above its limit.
func validateCallDraftVariables(vars map[string]string) error {
	if len(vars) > draftCallVariablesMaxKeys {
		return fmt.Errorf("variables has %d keys, the limit is %d", len(vars), draftCallVariablesMaxKeys)
	}
	total := 0
	for k, v := range vars {
		key := strings.TrimSpace(k)
		switch {
		case key == "":
			return fmt.Errorf("a variables key must not be empty")
		case key != k:
			return fmt.Errorf("a variables key must not start or end with a space")
		case strings.HasPrefix(strings.ToLower(key), draftCallVariablesReserved):
			return fmt.Errorf("a variables key must not start with %q", draftCallVariablesReserved)
		case len(v) > draftCallVariablesMaxValue:
			return fmt.Errorf("a variables value is above %d bytes", draftCallVariablesMaxValue)
		}
		total += len(k) + len(v)
	}
	if total > draftCallVariablesMaxTotal {
		return fmt.Errorf("variables are above %d bytes in total", draftCallVariablesMaxTotal)
	}
	return nil
}

// callDraftFillFlow reads the flow of a flow_id draft and fills the card
// labels and the binding value. A missing flow and a flow of another customer
// give the same masked error as create_call, so the tool is not a flow
// existence oracle.
func (h *aicallHandler) callDraftFillFlow(ctx context.Context, c *aicall.AIcall, d *CallDraft) error {
	flowID := uuid.FromStringOrNil(d.FlowID)
	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil || f == nil {
		logrus.WithField("aicall_id", c.ID).Errorf("Could not get the flow for the call draft. flow_id: %s, err: %v", flowID, err)
		return errCouldNotResolveFlow
	}
	if f.CustomerID != c.CustomerID {
		logrus.WithField("aicall_id", c.ID).Warnf("Flow does not belong to the customer. flow_id: %s, flow_customer_id: %s, customer_id: %s", flowID, f.CustomerID, c.CustomerID)
		return errCouldNotResolveFlow
	}

	persistent := f.Persist
	d.FlowName = f.Name
	d.FlowPersistent = &persistent
	d.FlowTMUpdate = callDraftTMUpdate(f.TMUpdate)
	d.FlowActionTypes, d.FlowDestinations = callDraftFlowSummary(f.Actions)
	return nil
}

// callDraftTMUpdate is the stored form of a flow's tm_update: empty for a flow
// that has never been updated, otherwise RFC 3339 with nanoseconds in UTC.
func callDraftTMUpdate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// callDraftFlowSummary lists the action types of a flow, and the destination
// addresses of its connect and call actions, so the approver sees where the
// flow itself dials. An action whose option cannot be read contributes its
// type only. Actions inside a call action's own actions list and flows
// referenced by flow_id are not followed.
func callDraftFlowSummary(actions []fmaction.Action) ([]string, []commonaddress.Address) {
	types := []string{}
	var dests []commonaddress.Address
	for _, a := range actions {
		types = append(types, string(a.Type))
		if a.Type != fmaction.TypeConnect && a.Type != fmaction.TypeCall {
			continue
		}
		raw, err := json.Marshal(a.Option)
		if err != nil {
			continue
		}
		var opt struct {
			Destinations []commonaddress.Address `json:"destinations"`
		}
		if err := json.Unmarshal(raw, &opt); err != nil {
			continue
		}
		dests = append(dests, opt.Destinations...)
	}
	return types, dests
}

// callDraftShapeStillValid reports whether a stored call draft passes the
// shape validation again.
func callDraftShapeStillValid(d *CallDraft) bool {
	return d != nil && validateCallDraftShape(d) == nil
}

// callDraftFlowBindingValid reports whether the flow of a flow_id draft is the
// flow the agent was shown: it exists, belongs to the customer and has the same
// tm_update. A draft without a flow_id has nothing to bind. A flow that is
// gone or foreign is (false, nil); a read error that is not "not found" is
// returned so the caller can fail before the draft is claimed.
func (h *aicallHandler) callDraftFlowBindingValid(ctx context.Context, c *aicall.AIcall, d *CallDraft) (bool, error) {
	if d.FlowID == "" {
		return true, nil
	}
	f, err := h.reqHandler.FlowV1FlowGet(ctx, uuid.FromStringOrNil(d.FlowID))
	if err != nil {
		var ve *cerrors.VoipbinError
		if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	if f == nil || f.CustomerID != c.CustomerID {
		return false, nil
	}
	return callDraftTMUpdate(f.TMUpdate) == d.FlowTMUpdate, nil
}

// callDraftCreateArguments rebuilds the arguments of the create_call tool from
// a stored draft.
func callDraftCreateArguments(d *CallDraft) ([]byte, error) {
	return json.Marshal(&struct {
		FlowID       string                  `json:"flow_id,omitempty"`
		Actions      []fmaction.Action       `json:"actions,omitempty"`
		Source       *commonaddress.Address  `json:"source,omitempty"`
		Destinations []commonaddress.Address `json:"destinations"`
		Anonymous    string                  `json:"anonymous,omitempty"`
		Variables    map[string]string       `json:"variables,omitempty"`
	}{
		FlowID:       d.FlowID,
		Actions:      d.Actions,
		Source:       d.Source,
		Destinations: d.Destinations,
		Anonymous:    d.Anonymous,
		Variables:    d.Variables,
	})
}
