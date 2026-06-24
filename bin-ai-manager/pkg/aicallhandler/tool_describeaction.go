package aicallhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/actioncatalog"

	"github.com/sirupsen/logrus"
)

// toolHandleDescribeAction handles the describe_action tool: it returns the
// option-field schema for a given flow action type so the LLM can correctly
// assemble create_call inline actions. It is a pure, customer-agnostic,
// read-only lookup (no RPC, no DB, no aicall data needed).
func (h *aicallHandler) toolHandleDescribeAction(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleDescribeAction",
		"aicall_id": c.ID,
	})

	res := newToolResult(tool.ID)

	var args struct {
		ActionType string `json:"action_type"`
	}
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	rendered, err := actioncatalog.DescribeAction(args.ActionType)
	if err != nil {
		// unknown/empty type: the error message already echoes the value + valid
		// types, so the LLM can self-correct.
		log.Debugf("describe_action lookup failed. action_type: %s, err: %v", args.ActionType, err)
		fillFailed(res, err)
		return res
	}

	log.Debugf("Described action. action_type: %s", args.ActionType)
	fillSuccess(res, "action", args.ActionType, rendered)
	return res
}
