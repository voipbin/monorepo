package bridge

import (
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_BridgeMatches_cov(t *testing.T) {
	now := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	later := now.Add(time.Hour)

	base := &Bridge{
		AsteriskID:  "3e:50:6b:43:bb:30",
		ID:          "bridge-1",
		ChannelIDs:  []string{"ch-1"},
		ReferenceID: uuid.FromStringOrNil("f4959208-972c-11ed-be90-6b3eb4bef16d"),
		TMCreate:    &now,
	}

	tests := []struct {
		name   string
		a      *Bridge
		b      *Bridge
		expect bool
	}{
		{
			name:   "same",
			a:      base,
			b:      &Bridge{AsteriskID: base.AsteriskID, ID: base.ID, ChannelIDs: []string{"ch-1"}, ReferenceID: base.ReferenceID, TMCreate: &now},
			expect: true,
		},
		{
			name:   "timestamps are ignored",
			a:      base,
			b:      &Bridge{AsteriskID: base.AsteriskID, ID: base.ID, ChannelIDs: []string{"ch-1"}, ReferenceID: base.ReferenceID, TMCreate: &later, TMUpdate: &later, TMDelete: &later},
			expect: true,
		},
		{
			name:   "different id",
			a:      base,
			b:      &Bridge{AsteriskID: base.AsteriskID, ID: "bridge-2", ChannelIDs: []string{"ch-1"}, ReferenceID: base.ReferenceID},
			expect: false,
		},
		{
			name:   "different channel ids",
			a:      base,
			b:      &Bridge{AsteriskID: base.AsteriskID, ID: base.ID, ChannelIDs: []string{"ch-2"}, ReferenceID: base.ReferenceID},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := tt.a.Matches(tt.b); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_BridgeString_cov(t *testing.T) {
	b := &Bridge{ID: "bridge-string-id", Name: "bridge-string-name"}

	res := b.String()
	if !strings.Contains(res, "bridge-string-id") || !strings.Contains(res, "bridge-string-name") {
		t.Errorf("Wrong match. the string must contain the id and name. got: %s", res)
	}
}
