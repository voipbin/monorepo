package aipromptproposalhandler

import (
	"encoding/json"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/message"
)

type evalDims struct {
	Helpfulness    evalDimReason
	Accuracy       evalDimReason
	Tone           evalDimReason
	GoalCompletion evalDimReason
	ToolUsageR     string
	Summary        string
}

type evalDimReason struct {
	Reason string
}

func parseAuditEvaluation(raw json.RawMessage) (*evalDims, error) {
	if len(raw) == 0 {
		return &evalDims{}, nil
	}
	var blob struct {
		Dimensions struct {
			Helpfulness struct {
				Reason string `json:"reason"`
			} `json:"helpfulness"`
			Accuracy struct {
				Reason string `json:"reason"`
			} `json:"accuracy"`
			Tone struct {
				Reason string `json:"reason"`
			} `json:"tone"`
			GoalCompletion struct {
				Reason string `json:"reason"`
			} `json:"goal_completion"`
			ToolUsage *struct {
				Reason string `json:"reason"`
			} `json:"tool_usage"`
		} `json:"dimensions"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(raw, &blob); err != nil {
		return nil, fmt.Errorf("parseAuditEvaluation: %w", err)
	}
	out := &evalDims{
		Helpfulness:    evalDimReason{Reason: blob.Dimensions.Helpfulness.Reason},
		Accuracy:       evalDimReason{Reason: blob.Dimensions.Accuracy.Reason},
		Tone:           evalDimReason{Reason: blob.Dimensions.Tone.Reason},
		GoalCompletion: evalDimReason{Reason: blob.Dimensions.GoalCompletion.Reason},
		Summary:        blob.Summary,
	}
	if blob.Dimensions.ToolUsage != nil {
		out.ToolUsageR = blob.Dimensions.ToolUsage.Reason
	}
	return out, nil
}

func buildTranscript(msgs []*message.Message, maxChars int) string {
	var sb strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&sb, "[%s]: %s\n", m.Role, m.Content)
		if sb.Len() >= maxChars {
			break
		}
	}
	out := sb.String()
	if len(out) > maxChars {
		out = out[:maxChars] + "...(truncated)"
	}
	return out
}
