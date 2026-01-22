package callapplication

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestAMDMachineHandleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"amd_machine_handle_hangup", AMDMachineHandleHangup, "hangup"},
		{"amd_machine_handle_continue", AMDMachineHandleContinue, "continue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestAMDStruct(t *testing.T) {
	callID := uuid.Must(uuid.NewV4())

	amd := AMD{
		CallID:        callID,
		MachineHandle: AMDMachineHandleHangup,
		Async:         true,
	}

	if amd.CallID != callID {
		t.Errorf("AMD.CallID = %v, expected %v", amd.CallID, callID)
	}
	if amd.MachineHandle != AMDMachineHandleHangup {
		t.Errorf("AMD.MachineHandle = %v, expected %v", amd.MachineHandle, AMDMachineHandleHangup)
	}
	if amd.Async != true {
		t.Errorf("AMD.Async = %v, expected %v", amd.Async, true)
	}
}

func TestAMDMachineHandles(t *testing.T) {
	tests := []struct {
		name          string
		machineHandle string
		async         bool
	}{
		{"hangup_sync", AMDMachineHandleHangup, false},
		{"hangup_async", AMDMachineHandleHangup, true},
		{"continue_sync", AMDMachineHandleContinue, false},
		{"continue_async", AMDMachineHandleContinue, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amd := AMD{
				MachineHandle: tt.machineHandle,
				Async:         tt.async,
			}
			if amd.MachineHandle != tt.machineHandle {
				t.Errorf("AMD.MachineHandle = %v, expected %v", amd.MachineHandle, tt.machineHandle)
			}
			if amd.Async != tt.async {
				t.Errorf("AMD.Async = %v, expected %v", amd.Async, tt.async)
			}
		})
	}
}
