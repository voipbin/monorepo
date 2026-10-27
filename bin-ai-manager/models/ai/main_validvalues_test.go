package ai

import (
	"reflect"
	"testing"
)

func Test_Type_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		input  Type
		expect bool
	}{
		{name: "none is valid", input: TypeNone, expect: true},
		{name: "normal is valid", input: TypeNormal, expect: true},
		{name: "insight is valid", input: TypeInsight, expect: true},
		{name: "unknown value", input: Type("unknown"), expect: false},
		{name: "case differs", input: Type("Normal"), expect: false},
		{name: "surrounding space", input: Type(" normal"), expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsValid(); got != tt.expect {
				t.Errorf("IsValid(%q) = %v, expect: %v", tt.input, got, tt.expect)
			}
		})
	}
}

func Test_Type_ValidValues(t *testing.T) {
	expect := []string{"insight", "normal"}

	// the receiver value does not influence the result
	for _, recv := range []Type{TypeNone, TypeNormal, Type("unknown")} {
		got := recv.ValidValues()
		if !reflect.DeepEqual(got, expect) {
			t.Errorf("ValidValues() on %q = %v, expect: %v", recv, got, expect)
		}
	}
}
