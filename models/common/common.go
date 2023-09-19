package common

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// baseDomainName base domain name
const baseDomainName = "registrar.voipbin.net"

// GenerateEndpoint returns the endpoint of the given customer with extension
func GenerateEndpoint(customerID uuid.UUID, extension string) string {
	realm := GenerateRealm(customerID)
	res := fmt.Sprintf("%s@%s", extension, realm)
	return res
}

// GenerateRealm returns the realm of the given customer
func GenerateRealm(customerID uuid.UUID) string {
	res := fmt.Sprintf("%s.%s", customerID.String(), baseDomainName)
	return res
}
