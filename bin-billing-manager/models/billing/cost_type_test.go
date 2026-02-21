package billing

import (
	"testing"
)

func Test_GetCostInfo(t *testing.T) {

	tests := []struct {
		name string

		costType CostType

		expectMode          CostMode
		expectTokenPerUnit  int64
		expectCreditPerUnit int64
	}{
		{
			name:                "call_pstn_outgoing - credit only",
			costType:            CostTypeCallPSTNOutgoing,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNOutgoing,
		},
		{
			name:                "call_pstn_incoming - credit only",
			costType:            CostTypeCallPSTNIncoming,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNIncoming,
		},
		{
			name:                "call_vn - token first",
			costType:            CostTypeCallVN,
			expectMode:          CostModeTokenFirst,
			expectTokenPerUnit:  DefaultTokenPerUnitCallVN,
			expectCreditPerUnit: DefaultCreditPerUnitCallVN,
		},
		{
			name:                "call_extension - free",
			costType:            CostTypeCallExtension,
			expectMode:          CostModeFree,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "call_direct_ext - free",
			costType:            CostTypeCallDirectExt,
			expectMode:          CostModeFree,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "sms - credit only",
			costType:            CostTypeSMS,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitSMS,
		},
		{
			name:                "email - credit only",
			costType:            CostTypeEmail,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitEmail,
		},
		{
			name:                "number - credit only",
			costType:            CostTypeNumber,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "number_renew - credit only",
			costType:            CostTypeNumberRenew,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "none - disabled",
			costType:            CostTypeNone,
			expectMode:          CostModeDisabled,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "unknown - disabled",
			costType:            CostType("unknown"),
			expectMode:          CostModeDisabled,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCostInfo(tt.costType)
			if got.Mode != tt.expectMode {
				t.Errorf("Wrong Mode. expect: %d, got: %d", tt.expectMode, got.Mode)
			}
			if got.TokenPerUnit != tt.expectTokenPerUnit {
				t.Errorf("Wrong TokenPerUnit. expect: %d, got: %d", tt.expectTokenPerUnit, got.TokenPerUnit)
			}
			if got.CreditPerUnit != tt.expectCreditPerUnit {
				t.Errorf("Wrong CreditPerUnit. expect: %d, got: %d", tt.expectCreditPerUnit, got.CreditPerUnit)
			}
		})
	}
}
