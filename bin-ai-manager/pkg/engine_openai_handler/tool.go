package engine_openai_handler

import "github.com/sashabaranov/go-openai"

var (
	tools = []openai.Tool{
		toolConnect,
		// toolEmailSend,
	}
)

var (
	// toolConnect = openai.Tool{
	// 	Type: openai.ToolTypeFunction,
	// 	Function: &openai.FunctionDefinition{
	// 		Name:        "connect",
	// 		Description: "creates a new call to the destinations and connects to them",
	// 		Parameters: map[string]any{
	// 			"type": "object",
	// 			"properties": map[string]any{
	// 				"source": map[string]any{
	// 					"type":        "one of agent/conference/email/extension/line/sip/tel",
	// 					"target":      "address endpoint",
	// 					"target_name": "address's name",
	// 				},
	// 				"destinations": []map[string]any{
	// 					{
	// 						"type":        "one of agent/conference/email/extension/line/sip/tel",
	// 						"target":      "address endpoint",
	// 						"target_name": "address's name",
	// 					},
	// 				},
	// 			},
	// 			"required": []string{"destinations"},
	// 		},
	// 	},
	// }

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

	// toolEmailSend = openai.Tool{
	// 	Type: openai.ToolTypeFunction,
	// 	Function: &openai.FunctionDefinition{
	// 		Name:        "email_send",
	// 		Description: "sends the email.",
	// 		Parameters: map[string]any{
	// 			"type": "object",
	// 			"properties": map[string]any{
	// 				"source": map[string]any{
	// 					"type":        "one of agent/conference/email/extension/line/sip/tel",
	// 					"target":      "address endpoint",
	// 					"target_name": "address's name",
	// 				},
	// 				"destinations": []map[string]any{
	// 					{
	// 						"type":        "one of agent/conference/email/extension/line/sip/tel",
	// 						"target":      "address endpoint",
	// 						"target_name": "address's name",
	// 					},
	// 				},
	// 				"subject": "email subject",
	// 				"content": "email content",
	// 				"attachments": []map[string]any{
	// 					{
	// 						"reference_type": "reference type of attachment resource",
	// 						"reference_id":   "reference id of attachment resource",
	// 					},
	// 				},
	// 			},
	// 			"required": []string{"destinations"},
	// 		},
	// 	},
	// }
)
