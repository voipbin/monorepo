package common

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GenerateEndpointExtension(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		extension  string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("685c675e-5706-11ee-87d5-9bb214c12c41"),
			extension:  "testexten",
			expectRes:  "testexten@685c675e-5706-11ee-87d5-9bb214c12c41.registrar.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if errSet := SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			res := GenerateEndpointExtension(tt.customerID, tt.extension)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}

func Test_GenerateRealmExtension(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectRes string
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("bc22cc08-570a-11ee-acf3-537a646d5f2f"),
			expectRes:  "bc22cc08-570a-11ee-acf3-537a646d5f2f.registrar.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			ResetBaseDomainNamesForTest()
			defer ResetBaseDomainNamesForTest()

			if errSet := SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			res := GenerateRealmExtension(tt.customerID)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}

func Test_GenerateRealmTrunk(t *testing.T) {

	tests := []struct {
		name string

		trunkDomain string

		expectRes string
	}{
		{
			name: "normal",

			trunkDomain: "test",
			expectRes:   "test.trunk.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			ResetBaseDomainNamesForTest()
			defer ResetBaseDomainNamesForTest()

			if errSet := SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			res := GenerateRealmTrunkDomain(tt.trunkDomain)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}

func Test_SetBaseDomainNames(t *testing.T) {
	tests := []struct {
		name string

		inputExtBase   string
		inputTrunkBase string

		expectExtBase   string
		expectTrunkBase string

		wantErr      bool
		errSubstring string
	}{
		{
			name:            "Validation Failure (Empty Extension)",
			inputExtBase:    "",
			inputTrunkBase:  "trunk.voipbin.net",
			expectExtBase:   "",
			expectTrunkBase: "",
			wantErr:         true,
			errSubstring:    "base_domain_name_extension cannot be empty",
		},
		{
			name:            "Validation Failure (Empty Trunk)",
			inputExtBase:    "registrar.voipbin.net",
			inputTrunkBase:  "",
			expectExtBase:   "",
			expectTrunkBase: "",
			wantErr:         true,
			errSubstring:    "base_domain_name_trunk cannot be empty",
		},
		{
			name:            "Success (Valid Inputs)",
			inputExtBase:    "registrar.voipbin.net",
			inputTrunkBase:  "trunk.voipbin.net",
			expectExtBase:   "registrar.voipbin.net",
			expectTrunkBase: "trunk.voipbin.net",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetBaseDomainNamesForTest()
			defer ResetBaseDomainNamesForTest()

			err := SetBaseDomainNames(tt.inputExtBase, tt.inputTrunkBase)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetBaseDomainNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("Error message mismatch. expect substring: %s, got: %s", tt.errSubstring, err.Error())
				}
			}

			if !reflect.DeepEqual(baseDomainNameExtension, tt.expectExtBase) {
				t.Errorf("Extension mismatch. expect: %s, got: %s", tt.expectExtBase, baseDomainNameExtension)
			}
			if !reflect.DeepEqual(baseDomainNameTrunk, tt.expectTrunkBase) {
				t.Errorf("Trunk mismatch. expect: %s, got: %s", tt.expectTrunkBase, baseDomainNameTrunk)
			}
		})
	}
}

// Test_SetBaseDomainNames_DuplicateCall verifies the "base domain names have already been initialized and cannot be changed" error.
func Test_SetBaseDomainNames_DuplicateCall(t *testing.T) {
	ResetBaseDomainNamesForTest()
	defer ResetBaseDomainNamesForTest()

	err := SetBaseDomainNames("first.com", "trunk.first.com")
	if err != nil {
		t.Errorf("First call failed: %v", err)
	}

	err = SetBaseDomainNames("second.com", "trunk.second.com")
	if err == nil {
		t.Errorf("Expected error on second call, but got nil")
	} else if err.Error() != "base domain names have already been initialized and cannot be changed" {
		t.Errorf("Expected 'base domain names have already been initialized and cannot be changed' error, got: %v", err)
	}

	if baseDomainNameExtension != "first.com" {
		t.Errorf("Global value changed unexpectedly. got: %s", baseDomainNameExtension)
	}
}
