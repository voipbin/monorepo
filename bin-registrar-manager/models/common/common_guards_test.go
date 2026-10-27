package common

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// recoverEntry runs f and returns the logrus entry it panicked with, or nil when f did not panic.
func recoverEntry(t *testing.T, f func()) (res *logrus.Entry) {
	t.Helper()

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		entry, ok := r.(*logrus.Entry)
		if !ok {
			t.Fatalf("Wrong match. expect: a logrus entry, got: %T (%v)", r, r)
		}
		res = entry
	}()

	f()
	return nil
}

func Test_baseDomainNameAccessors_uninitialized(t *testing.T) {
	ResetBaseDomainNamesForTest()
	t.Cleanup(ResetBaseDomainNamesForTest)

	tests := []struct {
		name string
		call func()

		expectMessage string
	}{
		{
			name:          "extension base domain name",
			call:          func() { _ = BaseDomainNameExtension() },
			expectMessage: "baseDomainNameExtension is not initialized",
		},
		{
			name:          "trunk base domain name",
			call:          func() { _ = GenerateRealmTrunkDomain("case") },
			expectMessage: "baseDomainNameTrunk is not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := recoverEntry(t, tt.call)
			if entry == nil {
				t.Fatalf("Wrong match. expect: a panic, got: none")
			}
			if !strings.Contains(entry.Message, tt.expectMessage) {
				t.Errorf("Wrong match. expect a message containing %q, got: %q", tt.expectMessage, entry.Message)
			}
		})
	}
}

func Test_SetBaseDomainNames_domainLength(t *testing.T) {
	// labels of one character are valid for the regular expression, so only the
	// total length decides.
	atLimit := strings.Repeat("a.", 126) + "a"
	justOverLimit := strings.Repeat("a.", 126) + "ab"
	overLimit := strings.Repeat("a.", 127) + "a"
	if len(atLimit) != 253 || len(justOverLimit) != 254 || len(overLimit) != 255 {
		t.Fatalf("Wrong fixture. lengths: %d, %d, %d", len(atLimit), len(justOverLimit), len(overLimit))
	}
	valid := "trunk.example.test"

	tests := []struct {
		name string

		extensionBase string
		trunkBase     string

		expectErr string
	}{
		{"extension at the limit", atLimit, valid, ""},
		{"extension one character over the limit", justOverLimit, valid, "extension base domain name is invalid"},
		{"extension over the limit", overLimit, valid, "extension base domain name is invalid"},
		{"trunk over the limit", valid, overLimit, "trunk base domain name is invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetBaseDomainNamesForTest()
			t.Cleanup(ResetBaseDomainNamesForTest)

			err := SetBaseDomainNames(tt.extensionBase, tt.trunkBase)
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}
