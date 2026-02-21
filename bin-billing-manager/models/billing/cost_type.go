package billing

// CostMode declares how a cost type is charged.
type CostMode int

const (
	CostModeDisabled   CostMode = iota // Service not available — requests rejected
	CostModeFree                        // Allowed, no charge
	CostModeCreditOnly                  // Credit only, tokens not accepted
	CostModeTokenFirst                  // Token first, overflow to credits
)

// CostInfo holds the billing mode and rates for a cost type.
type CostInfo struct {
	Mode          CostMode
	TokenPerUnit  int64
	CreditPerUnit int64
}

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
	CostTypeEmail            CostType = "email"
	CostTypeNumber           CostType = "number"
	CostTypeNumberRenew      CostType = "number_renew"
)

// Default credit rates per unit in micros (1 dollar = 1,000,000 micros).
const (
	DefaultCreditPerUnitCallPSTNOutgoing int64 = 10000   // $0.01/min
	DefaultCreditPerUnitCallPSTNIncoming int64 = 10000   // $0.01/min
	DefaultCreditPerUnitCallVN           int64 = 1000    // $0.001/min
	DefaultCreditPerUnitSMS              int64 = 10000   // $0.01/msg
	DefaultCreditPerUnitEmail            int64 = 10000   // $0.01/msg
	DefaultCreditPerUnitNumber           int64 = 5000000 // $5.00/number
)

// Default token rates per unit (plain integers).
const (
	DefaultTokenPerUnitCallVN int64 = 1
)

// GetCostInfo returns the billing mode and rates for a given cost type.
func GetCostInfo(ct CostType) CostInfo {
	switch ct {
	case CostTypeCallPSTNOutgoing:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitCallPSTNOutgoing}
	case CostTypeCallPSTNIncoming:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitCallPSTNIncoming}
	case CostTypeCallVN:
		return CostInfo{CostModeTokenFirst, DefaultTokenPerUnitCallVN, DefaultCreditPerUnitCallVN}
	case CostTypeCallExtension, CostTypeCallDirectExt:
		return CostInfo{CostModeFree, 0, 0}
	case CostTypeSMS:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitSMS}
	case CostTypeEmail:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitEmail}
	case CostTypeNumber, CostTypeNumberRenew:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitNumber}
	default:
		return CostInfo{CostModeDisabled, 0, 0}
	}
}
