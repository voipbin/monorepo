package dbhandler

import (
	"testing"
)

func Test_CalculateTokenCreditDeduction(t *testing.T) {

	tests := []struct {
		name string

		balanceToken     int64
		billableUnits    int
		rateTokenPerUnit int64
		rateCreditPerUnit int64

		expectTokenDeducted  int64
		expectCreditDeducted int64
	}{
		// =====================================================================
		// Zero / no-op cases
		// =====================================================================
		{
			name:                 "zero billable units",
			balanceToken:         100,
			billableUnits:        0,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 0,
		},
		{
			name:                 "zero rates",
			balanceToken:         100,
			billableUnits:        5,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    0,
			expectTokenDeducted:  0,
			expectCreditDeducted: 0,
		},
		{
			name:                 "negative billable units",
			balanceToken:         100,
			billableUnits:        -3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 0,
		},

		// =====================================================================
		// Credit-only cost types (rateTokenPerUnit = 0)
		// =====================================================================
		{
			name:                 "credit only - PSTN outgoing",
			balanceToken:         1000,
			billableUnits:        2,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 12000,
		},
		{
			name:                 "credit only - number purchase",
			balanceToken:         500,
			billableUnits:        1,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    5000000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 5000000,
		},
		{
			name:                 "credit only - zero token balance irrelevant",
			balanceToken:         0,
			billableUnits:        3,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    8000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 24000,
		},

		// =====================================================================
		// Token-only cost types (rateCreditPerUnit = 0, rateTokenPerUnit > 0)
		// =====================================================================
		{
			name:                 "token only - exact match",
			balanceToken:         10,
			billableUnits:        1,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    0,
			expectTokenDeducted:  10,
			expectCreditDeducted: 0,
		},
		{
			name:                 "token only - surplus tokens",
			balanceToken:         100,
			billableUnits:        2,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    0,
			expectTokenDeducted:  20,
			expectCreditDeducted: 0,
		},
		{
			name:                 "token only - insufficient (partial, zero credit rate)",
			balanceToken:         15,
			billableUnits:        2,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    0,
			expectTokenDeducted:  10,
			expectCreditDeducted: 0, // remaining 1 unit * 0 credit = 0
		},
		{
			name:                 "token only - zero balance (zero credit rate)",
			balanceToken:         0,
			billableUnits:        3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    0,
			expectTokenDeducted:  0,
			expectCreditDeducted: 0, // 3 units * 0 credit = 0
		},

		// =====================================================================
		// Token-first with credit overflow (both rates > 0)
		// =====================================================================
		{
			name:                 "both rates - tokens cover all units",
			balanceToken:         100,
			billableUnits:        3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  30,
			expectCreditDeducted: 0,
		},
		{
			name:                 "both rates - exact token match",
			balanceToken:         30,
			billableUnits:        3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  30,
			expectCreditDeducted: 0,
		},
		{
			name:                 "both rates - partial tokens overflow to credit",
			balanceToken:         25,
			billableUnits:        3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  20, // 2 full units @ 10 tokens each
			expectCreditDeducted: 4500, // 1 remaining unit @ 4500 credit
		},
		{
			name:                 "both rates - zero tokens all to credit",
			balanceToken:         0,
			billableUnits:        3,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  0,
			expectCreditDeducted: 13500, // 3 units @ 4500 credit each
		},
		{
			name:                 "both rates - 1 token less than 1 unit",
			balanceToken:         9,
			billableUnits:        2,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0, // 9 < 10 → 0 full units
			expectCreditDeducted: 12000, // 2 units @ 6000
		},
		{
			name:                 "both rates - single unit exact token",
			balanceToken:         1,
			billableUnits:        1,
			rateTokenPerUnit:     1,
			rateCreditPerUnit:    8000,
			expectTokenDeducted:  1,
			expectCreditDeducted: 0,
		},

		// =====================================================================
		// VN call scenario (token-eligible with credit fallback)
		// =====================================================================
		{
			name:                 "VN call - 2 min call with 100 tokens at rate 1",
			balanceToken:         100,
			billableUnits:        2,
			rateTokenPerUnit:     1,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  2,
			expectCreditDeducted: 0,
		},
		{
			name:                 "VN call - 3 min call with 1 token left",
			balanceToken:         1,
			billableUnits:        3,
			rateTokenPerUnit:     1,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  1,
			expectCreditDeducted: 9000, // 2 remaining * 4500
		},

		// =====================================================================
		// SMS scenario (credit only)
		// =====================================================================
		{
			name:                 "SMS - credit only",
			balanceToken:         50,
			billableUnits:        1,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    8000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 8000,
		},

		// =====================================================================
		// Defensive: negative balanceToken (possible from concurrent calls)
		// =====================================================================
		{
			name:                 "negative token balance - all overflows to credit",
			balanceToken:         -5,
			billableUnits:        2,
			rateTokenPerUnit:     10,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0, // Go int division: -5/10 = 0 full units
			expectCreditDeducted: 12000,
		},
		{
			name:                 "negative token balance with rate 1",
			balanceToken:         -3,
			billableUnits:        5,
			rateTokenPerUnit:     1,
			rateCreditPerUnit:    4500,
			expectTokenDeducted:  0, // negative balance → skip tokens, all to credit
			expectCreditDeducted: 22500,
		},

		// =====================================================================
		// Large values (near int64 boundaries)
		// =====================================================================
		{
			name:                 "large token balance",
			balanceToken:         1000000000,
			billableUnits:        1000,
			rateTokenPerUnit:     100,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  100000,
			expectCreditDeducted: 0,
		},
		{
			name:                 "large credit values (micros)",
			balanceToken:         0,
			billableUnits:        60,
			rateTokenPerUnit:     0,
			rateCreditPerUnit:    6000,
			expectTokenDeducted:  0,
			expectCreditDeducted: 360000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTokenCreditDeduction(tt.balanceToken, tt.billableUnits, tt.rateTokenPerUnit, tt.rateCreditPerUnit)

			if result.TokenDeducted != tt.expectTokenDeducted {
				t.Errorf("TokenDeducted = %d, expected %d", result.TokenDeducted, tt.expectTokenDeducted)
			}
			if result.CreditDeducted != tt.expectCreditDeducted {
				t.Errorf("CreditDeducted = %d, expected %d", result.CreditDeducted, tt.expectCreditDeducted)
			}
		})
	}
}
