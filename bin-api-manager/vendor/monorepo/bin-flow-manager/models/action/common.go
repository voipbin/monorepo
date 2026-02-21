package action

// OptionAMDMachineHandleType defines
type OptionAMDMachineHandleType string

// list of OptionAMDMachineHandleType
const (
	OptionAMDMachineHandleTypeHangup   OptionAMDMachineHandleType = "hangup"
	OptionAMDMachineHandleTypeContinue OptionAMDMachineHandleType = "continue"
)

// OptionConditionCommonCondition define
type OptionConditionCommonCondition string

// list of OptionConditionCommonCondition.
const (
	OptionConditionCommonConditionEqual        OptionConditionCommonCondition = "=="
	OptionConditionCommonConditionNotEqual     OptionConditionCommonCondition = "!="
	OptionConditionCommonConditionGreater      OptionConditionCommonCondition = ">"
	OptionConditionCommonConditionGreaterEqual OptionConditionCommonCondition = ">="
	OptionConditionCommonConditionLess         OptionConditionCommonCondition = "<"
	OptionConditionCommonConditionLessEqual    OptionConditionCommonCondition = "<="
)

// OptionConditionCallStatusStatus define
type OptionConditionCallStatusStatus string

// list of OptionConditionCallStatusStatus
// copied from the call-manager
const (
	OptionConditionCallStatusStatusDialing     OptionConditionCallStatusStatus = "dialing"     // The call is created. We are dialing to the destination.
	OptionConditionCallStatusStatusRinging     OptionConditionCallStatusStatus = "ringing"     // The destination has confirmed that the call is ringng.
	OptionConditionCallStatusStatusProgressing OptionConditionCallStatusStatus = "progressing" // The call has answered. The both endpoints are talking to each other.
	OptionConditionCallStatusStatusTerminating OptionConditionCallStatusStatus = "terminating" // The call is terminating.
	OptionConditionCallStatusStatusCanceling   OptionConditionCallStatusStatus = "canceling"   // The call originator is canceling the call.
	OptionConditionCallStatusStatusHangup      OptionConditionCallStatusStatus = "hangup"      // The call has been completed.
)

// OptionConditionVariableValueType define
type OptionConditionVariableValueType string

// list of OptionConditionVariableType.
const (
	OptionConditionVariableTypeString OptionConditionVariableValueType = "string"
	OptionConditionVariableTypeNumber OptionConditionVariableValueType = "number"
	OptionConditionVariableTypeLength OptionConditionVariableValueType = "length"
)

// list of OptionBranch's const
const (
	OptionBranchVariableDefault = "voipbin.call.digits" // Default variable for branch option.
)

// OptionTalkDigitsHandle define
type OptionTalkDigitsHandle string

// list of OptionTalkDigitsHandle
const (
	OptionTalkDigitsHandleNone OptionTalkDigitsHandle = ""     // do nothing
	OptionTalkDigitsHandleNext OptionTalkDigitsHandle = "next" // if it receives digits, move to the next action
)
