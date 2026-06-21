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
