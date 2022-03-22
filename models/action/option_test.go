package action

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

func Test_marshalOptionAgentCall(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionAgentCall
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"agent_id": "40fa82a8-951b-11ec-ace0-9fa349fe2070"}`),

			OptionAgentCall{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionAMD(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionAMD
	}

	tests := []test{
		{
			"machine handle hangup",

			[]byte(`{"machine_handle": "hangup"}`),

			OptionAMD{
				MachineHandle: OptionAMDMachineHandleTypeHangup,
			},
		},
		{
			"machine handle continue",

			[]byte(`{"machine_handle": "continue"}`),

			OptionAMD{
				MachineHandle: OptionAMDMachineHandleTypeContinue,
			},
		},
		{
			"machine handle continue with async true",

			[]byte(`{"machine_handle": "continue", "async": true}`),

			OptionAMD{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionAnswer(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionAnswer
	}

	tests := []test{
		{
			"normal",

			[]byte(`{}`),

			OptionAnswer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionAnswer{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionBeep(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionBeep
	}

	tests := []test{
		{
			"normal",

			[]byte(`{}`),

			OptionBeep{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := OptionBeep{}
			if err := json.Unmarshal(tt.option, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionBranch(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionBranch
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"default_target_id": "fd73c1b4-9841-11ec-bc63-df666ba736e8", "target_ids":{"1": "13fce870-9842-11ec-a83f-970e9052be06", "2": "1428ec22-9842-11ec-92d4-2ff427b3bb21"}}`),

			OptionBranch{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionCall(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionCall
	}

	tests := []test{
		{
			"flow id set",

			[]byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}, {"type": "tel", "target": "+821100000003"}], "flow_id": "5ba29abc-a93b-11ec-ae94-6b77822f1a16"}`),

			OptionCall{
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   cmaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				FlowID: uuid.FromStringOrNil("5ba29abc-a93b-11ec-ae94-6b77822f1a16"),
			},
		},
		{
			"actions set",

			[]byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}, {"type": "tel", "target": "+821100000003"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),

			OptionCall{
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   cmaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				Actions: []Action{
					{
						Type: TypeAnswer,
					},
					{
						Type:   TypeTalk,
						Option: []byte(`{"text": "hello world"}`),
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionConfbridgeJoin(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionConfbridgeJoin
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"confbridge_id": "1eba27a4-979e-11ec-989d-2b0bbc04a661"}`),

			OptionConfbridgeJoin{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionCallDigits(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionConditionCallDigits
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"length": 5, "key": "#", "false_target_id": "e998777a-9841-11ec-a7e3-3396ba072ea6"}`),

			OptionConditionCallDigits{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshalOptionConditionCallStatus(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionConditionCallStatus
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"status": "ringing", "false_target_id": "bcc57e5a-9841-11ec-b4ed-df97ae826297"}`),

			OptionConditionCallStatus{
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

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
