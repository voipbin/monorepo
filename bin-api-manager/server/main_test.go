package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ConvertFlowManagerAction(t *testing.T) {
	tests := []struct {
		name     string
		input    openapi_server.FlowManagerAction
		expected fmaction.Action
	}{
		{
			name: "Valid input with all fields",
			input: openapi_server.FlowManagerAction{
				Id:        "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				NextId:    stringPtr("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
				Option:    &map[string]interface{}{"key": "value"},
				TmExecute: stringPtr("2025-01-21T17:00:00+09:00"),
				Type:      "example",
			},
			expected: fmaction.Action{
				ID:     uuid.Must(uuid.FromString("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")),
				NextID: uuid.Must(uuid.FromString("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")),
				Type:   "example",
				Option: map[string]any{
					"key": "value",
				},
				TMExecute: timePtr("2025-01-21T17:00:00+09:00"),
			},
		},
		{
			name: "Input with nil NextId and Option",
			input: openapi_server.FlowManagerAction{
				Id:        "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				NextId:    nil,
				Option:    nil,
				TmExecute: stringPtr("2025-01-21T17:00:00+09:00"),
				Type:      "example",
			},
			expected: fmaction.Action{
				ID:        uuid.Must(uuid.FromString("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")),
				NextID:    uuid.Nil,
				Type:      "example",
				Option:    nil,
				TMExecute: timePtr("2025-01-21T17:00:00+09:00"),
			},
		},
		{
			name: "Input with invalid UUIDs",
			input: openapi_server.FlowManagerAction{
				Id:        "invalid-uuid",
				NextId:    stringPtr("invalid-uuid"),
				Option:    &map[string]interface{}{"key": "value"},
				TmExecute: stringPtr("2025-01-21T17:00:00+09:00"),
				Type:      "example",
			},
			expected: fmaction.Action{
				ID:        uuid.Nil,
				NextID:    uuid.Nil,
				Type:      "example",
				Option:    map[string]any{"key": "value"},
				TMExecute: timePtr("2025-01-21T17:00:00+09:00"),
			},
		},
		{
			name: "Input with missing TmExecute",
			input: openapi_server.FlowManagerAction{
				Id:        "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				NextId:    stringPtr("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
				Option:    &map[string]interface{}{"key": "value"},
				TmExecute: nil,
				Type:      "example",
			},
			expected: fmaction.Action{
				ID:        uuid.Must(uuid.FromString("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")),
				NextID:    uuid.Must(uuid.FromString("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")),
				Type:      "example",
				Option:    map[string]any{"key": "value"},
				TMExecute: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertFlowManagerAction(tt.input)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.NextID, result.NextID)
			assert.Equal(t, tt.expected.Type, result.Type)
			if tt.input.Option == nil {
				assert.Equal(t, tt.expected.Option, result.Option)
			}
			//  else {
			// 	assert.JSONEq(t, string(tt.expected.Option), string(result.Option))
			// }
			assert.Equal(t, tt.expected.TMExecute, result.TMExecute)
		})
	}
}

func TestConvertCommonAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    openapi_server.CommonAddress
		expected commonaddress.Address
	}{
		{
			name: "All fields set",
			input: openapi_server.CommonAddress{
				Detail:     stringPtr("This is the detail"),
				Name:       stringPtr("Address Name"),
				Target:     stringPtr("Target Endpoint"),
				TargetName: stringPtr("Target Name"),
				Type:       (*openapi_server.CommonAddressType)(stringPtr("type1")),
			},
			expected: commonaddress.Address{
				Type:       commonaddress.Type("type1"),
				Target:     "Target Endpoint",
				TargetName: "Target Name",
				Name:       "Address Name",
				Detail:     "This is the detail",
			},
		},
		{
			name: "Nil fields",
			input: openapi_server.CommonAddress{
				Detail:     nil,
				Name:       nil,
				Target:     nil,
				TargetName: nil,
				Type:       nil,
			},
			expected: commonaddress.Address{
				Type:       "",
				Target:     "",
				TargetName: "",
				Name:       "",
				Detail:     "",
			},
		},
		{
			name: "Some fields set",
			input: openapi_server.CommonAddress{
				Detail:     stringPtr("Only detail set"),
				Name:       nil,
				Target:     stringPtr("Target Only"),
				TargetName: nil,
				Type:       (*openapi_server.CommonAddressType)(stringPtr("type2")),
			},
			expected: commonaddress.Address{
				Type:       commonaddress.Type("type2"),
				Target:     "Target Only",
				TargetName: "",
				Name:       "",
				Detail:     "Only detail set",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertCommonAddress(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
