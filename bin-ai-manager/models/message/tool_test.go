package message

import "testing"

func TestToolTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ToolType
		expected string
	}{
		{
			name:     "tool_type_function",
			constant: ToolTypeFunction,
			expected: "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestFunctionCallNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant FunctionCallName
		expected string
	}{
		{
			name:     "function_call_name_none",
			constant: FunctionCallNameNone,
			expected: "",
		},
		{
			name:     "function_call_name_connect_call",
			constant: FunctionCallNameConnectCall,
			expected: "connect_call",
		},
		{
			name:     "function_call_name_get_variables",
			constant: FunctionCallNameGetVariables,
			expected: "get_variables",
		},
		{
			name:     "function_call_name_get_aicall_messages",
			constant: FunctionCallNameGetAIcallMessages,
			expected: "get_aicall_messages",
		},
		{
			name:     "function_call_name_send_email",
			constant: FunctionCallNameSendEmail,
			expected: "send_email",
		},
		{
			name:     "function_call_name_send_message",
			constant: FunctionCallNameSendMessage,
			expected: "send_message",
		},
		{
			name:     "function_call_name_set_variables",
			constant: FunctionCallNameSetVariables,
			expected: "set_variables",
		},
		{
			name:     "function_call_name_stop_media",
			constant: FunctionCallNameStopMedia,
			expected: "stop_media",
		},
		{
			name:     "function_call_name_stop_flow",
			constant: FunctionCallNameStopFlow,
			expected: "stop_flow",
		},
		{
			name:     "function_call_name_stop_service",
			constant: FunctionCallNameStopService,
			expected: "stop_service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestToolCall(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		toolType     ToolType
		functionName FunctionCallName
		arguments    string
	}{
		{
			name:         "creates_tool_call_with_all_fields",
			id:           "call_123",
			toolType:     ToolTypeFunction,
			functionName: FunctionCallNameConnectCall,
			arguments:    `{"number": "+1234567890"}`,
		},
		{
			name:         "creates_tool_call_with_send_email",
			id:           "call_456",
			toolType:     ToolTypeFunction,
			functionName: FunctionCallNameSendEmail,
			arguments:    `{"to": "test@example.com", "subject": "Test"}`,
		},
		{
			name:         "creates_tool_call_with_empty_arguments",
			id:           "call_789",
			toolType:     ToolTypeFunction,
			functionName: FunctionCallNameStopService,
			arguments:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &ToolCall{
				ID:   tt.id,
				Type: tt.toolType,
				Function: FunctionCall{
					Name:      tt.functionName,
					Arguments: tt.arguments,
				},
			}

			if tc.ID != tt.id {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.id, tc.ID)
			}
			if tc.Type != tt.toolType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.toolType, tc.Type)
			}
			if tc.Function.Name != tt.functionName {
				t.Errorf("Wrong Function Name. expect: %s, got: %s", tt.functionName, tc.Function.Name)
			}
			if tc.Function.Arguments != tt.arguments {
				t.Errorf("Wrong Function Arguments. expect: %s, got: %s", tt.arguments, tc.Function.Arguments)
			}
		})
	}
}

func TestToolResponse(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		content string
	}{
		{
			name:    "creates_tool_response_with_all_fields",
			id:      "resp_123",
			content: "Tool execution successful",
		},
		{
			name:    "creates_tool_response_with_empty_content",
			id:      "resp_456",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &ToolResponse{
				ID:      tt.id,
				Content: tt.content,
			}

			if tr.ID != tt.id {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.id, tr.ID)
			}
			if tr.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, tr.Content)
			}
		})
	}
}

func TestFunctionCall(t *testing.T) {
	tests := []struct {
		name      string
		funcName  FunctionCallName
		arguments string
	}{
		{
			name:      "creates_function_call_with_arguments",
			funcName:  FunctionCallNameSetVariables,
			arguments: `{"key": "value"}`,
		},
		{
			name:      "creates_function_call_without_arguments",
			funcName:  FunctionCallNameStopFlow,
			arguments: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FunctionCall{
				Name:      tt.funcName,
				Arguments: tt.arguments,
			}

			if fc.Name != tt.funcName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.funcName, fc.Name)
			}
			if fc.Arguments != tt.arguments {
				t.Errorf("Wrong Arguments. expect: %s, got: %s", tt.arguments, fc.Arguments)
			}
		})
	}
}
