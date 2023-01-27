package domainhandler

import (
	"testing"
)

func Test_isValidDomainName(t *testing.T) {

	type test struct {
		name       string
		domainName string
		expectRes  bool
	}

	tests := []test{
		{
			"normal text",
			"test",
			true,
		},
		{
			"has . in the middle",
			"test.domain",
			true,
		},
		{
			"has wrong prefix -",
			"-test-domain",
			false,
		},
		{
			"has wrong prefix .",
			".test-domain",
			false,
		},
		{
			"has wrong character in the middle #",
			"test#domain",
			false,
		},
		{
			"has - in the middle",
			"test-domain",
			true,
		},
		{
			"too long",
			"testdomaintestdomaintestdomaintestdomaintestdomaintestdomaintestdomaintestdomaintestdomaintestdomain",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := isValidDomainName(tt.domainName)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
