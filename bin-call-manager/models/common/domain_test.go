package common

import (
	"testing"
)

func Test_ParseSIPURI(t *testing.T) {

	type test struct {
		name string

		uri string

		expectResExtension string
		expectResDomain    string
	}

	tests := []test{
		{
			name: "normal short realm",

			uri: "test11@ab12.reg.voipbin.net",

			expectResExtension: "test11",
			expectResDomain:    "ab12.reg.voipbin.net",
		},
		{
			name: "legacy uuid realm",

			uri: "test11@1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",

			expectResExtension: "test11",
			expectResDomain:    "1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",
		},
		{
			name: "arbitrary domain. no suffix knowledge",

			uri: "someuser@example.com",

			expectResExtension: "someuser",
			expectResDomain:    "example.com",
		},
		{
			name: "multiple at signs. first split wins",

			uri: "user@domain1@domain2",

			expectResExtension: "user",
			expectResDomain:    "domain1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension, domain, err := ParseSIPURI(tt.uri)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if extension != tt.expectResExtension {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResExtension, extension)
			}

			if domain != tt.expectResDomain {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResDomain, domain)
			}
		})
	}
}

func Test_ParseSIPURI_error(t *testing.T) {

	type test struct {
		name string

		uri string
	}

	tests := []test{
		{
			name: "no at sign",

			uri: "test11.reg.voipbin.net",
		},
		{
			name: "empty uri",

			uri: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension, domain, err := ParseSIPURI(tt.uri)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok, extension: %s, domain: %s", extension, domain)
			}
		})
	}
}
