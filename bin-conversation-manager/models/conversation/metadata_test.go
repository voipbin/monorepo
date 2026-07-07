package conversation

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
)

func TestMetadata_ContactCaseID_JSONRoundTrip(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000001")

	tests := []struct {
		name   string
		input  string
		expect Metadata
	}{
		{
			"contact_case_id set",
			`{"contact_case_id":"f1b2c3d4-0001-0001-0001-000000000001"}`,
			Metadata{ContactCaseID: &caseID},
		},
		{
			"contact_case_id absent (nil)",
			`{}`,
			Metadata{ContactCaseID: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Metadata
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if tt.expect.ContactCaseID == nil {
				if got.ContactCaseID != nil {
					t.Errorf("Got %+v, expected nil ContactCaseID", got)
				}
			} else {
				if got.ContactCaseID == nil || *got.ContactCaseID != *tt.expect.ContactCaseID {
					t.Errorf("Got %+v, expected %+v", got, tt.expect)
				}
			}

			b, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			var got2 Metadata
			if err := json.Unmarshal(b, &got2); err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}
			if tt.expect.ContactCaseID == nil {
				if got2.ContactCaseID != nil {
					t.Errorf("Round-trip mismatch. Got %+v, expected nil", got2)
				}
			} else {
				if got2.ContactCaseID == nil || *got2.ContactCaseID != *tt.expect.ContactCaseID {
					t.Errorf("Round-trip mismatch. Got %+v, expected %+v", got2, tt.expect)
				}
			}
		})
	}
}

func TestMetadata_NilContactCaseID_OmitsFieldOnMarshal(t *testing.T) {
	m := Metadata{}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(b) != `{}` {
		t.Errorf("Got %s, expected {}", string(b))
	}
}
