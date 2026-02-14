package billing

import (
	"testing"
)

func Test_GetCostInfo(t *testing.T) {

	tests := []struct {
		name string

		costType CostType

		expectTokenPerUnit  int
		expectCreditPerUnit float32
	}{
		{
			name:                "call_pstn_outgoing - credit only",
			costType:            CostTypeCallPSTNOutgoing,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNOutgoing,
		},
		{
			name:                "call_pstn_incoming - credit only",
			costType:            CostTypeCallPSTNIncoming,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNIncoming,
		},
		{
			name:                "call_vn - token + credit overflow",
			costType:            CostTypeCallVN,
			expectTokenPerUnit:  DefaultTokenPerUnitCallVN,
			expectCreditPerUnit: DefaultCreditPerUnitCallVN,
		},
		{
			name:                "call_extension - free",
			costType:            CostTypeCallExtension,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "call_direct_ext - free",
			costType:            CostTypeCallDirectExt,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "sms - token + credit overflow",
			costType:            CostTypeSMS,
			expectTokenPerUnit:  DefaultTokenPerUnitSMS,
			expectCreditPerUnit: DefaultCreditPerUnitSMS,
		},
		{
			name:                "number - credit only",
			costType:            CostTypeNumber,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "number_renew - credit only",
			costType:            CostTypeNumberRenew,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "none - zero rates",
			costType:            CostTypeNone,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "unknown - zero rates",
			costType:            CostType("unknown"),
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTokenPerUnit, gotCreditPerUnit := GetCostInfo(tt.costType)
			if gotTokenPerUnit != tt.expectTokenPerUnit {
				t.Errorf("Wrong tokenPerUnit. expect: %d, got: %d", tt.expectTokenPerUnit, gotTokenPerUnit)
			}
			if gotCreditPerUnit != tt.expectCreditPerUnit {
				t.Errorf("Wrong creditPerUnit. expect: %f, got: %f", tt.expectCreditPerUnit, gotCreditPerUnit)
			}
		})
	}
}
