package billing

// CostType classifies why a billing cost was applied.
type CostType string

const (
	CostTypeNone             CostType = ""
	CostTypeCallPSTNOutgoing CostType = "call_pstn_outgoing"
	CostTypeCallPSTNIncoming CostType = "call_pstn_incoming"
	CostTypeCallVN           CostType = "call_vn"
	CostTypeCallExtension    CostType = "call_extension"
	CostTypeCallDirectExt    CostType = "call_direct_ext"
	CostTypeSMS              CostType = "sms"
	CostTypeNumber           CostType = "number"
	CostTypeNumberRenew      CostType = "number_renew"
)

// Default credit rates per unit (per-minute for calls, per-message for SMS, per-number for numbers).
const (
	DefaultCreditPerUnitCallPSTNOutgoing float32 = 0.006
	DefaultCreditPerUnitCallPSTNIncoming float32 = 0.0045
	DefaultCreditPerUnitCallVN           float32 = 0.0045
	DefaultCreditPerUnitSMS              float32 = 0.008
	DefaultCreditPerUnitNumber           float32 = 5.0
)

// Default token rates per unit.
const (
	DefaultTokenPerUnitCallVN int = 1
	DefaultTokenPerUnitSMS    int = 10
)

// GetCostInfo returns the token rate and credit rate for a given cost type.
func GetCostInfo(ct CostType) (tokenPerUnit int, creditPerUnit float32) {
	switch ct {
	case CostTypeCallPSTNOutgoing:
		return 0, DefaultCreditPerUnitCallPSTNOutgoing
	case CostTypeCallPSTNIncoming:
		return 0, DefaultCreditPerUnitCallPSTNIncoming
	case CostTypeCallVN:
		return DefaultTokenPerUnitCallVN, DefaultCreditPerUnitCallVN
	case CostTypeCallExtension, CostTypeCallDirectExt:
		return 0, 0
	case CostTypeSMS:
		return DefaultTokenPerUnitSMS, DefaultCreditPerUnitSMS
	case CostTypeNumber, CostTypeNumberRenew:
		return 0, DefaultCreditPerUnitNumber
	default:
		return 0, 0
	}
}
