package common

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// list of const variables
var (
	baseDomainNameExtension = "" // base domain name for extension realm
	baseDomainNameTrunk     = "" // base domain name for trunk realm
)

func SetBaseDomainNames(extensionDomain string, trunkDomain string) {
	log := logrus.WithFields(logrus.Fields{
		"func": "SetBaseDomainNames",
	})

	if extensionDomain != "" {
		baseDomainNameExtension = extensionDomain
	}

	if trunkDomain != "" {
		baseDomainNameTrunk = trunkDomain
	}

	log.Infof("Set base domain names. base_domain_name_extension: %s, base_domain_name_trunk: %s", baseDomainNameExtension, baseDomainNameTrunk)
}

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
