package engine_openai_handler

import (
	"github.com/sashabaranov/go-openai"
)

var (
	tools = []openai.Tool{
		toolConnect,
		toolMessageSend,
	}
)

var (
	toolConnect = openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "connect",
			Description: `
				Establishes a call from a source endpoint to one or more destination endpoints. 
				Use this when you need to connect a caller to specific endpoints like agents, conferences, or lines. 
				The source and destination types can be agent, conference, extension, sip, or tel. 
				Each endpoint must include a type and target, and optionally a target_name for display purposes.
			`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type":        "string",
								"description": "one of agent/conference/extension/sip/tel",
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

	toolMessageSend = openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "message_send",
			Description: `
				Sends an SMS text message from a source telephone number to one or more destination telephone numbers.
				Use this when you need to deliver SMS messages between phone numbers.
				The source and destination types must be "tel".
			`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type":        "string",
								"enum":        []string{"tel"},
								"description": "must be tel",
							},
							"target": map[string]any{
								"type":        "string",
								"description": "+E.164 formatted phone number",
							},
							"target_name": map[string]any{
								"type":        "string",
								"description": "optional display name for the number",
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
									"enum":        []string{"tel"},
									"description": "must be tel",
								},
								"target": map[string]any{
									"type":        "string",
									"description": "+E.164 formatted phone number",
								},
								"target_name": map[string]any{
									"type":        "string",
									"description": "optional display name for the number",
								},
							},
							"required": []string{"type", "target"},
						},
					},
					"text": map[string]any{
						"type":        "string",
						"description": "SMS message content",
					},
				},
				"required": []string{"destinations", "text"},
			},
		},
	}
)
