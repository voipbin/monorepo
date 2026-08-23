package customerdomainhandler

import (
	"strings"
	"testing"
)

func Test_generateLabel_charsetAndLength(t *testing.T) {
	for i := 0; i < 1000; i++ {
		label, err := generateLabel()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if len(label) != labelLength {
			t.Fatalf("Wrong match. expect length: %d, got: %d (%s)", labelLength, len(label), label)
		}

		for _, c := range label {
			if !strings.ContainsRune(labelCharset, c) {
				t.Fatalf("Wrong match. character outside base36 lowercase charset: %q in %s", c, label)
			}
		}

		if isReservedLabel(label) {
			t.Fatalf("Wrong match. generated a reserved label: %s", label)
		}
	}
}

func Test_generateLabel_randomness(t *testing.T) {
	// 100 draws from a 1.68M space colliding would indicate broken randomness
	seen := map[string]int{}
	for i := 0; i < 100; i++ {
		label, err := generateLabel()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		seen[label]++
	}

	if len(seen) < 95 {
		t.Errorf("Wrong match. expect: >= 95 distinct labels out of 100, got: %d", len(seen))
	}
}

func Test_isReservedLabel(t *testing.T) {
	tests := []struct {
		name string

		label string

		expectRes bool
	}{
		{"pstn is reserved", "pstn", true},
		{"sip is reserved", "sip", true},
		{"echo is reserved", "echo", true},
		{"reg is reserved", "reg", true},
		{"www is reserved", "www", true},
		{"api is reserved", "api", true},
		{"normal label", "ab12", false},
		{"empty", "", false},
		{"case sensitive (labels are lowercase)", "PSTN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isReservedLabel(tt.label); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_reservedLabels_complete(t *testing.T) {
	// the reserved list is a design contract ({pstn, sip, echo, reg, www, api})
	expect := []string{"pstn", "sip", "echo", "reg", "www", "api"}

	if len(reservedLabels) != len(expect) {
		t.Errorf("Wrong match. expect: %d reserved labels, got: %d", len(expect), len(reservedLabels))
	}
	for _, label := range expect {
		if _, ok := reservedLabels[label]; !ok {
			t.Errorf("Wrong match. expect reserved: %s, got: missing", label)
		}
	}
}
