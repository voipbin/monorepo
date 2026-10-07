package direct

import (
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_MaskHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"short unchanged", "direct.abc", "direct.abc"},
		{"exactly 12 unchanged", "direct.abcde", "direct.abcde"},
		{"long truncated", "direct.0123456789ab", "direct.01234..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskHash(tt.input); got != tt.want {
				t.Errorf("MaskHash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func Test_LogFields(t *testing.T) {
	d := &Direct{
		Identity:     commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
		ResourceType: ResourceTypeAgent,
		ResourceID:   uuid.Must(uuid.NewV4()),
		Hash:         "direct.SECRET-SENTINEL-HASH-1234",
	}

	f := d.LogFields()
	for _, key := range []string{"direct_id", "customer_id", "resource_type", "resource_id"} {
		if _, ok := f[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	for k, v := range f {
		if strings.Contains(k, "hash") {
			t.Errorf("unexpected hash key %q", k)
		}
		if s, ok := v.(string); ok && strings.Contains(s, "SECRET-SENTINEL") {
			t.Errorf("hash leaked into field %q", k)
		}
	}

	var nilDirect *Direct
	if got := nilDirect.LogFields(); len(got) != 0 {
		t.Errorf("nil receiver should return empty fields, got %v", got)
	}
}
