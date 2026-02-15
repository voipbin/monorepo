package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	ememail "monorepo/bin-email-manager/models/email"
	fmaction "monorepo/bin-flow-manager/models/action"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
		{
			name: "Empty NextId string",
			input: openapi_server.FlowManagerAction{
				Id:        "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				NextId:    stringPtr(""),
				Option:    nil,
				TmExecute: nil,
				Type:      "example",
			},
			expected: fmaction.Action{
				ID:        uuid.Must(uuid.FromString("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")),
				NextID:    uuid.Nil,
				Type:      "example",
				Option:    nil,
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

func TestNewServer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	srv := NewServer(mockSH)

	if srv == nil {
		t.Error("NewServer returned nil")
	}

	// Verify it's the correct type
	if _, ok := srv.(openapi_server.ServerInterface); !ok {
		t.Error("NewServer did not return ServerInterface")
	}
}

func TestConvertEmailManagerEmailAttachment(t *testing.T) {
	testID := "d152e69e-105b-11ee-b395-eb18426de979"

	tests := []struct {
		name   string
		input  openapi_server.EmailManagerEmailAttachment
		expect ememail.Attachment
	}{
		{
			name: "storage attachment",
			input: openapi_server.EmailManagerEmailAttachment{
				ReferenceType: "storage",
				ReferenceId:   testID,
			},
			expect: ememail.Attachment{
				ReferenceType: ememail.AttachmentReferenceType("storage"),
				ReferenceID:   uuid.FromStringOrNil(testID),
			},
		},
		{
			name: "file attachment",
			input: openapi_server.EmailManagerEmailAttachment{
				ReferenceType: "file",
				ReferenceId:   testID,
			},
			expect: ememail.Attachment{
				ReferenceType: ememail.AttachmentReferenceType("file"),
				ReferenceID:   uuid.FromStringOrNil(testID),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertEmailMamagerEmailAttachment(tt.input)

			if !reflect.DeepEqual(result, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expect, result)
			}
		})
	}
}

func TestStringPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"normal string", "test"},
		{"long string", "this is a longer test string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringPtr(tt.input)

			if result == nil {
				t.Error("stringPtr returned nil")
			}
			if *result != tt.input {
				t.Errorf("Wrong value. expect: %v, got: %v", tt.input, *result)
			}
		})
	}
}

func TestGenerateListResponse(t *testing.T) {
	type TestStruct struct {
		ID   string
		Name string
	}

	tests := []struct {
		name            string
		items           []*TestStruct
		nextToken       string
		expectNextToken string
	}{
		{
			name: "with items",
			items: []*TestStruct{
				{ID: "1", Name: "Test1"},
				{ID: "2", Name: "Test2"},
			},
			nextToken:       "next_token_value",
			expectNextToken: "next_token_value",
		},
		{
			name:            "empty items",
			items:           []*TestStruct{},
			nextToken:       "next_token_value",
			expectNextToken: "",
		},
		{
			name:            "nil items",
			items:           nil,
			nextToken:       "next_token_value",
			expectNextToken: "",
		},
		{
			name: "single item",
			items: []*TestStruct{
				{ID: "1", Name: "Test1"},
			},
			nextToken:       "single_next",
			expectNextToken: "single_next",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateListResponse(tt.items, tt.nextToken)

			if len(result.Result) != len(tt.items) {
				t.Errorf("Wrong result length. expect: %v, got: %v", len(tt.items), len(result.Result))
			}

			if result.NextPageToken == nil {
				t.Error("NextPageToken is nil")
			} else if *result.NextPageToken != tt.expectNextToken {
				t.Errorf("Wrong next token. expect: %v, got: %v", tt.expectNextToken, *result.NextPageToken)
			}
		})
	}
}

func TestStructToFilteredMap(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email,omitempty"`
	}

	tests := []struct {
		name      string
		input     any
		expectErr bool
		expectMap map[string]any
	}{
		{
			name: "valid struct",
			input: TestStruct{
				Name:  "John",
				Age:   30,
				Email: "john@example.com",
			},
			expectErr: false,
			expectMap: map[string]any{
				"name":  "John",
				"age":   float64(30), // JSON numbers unmarshal to float64
				"email": "john@example.com",
			},
		},
		{
			name: "struct with omitempty empty field",
			input: TestStruct{
				Name: "Jane",
				Age:  25,
			},
			expectErr: false,
			expectMap: map[string]any{
				"name": "Jane",
				"age":  float64(25),
			},
		},
		{
			name: "map input",
			input: map[string]any{
				"key": "value",
				"num": 42,
			},
			expectErr: false,
			expectMap: map[string]any{
				"key": "value",
				"num": float64(42),
			},
		},
		{
			name:      "invalid input - chan",
			input:     make(chan int),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := structToFilteredMap(tt.input)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectErr && !reflect.DeepEqual(result, tt.expectMap) {
				t.Errorf("Wrong map.\nexpect: %+v\ngot: %+v", tt.expectMap, result)
			}
		})
	}
}
