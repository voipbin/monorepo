package envsecret

import "testing"

func Test_Lookup(t *testing.T) {
	tests := []struct {
		name string

		set   bool
		value string

		expectValue string
		expectState State
	}{
		{"unset", false, "", "", Unset},
		{"empty", true, "", "", Unset},
		{"spaces only", true, "  \t\n", "", Unset},
		{"plain value", true, "abc123", "abc123", Set},
		{"surrounding whitespace is removed", true, " abc123\n", "abc123", Set},
		{"placeholder", true, "[[BIN_MANAGER__X]]", "", Placeholder},
		{"placeholder with leading space", true, "  [[X]]", "", Placeholder},
		{"brackets in the middle are a value", true, "ab[[cd", "ab[[cd", Set},
		{"single bracket is a value", true, "[abc", "[abc", Set},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "ENVSECRET_TEST_VARIABLE"
			if tt.set {
				t.Setenv(key, tt.value)
			}

			gotValue, gotState := Lookup(key)
			if gotValue != tt.expectValue || gotState != tt.expectState {
				t.Errorf("Wrong match. expect: (%q, %v), got: (%q, %v)", tt.expectValue, tt.expectState, gotValue, gotState)
			}
		})
	}
}

func Test_State_String(t *testing.T) {
	tests := []struct {
		state  State
		expect string
	}{
		{Unset, "unset"},
		{Set, "set"},
		{Placeholder, "placeholder"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expect {
			t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.state.String())
		}
	}
}
