package confighandler_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/confighandler"
)

func Test_Secret_masking(t *testing.T) {
	const value = "SUPER-SECRET-VALUE"
	s := confighandler.NewSecret(value)

	type holder struct {
		Key confighandler.Secret
		M   map[string]confighandler.Secret
	}
	h := holder{Key: s, M: map[string]confighandler.Secret{"k": s}}

	outputs := map[string]string{
		"String":    s.String(),
		"GoString":  s.GoString(),
		"%v":        fmt.Sprintf("%v", s),
		"%+v":       fmt.Sprintf("%+v", s),
		"%#v":       fmt.Sprintf("%#v", s),
		"%s":        fmt.Sprintf("%s", s), //nolint:gosimple // checks the verb
		"%q":        fmt.Sprintf("%q", s),
		"%x":        fmt.Sprintf("%x", s),
		"%d":        fmt.Sprintf("%d", s),
		"ptr %v":    fmt.Sprintf("%v", &s),
		"struct %v": fmt.Sprintf("%v", h),
		"struct +v": fmt.Sprintf("%+v", h),
		"struct #v": fmt.Sprintf("%#v", h),
		"Sprint":    fmt.Sprint(s),
		"Sprintln":  fmt.Sprintln(s),
	}

	j, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	outputs["json struct"] = string(j)

	text, err := s.MarshalText()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	outputs["text"] = string(text)

	for name, out := range outputs {
		if strings.Contains(out, value) {
			t.Errorf("Wrong match. %s leaks the value: %s", name, out)
		}
		if !strings.Contains(out, "[REDACTED]") {
			t.Errorf("Wrong match. %s does not show the mask: %s", name, out)
		}
	}

	if s.Reveal() != value {
		t.Errorf("Wrong match. Reveal expect: %q, got: %q", value, s.Reveal())
	}

	var zero confighandler.Secret
	if fmt.Sprintf("%v", zero) != "[REDACTED]" || zero.Reveal() != "" {
		t.Errorf("Wrong match. the zero value must be masked too")
	}
}
