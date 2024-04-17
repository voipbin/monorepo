package providernumber

import "monorepo/bin-number-manager/models/number"

// ProviderNumber defines
type ProviderNumber struct {
	ID               string
	Status           number.Status
	T38Enabled       bool
	EmergencyEnabled bool
}
