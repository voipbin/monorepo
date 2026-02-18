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

// Default credit rates per unit in micros (1 dollar = 1,000,000 micros).
const (
	DefaultCreditPerUnitCallPSTNOutgoing int64 = 6000    // $0.006/min
	DefaultCreditPerUnitCallPSTNIncoming int64 = 4500    // $0.0045/min
	DefaultCreditPerUnitCallVN           int64 = 4500    // $0.0045/min
	DefaultCreditPerUnitSMS              int64 = 8000    // $0.008/msg
	DefaultCreditPerUnitNumber           int64 = 5000000 // $5.00/number
)

// Default token rates per unit (plain integers).
const (
	DefaultTokenPerUnitCallVN int64 = 1
)

// GetCostInfo returns the token rate and credit rate for a given cost type.
func GetCostInfo(ct CostType) (tokenPerUnit int64, creditPerUnit int64) {
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
		return 0, DefaultCreditPerUnitSMS
	case CostTypeNumber, CostTypeNumberRenew:
		return 0, DefaultCreditPerUnitNumber
	default:
		return 0, 0
	}
}
