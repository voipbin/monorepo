package action

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_MarshalOptionDigitsSend(t *testing.T) {
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

func Test_MarshalOptionAMD(t *testing.T) {
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
