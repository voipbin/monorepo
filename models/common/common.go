package common

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// list of const variables
const (
	baseDomainNameExtension = "registrar.voipbin.net" // base domain name for extension realm
	baseDomainNameTrunk     = "trunk.voipbin.net"     // base domain name for trunk realm
)

// GenerateEndpointExtension returns the endpoint of the given customer with extension
func GenerateEndpointExtension(customerID uuid.UUID, extension string) string {
	realm := GenerateRealmExtension(customerID)
	res := fmt.Sprintf("%s@%s", extension, realm)
	return res
}

// GenerateRealmExtension returns the realm of the given customer
func GenerateRealmExtension(customerID uuid.UUID) string {
	res := fmt.Sprintf("%s.%s", customerID.String(), baseDomainNameExtension)
	return res
}

// GenerateRealmTrunkDomain returns the realm of the given turnk's domain name
func GenerateRealmTrunkDomain(domainName string) string {
	res := fmt.Sprintf("%s.%s", domainName, baseDomainNameTrunk)
	return res
}
