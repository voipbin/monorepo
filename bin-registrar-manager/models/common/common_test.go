package common

import (
	"reflect"
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
			"normal",

			uuid.FromStringOrNil("685c675e-5706-11ee-87d5-9bb214c12c41"),
			"testexten",
			"testexten@685c675e-5706-11ee-87d5-9bb214c12c41.registrar.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := GenerateEndpointExtension(tt.customerID, tt.extension)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}

func Test_GenerateRealmExtension(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		expectRes string
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("bc22cc08-570a-11ee-acf3-537a646d5f2f"),
			"bc22cc08-570a-11ee-acf3-537a646d5f2f.registrar.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := GenerateRealmExtension(tt.customerID)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}

func Test_GenerateRealmTrunk(t *testing.T) {

	type test struct {
		name string

		trunkDomain string

		expectRes string
	}

	tests := []test{
		{
			"normal",

			"test",
			"test.trunk.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := GenerateRealmTrunkDomain(tt.trunkDomain)
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})

	}
}
