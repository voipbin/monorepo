package engine_openai_handler

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	fmaction "monorepo/bin-flow-manager/models/action"
)

var (
	tools = []openai.Tool{
		toolConnect,
	}
)

var (
	toolConnect = openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "connect",
			Description: "creates a new call to the destinations and connects to them",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type":        "string",
								"description": "one of agent/conference/email/extension/line/sip/tel",
							},
							"target": map[string]any{
								"type":        "string",
								"description": "address endpoint",
							},
							"target_name": map[string]any{
								"type":        "string",
								"description": "address's name",
							},
						},
						"required": []string{"type", "target"},
					},
					"destinations": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type": map[string]any{
									"type":        "string",
									"description": "one of agent/conference/email/extension/line/sip/tel",
								},
								"target": map[string]any{
									"type":        "string",
									"description": "address endpoint",
								},
								"target_name": map[string]any{
									"type":        "string",
									"description": "address's name",
								},
							},
							"required": []string{"type", "target"},
						},
					},
				},
				"required": []string{"destinations"},
			},
		},
	}
)

func (h *engineOpenaiHandler) toolHandle(actionName string, actionOption []byte) (*fmaction.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "toolHandle",
		"action_name":   actionName,
		"action_option": string(actionOption),
	})
	log.Debugf("Handling the tool action. action_name: %s, action_option: %s", actionName, actionOption)

	switch fmaction.Type(actionName) {
	case fmaction.TypeConnect:
		var tmpOpt fmaction.OptionConnect
		if errUnmarshal := json.Unmarshal(actionOption, &tmpOpt); errUnmarshal != nil {
			return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the tool option correctly. action_name: %s", actionName)
		}

		opt := fmaction.ConvertOption(tmpOpt)
		res := fmaction.Action{
			Type:   fmaction.TypeConnect,
			Option: opt,
		}
		return &res, nil

	default:
		return nil, fmt.Errorf("unsupported action type. action_type: %s", actionName)
	}
}
