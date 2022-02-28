package action

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
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

			[]byte(`{"machine_handle": "hangup"}`),

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

func Test_marshalOptionBranch(t *testing.T) {
	type test struct {
		name string

		option []byte

		expectRes OptionBranch
	}

	tests := []test{
		{
			"normal",

			[]byte(`{"default_index": 1, "target_indexes":{"1": 2, "2": 3}}`),

			OptionBranch{
				DefaultIndex: 1,
				TargetIndexes: map[string]int{
					"1": 2,
					"2": 3,
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

			[]byte(`{"length": 5, "key": "#", "false_target_index": 3}`),

			OptionConditionCallDigits{
				Length:           5,
				Key:              "#",
				FalseTargetIndex: 3,
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

			[]byte(`{"status": "ringing", "false_target_index": 3}`),

			OptionConditionCallStatus{
				Status:           OptionConditionCallStatusStatusRinging,
				FalseTargetIndex: 3,
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
