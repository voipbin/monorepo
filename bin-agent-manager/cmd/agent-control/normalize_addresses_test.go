package main

import (
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func Test_normalizeAddresses(t *testing.T) {
	tests := []struct {
		name string

		addresses []commonaddress.Address

		expectChanged bool
	}{
		{
			name: "tel with punctuation is canonicalized",
			addresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+1-555-0100"},
			},
			expectChanged: true,
		},
		{
			name: "already canonical tel is unchanged",
			addresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: mustNorm(commonaddress.TypeTel, "+1-555-0100")},
			},
			expectChanged: false,
		},
		{
			name: "extension (opaque) is unchanged",
			addresses: []commonaddress.Address{
				{Type: commonaddress.TypeExtension, Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71"},
			},
			expectChanged: false,
		},
		{
			name: "sip host token is lowercased, params preserved",
			addresses: []commonaddress.Address{
				{Type: commonaddress.TypeSIP, Target: "Alice@EXAMPLE.com;transport=TCP"},
			},
			expectChanged: true,
		},
		{
			name: "mixed types: tel changes, extension stays",
			addresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+1 (555) 0100"},
				{Type: commonaddress.TypeExtension, Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71"},
			},
			expectChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAddresses(tt.addresses)

			// per-element it must equal NormalizeTarget applied to that element
			for i := range tt.addresses {
				want, _ := commonaddress.NormalizeTarget(tt.addresses[i].Type, tt.addresses[i].Target)
				if got[i].Target != want {
					t.Errorf("element %d: got %q, want %q", i, got[i].Target, want)
				}
				if got[i].Type != tt.addresses[i].Type {
					t.Errorf("element %d: type changed from %s to %s", i, tt.addresses[i].Type, got[i].Type)
				}
			}

			if addressesDiffer(tt.addresses, got) != tt.expectChanged {
				t.Errorf("addressesDiffer mismatch. expect changed=%v", tt.expectChanged)
			}
		})
	}
}

func Test_normalizeAddresses_doesNotMutateInput(t *testing.T) {
	input := []commonaddress.Address{
		{Type: commonaddress.TypeTel, Target: "+1-555-0100"},
	}
	original := input[0].Target

	_ = normalizeAddresses(input)

	if input[0].Target != original {
		t.Errorf("input was mutated: got %q, want %q", input[0].Target, original)
	}
}

func Test_collisionDetection(t *testing.T) {
	// two distinct raw tels that collapse to the same canonical, on two
	// different agents of the same customer => collision
	customerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	agentA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	agentB := uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36")

	rawA, _ := commonaddress.NormalizeTarget(commonaddress.TypeTel, "+1-555-0100")
	rawB, _ := commonaddress.NormalizeTarget(commonaddress.TypeTel, "+15550100")

	if rawA != rawB {
		t.Fatalf("test premise broken: %q != %q", rawA, rawB)
	}

	canonicalOwners := map[string][]uuid.UUID{}
	addrA := commonaddress.Address{Type: commonaddress.TypeTel, Target: rawA}
	addrB := commonaddress.Address{Type: commonaddress.TypeTel, Target: rawB}
	keyA := collisionKey(customerID, addrA)
	keyB := collisionKey(customerID, addrB)
	canonicalOwners[keyA] = append(canonicalOwners[keyA], agentA)
	canonicalOwners[keyB] = append(canonicalOwners[keyB], agentB)

	if keyA != keyB {
		t.Fatalf("collision keys should match for the same canonical: %q != %q", keyA, keyB)
	}

	owners := dedupeUUIDs(canonicalOwners[keyA])
	if len(owners) != 2 {
		t.Errorf("expected 2 distinct owners (a collision), got %d", len(owners))
	}
}

func Test_collisionKey_differentCustomersDoNotCollide(t *testing.T) {
	customerA := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	customerB := uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b")
	addr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}

	if collisionKey(customerA, addr) == collisionKey(customerB, addr) {
		t.Errorf("same canonical target on different customers must not share a collision key")
	}
}

func mustNorm(addressType commonaddress.Type, target string) string {
	res, _ := commonaddress.NormalizeTarget(addressType, target)
	return res
}

func Test_normalizeAddresses_extensionVariants(t *testing.T) {
	canonical := "49b41028-2d8d-11ef-b38d-27dd55f2bb71"

	tests := []struct {
		name   string
		target string

		expectTarget  string
		expectChanged bool
	}{
		{"canonical is unchanged", canonical, canonical, false},
		{"32 hex digits", "49b410282d8d11efb38d27dd55f2bb71", canonical, true},
		{"braces", "{" + canonical + "}", canonical, true},
		{"urn", "urn:uuid:" + canonical, canonical, true},
		{"upper case", "49B41028-2D8D-11EF-B38D-27DD55F2BB71", canonical, true},
		{"not a uuid is kept as it is", "not-a-uuid", "not-a-uuid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: tt.target}}

			got := normalizeAddresses(input)
			if got[0].Target != tt.expectTarget {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTarget, got[0].Target)
			}
			if addressesDiffer(input, got) != tt.expectChanged {
				t.Errorf("Wrong match. expect changed: %v", tt.expectChanged)
			}
		})
	}
}

// A variant that is not the first element must be detected as well, otherwise the agent is skipped by the backfill.
func Test_addressesDiffer_variantAfterOtherAddresses(t *testing.T) {
	tel := commonaddress.Address{Type: commonaddress.TypeTel, Target: mustNorm(commonaddress.TypeTel, "+1-555-0100")}
	canonicalExt := commonaddress.Address{Type: commonaddress.TypeExtension, Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71"}
	variantExt := commonaddress.Address{Type: commonaddress.TypeExtension, Target: "49B41028-2D8D-11EF-B38D-27DD55F2BB71"}

	tests := []struct {
		name          string
		input         []commonaddress.Address
		expectChanged bool
	}{
		{"variant is the second element", []commonaddress.Address{tel, variantExt}, true},
		{"variant is the third element", []commonaddress.Address{tel, canonicalExt, {Type: commonaddress.TypeExtension, Target: "{49b41028-2d8d-11ef-b38d-27dd55f2bb72}"}}, true},
		{"all canonical stays unchanged", []commonaddress.Address{tel, canonicalExt}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAddresses(tt.input)
			if addressesDiffer(tt.input, got) != tt.expectChanged {
				t.Errorf("Wrong match. expect changed: %v", tt.expectChanged)
			}
		})
	}
}

// A variant row and the canonical row of another agent must collide, because that is the trace of the
// registration of an extension that another agent already owns.
func Test_collisionKey_extensionVariantCollidesWithCanonical(t *testing.T) {
	customerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	canonical := "49b41028-2d8d-11ef-b38d-27dd55f2bb71"

	variant := normalizeAddresses([]commonaddress.Address{{Type: commonaddress.TypeExtension, Target: "49b410282d8d11efb38d27dd55f2bb71"}})[0]
	other := commonaddress.Address{Type: commonaddress.TypeExtension, Target: canonical}

	if collisionKey(customerID, variant) != collisionKey(customerID, other) {
		t.Errorf("Wrong match. the collision keys must be equal: %q != %q", collisionKey(customerID, variant), collisionKey(customerID, other))
	}
}
