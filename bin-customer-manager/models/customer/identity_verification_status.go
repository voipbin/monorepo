package customer

// IdentityVerificationStatus represents the customer's identity verification state.
type IdentityVerificationStatus string

const (
	IdentityVerificationStatusNone     IdentityVerificationStatus = "none"
	IdentityVerificationStatusPending  IdentityVerificationStatus = "pending"
	IdentityVerificationStatusVerified IdentityVerificationStatus = "verified"
	IdentityVerificationStatusRejected IdentityVerificationStatus = "rejected"
)

// ValidIdentityVerificationStatuses contains all valid status values for input validation.
var ValidIdentityVerificationStatuses = []IdentityVerificationStatus{
	IdentityVerificationStatusNone,
	IdentityVerificationStatusPending,
	IdentityVerificationStatusVerified,
	IdentityVerificationStatusRejected,
}

// IsValid returns true if the status is one of the defined constants.
func (s IdentityVerificationStatus) IsValid() bool {
	for _, v := range ValidIdentityVerificationStatuses {
		if s == v {
			return true
		}
	}
	return false
}
