package providernumber

import "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

// ProviderNumber defines
type ProviderNumber struct {
	ID               string
	Status           number.Status
	T38Enabled       bool
	EmergencyEnabled bool
}
