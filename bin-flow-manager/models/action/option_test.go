package action

import (
	"encoding/json"
	"reflect"
	"testing"

	amsummary "monorepo/bin-ai-manager/models/summary"
	commonaddress "monorepo/bin-common-handler/models/address"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
)

func Test_ConvertOption(t *testing.T) {
	type test struct {
		name string

		option any

		expectedRes map[string]any
	}

	tests := []test{
		{
			name: "OptionAgentCall",

			option: OptionAgentCall{
				AgentID: uuid.FromStringOrNil("35b8c7b8-111b-11f0-93a9-7b24e0bd5cb6"),
			},

			expectedRes: map[string]any{
				"agent_id": "35b8c7b8-111b-11f0-93a9-7b24e0bd5cb6",
			},
		},
		{
			name: "OptionAISummary",

			option: OptionAISummary{
				OnEndFlowID:   uuid.FromStringOrNil("355e3cda-111b-11f0-94b7-33070885b126"),
				ReferenceType: amsummary.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("358acef8-111b-11f0-acd7-c7f3423f233f"),
				Language:      "en-US",
			},

			expectedRes: map[string]any{
				"on_end_flow_id": "355e3cda-111b-11f0-94b7-33070885b126",
				"reference_type": "call",
				"reference_id":   "358acef8-111b-11f0-acd7-c7f3423f233f",
				"language":       "en-US",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := ConvertOption(tt.option)
			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_ParseOption(t *testing.T) {
	type test struct {
		name string

		option map[string]any
		target any

		expectRes any
	}

	tests := []test{
		{
			name: "OptionAgentCall",

			option: map[string]any{
				"agent_id": "d45f391c-111e-11f0-bfe0-cf4ad671ba02",
			},
			target: &OptionAgentCall{},

			expectRes: &OptionAgentCall{
				AgentID: uuid.FromStringOrNil("d45f391c-111e-11f0-bfe0-cf4ad671ba02"),
			},
		},
		{
			name: "OptionAISummary",

			option: map[string]any{
				"on_end_flow_id": "d48cf8f2-111e-11f0-9eff-634180905812",
				"reference_type": "call",
				"reference_id":   "d4bf1ae4-111e-11f0-bf3d-4bf8bc30c112",
				"language":       "en-US",
			},
			target: &OptionAISummary{},

			expectRes: &OptionAISummary{
				OnEndFlowID:   uuid.FromStringOrNil("d48cf8f2-111e-11f0-9eff-634180905812"),
				ReferenceType: amsummary.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d4bf1ae4-111e-11f0-bf3d-4bf8bc30c112"),
				Language:      "en-US",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if errParse := ParseOption(tt.option, &tt.target); errParse != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errParse)
			}

			if !reflect.DeepEqual(tt.expectRes, tt.target) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.target)
			}
		})
	}
}

func Test_marshalOptionAgentCall(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionAgentCall
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"agent_id": "40fa82a8-951b-11ec-ace0-9fa349fe2070"}`),

			expectedRes: OptionAgentCall{
				AgentID: uuid.FromStringOrNil("40fa82a8-951b-11ec-ace0-9fa349fe2070"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAgentCall{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionAMD(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionAMD
	}

	tests := []test{
		{
			name: "machine handle hangup",

			option: []byte(`{"machine_handle": "hangup"}`),

			expectedRes: OptionAMD{
				MachineHandle: OptionAMDMachineHandleTypeHangup,
			},
		},
		{
			name: "machine handle continue",

			option: []byte(`{"machine_handle": "continue"}`),

			expectedRes: OptionAMD{
				MachineHandle: OptionAMDMachineHandleTypeContinue,
			},
		},
		{
			name: "machine handle continue with async true",

			option: []byte(`{"machine_handle": "continue", "async": true}`),

			expectedRes: OptionAMD{
				MachineHandle: OptionAMDMachineHandleTypeContinue,
				Async:         true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAMD{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionAnswer(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionAnswer
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{}`),

			expectedRes: OptionAnswer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAnswer{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionBeep(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionBeep
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{}`),

			expectedRes: OptionBeep{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionBeep{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionBranch(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionBranch
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"variable": "voipbin.call.destination", "default_target_id": "fd73c1b4-9841-11ec-bc63-df666ba736e8", "target_ids":{"1": "13fce870-9842-11ec-a83f-970e9052be06", "2": "1428ec22-9842-11ec-92d4-2ff427b3bb21"}}`),

			expectedRes: OptionBranch{
				Variable:        "voipbin.call.destination",
				DefaultTargetID: uuid.FromStringOrNil("fd73c1b4-9841-11ec-bc63-df666ba736e8"),
				TargetIDs: map[string]uuid.UUID{
					"1": uuid.FromStringOrNil("13fce870-9842-11ec-a83f-970e9052be06"),
					"2": uuid.FromStringOrNil("1428ec22-9842-11ec-92d4-2ff427b3bb21"),
				},
			},
		},
		{
			name: "has no variable",

			option: []byte(`{"default_target_id": "fd73c1b4-9841-11ec-bc63-df666ba736e8", "target_ids":{"1": "13fce870-9842-11ec-a83f-970e9052be06", "2": "1428ec22-9842-11ec-92d4-2ff427b3bb21"}}`),

			expectedRes: OptionBranch{
				Variable:        "",
				DefaultTargetID: uuid.FromStringOrNil("fd73c1b4-9841-11ec-bc63-df666ba736e8"),
				TargetIDs: map[string]uuid.UUID{
					"1": uuid.FromStringOrNil("13fce870-9842-11ec-a83f-970e9052be06"),
					"2": uuid.FromStringOrNil("1428ec22-9842-11ec-92d4-2ff427b3bb21"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionBranch{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionCall(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionCall
	}

	tests := []test{
		{
			name: "have all",

			option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}, {"type": "tel", "target": "+821100000003"}], "flow_id": "5ba29abc-a93b-11ec-ae94-6b77822f1a16", "early_execution":true}`),

			expectedRes: OptionCall{
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				FlowID:         uuid.FromStringOrNil("5ba29abc-a93b-11ec-ae94-6b77822f1a16"),
				EarlyExecution: true,
			},
		},
		{
			name: "actions set",

			option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}, {"type": "tel", "target": "+821100000003"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),

			expectedRes: OptionCall{
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				Actions: []Action{
					{
						Type: TypeAnswer,
					},
					{
						Type: TypeTalk,
						Option: map[string]any{
							"text": "hello world",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionCall{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConfbridgeJoin(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConfbridgeJoin
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"confbridge_id": "1eba27a4-979e-11ec-989d-2b0bbc04a661"}`),

			expectedRes: OptionConfbridgeJoin{
				ConfbridgeID: uuid.FromStringOrNil("1eba27a4-979e-11ec-989d-2b0bbc04a661"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConfbridgeJoin{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionCallDigits(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConditionCallDigits
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"length": 5, "key": "#", "false_target_id": "e998777a-9841-11ec-a7e3-3396ba072ea6"}`),

			expectedRes: OptionConditionCallDigits{
				Length:        5,
				Key:           "#",
				FalseTargetID: uuid.FromStringOrNil("e998777a-9841-11ec-a7e3-3396ba072ea6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConditionCallDigits{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionCallStatus(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConditionCallStatus
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"status": "ringing", "false_target_id": "bcc57e5a-9841-11ec-b4ed-df97ae826297"}`),

			expectedRes: OptionConditionCallStatus{
				Status:        OptionConditionCallStatusStatusRinging,
				FalseTargetID: uuid.FromStringOrNil("bcc57e5a-9841-11ec-b4ed-df97ae826297"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConditionCallStatus{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConversationSend(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConversationSend
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"conversation_id": "af3620f8-f464-11ec-926e-23a17cd3e34b", "text": "hello world!", "sync": true}`),

			expectedRes: OptionConversationSend{
				ConversationID: uuid.FromStringOrNil("af3620f8-f464-11ec-926e-23a17cd3e34b"),
				Text:           "hello world!",
				Sync:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConversationSend{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionVariableSet(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionVariableSet
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"key": "key 1", "value": "value 1"}`),

			expectedRes: OptionVariableSet{
				Key:   "key 1",
				Value: "value 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionVariableSet{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_OptionWebhookSend(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionWebhookSend
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"test com"}`),

			expectedRes: OptionWebhookSend{
				Sync:     false,
				URI:      "test.com",
				Method:   "POST",
				DataType: "application/json",
				Data:     "test com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionWebhookSend{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionDatetime(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConditionDatetime
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"condition": ">=", "hour": 8, "day": -1, "month": -1, "weekdays": [], "false_target_id": "722c49b0-a976-4671-b946-489be3b1dc23"}`),

			expectedRes: OptionConditionDatetime{
				Condition:     OptionConditionCommonConditionGreaterEqual,
				Minute:        0,
				Hour:          8,
				Day:           -1,
				Month:         -1,
				Weekdays:      []int{},
				FalseTargetID: uuid.FromStringOrNil("722c49b0-a976-4671-b946-489be3b1dc23"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConditionDatetime{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionVariable(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConditionVariable
	}

	tests := []test{
		{
			name: "value type string",

			option: []byte(`{"condition": ">=", "variable":"${voipbin.call.source.target}", "value_type": "string", "value_string": "test", "false_target_id": "ebccdde3-f408-4736-99dc-d37407dc14fb"}`),

			expectedRes: OptionConditionVariable{
				Condition:     OptionConditionCommonConditionGreaterEqual,
				Variable:      "${voipbin.call.source.target}",
				ValueType:     OptionConditionVariableTypeString,
				ValueString:   "test",
				FalseTargetID: uuid.FromStringOrNil("ebccdde3-f408-4736-99dc-d37407dc14fb"),
			},
		},
		{
			name: "value type number",

			option: []byte(`{"condition": ">=", "variable":"${test.number}", "value_type": "number", "value_number": 110.1, "false_target_id": "79cd79c2-6a94-4d4c-8da4-a4edf875788e"}`),

			expectedRes: OptionConditionVariable{
				Condition:     OptionConditionCommonConditionGreaterEqual,
				Variable:      "${test.number}",
				ValueType:     OptionConditionVariableTypeNumber,
				ValueNumber:   110.1,
				FalseTargetID: uuid.FromStringOrNil("79cd79c2-6a94-4d4c-8da4-a4edf875788e"),
			},
		},
		{
			name: "value type length",

			option: []byte(`{"condition": ">=", "variable":"${voipbin.call.source.target}", "value_type": "length", "value_length": 10, "false_target_id": "83b3a0ba-2c1f-4dd3-b045-3ed6ab6a5eb2"}`),

			expectedRes: OptionConditionVariable{
				Condition:     OptionConditionCommonConditionGreaterEqual,
				Variable:      "${voipbin.call.source.target}",
				ValueType:     OptionConditionVariableTypeLength,
				ValueLength:   10,
				FalseTargetID: uuid.FromStringOrNil("83b3a0ba-2c1f-4dd3-b045-3ed6ab6a5eb2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConditionVariable{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionHangup(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionHangup
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"reason": "busy", "reference_id": "daf4b1ae-e95c-4a7f-9bb4-f8f52d68fdeb"}`),

			expectedRes: OptionHangup{
				Reason:      "busy",
				ReferenceID: uuid.FromStringOrNil("daf4b1ae-e95c-4a7f-9bb4-f8f52d68fdeb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionHangup{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshal_OptionConnect(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionConnect
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{
				"source":{"type":"tel","target":"+821100000001"},
				"destinations":[{"type":"tel","target":"+821100000002"},{"type":"tel","target":"+821100000003"}],
				"early_media":true,
				"relay_reason":true
			}`),

			expectedRes: OptionConnect{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				EarlyMedia:  true,
				RelayReason: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionConnect{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshal_OptionAITalk(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionAITalk
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{
				"ai_id":"d1c4f676-a8a5-11ed-85ca-7fe57e970bcd",
				"gender":"female",
				"language":"en-US",
				"duration":6000
			}`),

			expectedRes: OptionAITalk{
				AIID:     uuid.FromStringOrNil("d1c4f676-a8a5-11ed-85ca-7fe57e970bcd"),
				Gender:   "female",
				Language: "en-US",
				Duration: 6000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAITalk{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshal_OptionEmailSend(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionEmailSend
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{
				"destinations": [
					{"type": "email", "target": "test@voipbin.net", "target_name": "test name"}
				],
				"subject": "test subject",
				"content": "test content",
				"attachments": [
					{"reference_type": "recording", "reference_id": "74ae44d2-00f1-11f0-b658-07d12a1ba40c"}
				]
			}`),

			expectedRes: OptionEmailSend{
				Destinations: []commonaddress.Address{
					{
						Type:       commonaddress.TypeEmail,
						Target:     "test@voipbin.net",
						TargetName: "test name",
					},
				},
				Subject: "test subject",
				Content: "test content",
				Attachments: []ememail.Attachment{
					{
						ReferenceType: ememail.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("74ae44d2-00f1-11f0-b658-07d12a1ba40c"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionEmailSend{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshalOptionRecordingStart(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionRecordingStart
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"format": "wav", "end_of_silence": 3, "end_of_key": "1", "duration": 60, "beep_start": true, "on_end_flow_id":"c34194ca-0545-11f0-b0ee-2be0bbe6695e"}`),

			expectedRes: OptionRecordingStart{
				Format:       "wav",
				EndOfSilence: 3,
				EndOfKey:     "1",
				Duration:     60,
				BeepStart:    true,
				OnEndFlowID:  uuid.FromStringOrNil("c34194ca-0545-11f0-b0ee-2be0bbe6695e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionRecordingStart{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_OptionTranscribeStart(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionTranscribeStart
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"language": "en-US", "on_end_flow_id": "ccde4a38-093b-11f0-921a-93245e27ef98"}`),

			expectedRes: OptionTranscribeStart{
				Language:    "en-US",
				OnEndFlowID: uuid.FromStringOrNil("ccde4a38-093b-11f0-921a-93245e27ef98"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionTranscribeStart{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_OptionTranscribeRecording(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionTranscribeRecording
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{"language": "en-US", "on_end_flow_id": "cd02b6a2-093b-11f0-b71b-7bc0ff6efaaf"}`),

			expectedRes: OptionTranscribeRecording{
				Language:    "en-US",
				OnEndFlowID: uuid.FromStringOrNil("cd02b6a2-093b-11f0-b71b-7bc0ff6efaaf"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionTranscribeRecording{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_marshal_OptionAISummary(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectedRes OptionAISummary
	}

	tests := []test{
		{
			name: "normal",

			option: []byte(`{
				"on_end_flow_id":"0d73ca62-0cbd-11f0-8a18-db8b57e75484",
				"reference_type":"call",
				"reference_id":"0daf4c36-0cbd-11f0-b157-f3ce7acc9a2d",
				"language":"en-US"
			}`),

			expectedRes: OptionAISummary{
				OnEndFlowID:   uuid.FromStringOrNil("0d73ca62-0cbd-11f0-8a18-db8b57e75484"),
				ReferenceType: amsummary.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("0daf4c36-0cbd-11f0-b157-f3ce7acc9a2d"),
				Language:      "en-US",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAISummary{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
