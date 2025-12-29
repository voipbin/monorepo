package common

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// list of base domain name variables
var (
	baseDomainNameExtension = "" // base domain name for extension realm
	baseDomainNameTrunk     = "" // base domain name for trunk realm
	initOnce                sync.Once
)

// SetBaseDomainNames sets the base domain names for extension and trunk realms
func SetBaseDomainNames(extensionBase string, trunkBase string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "SetBaseDomainNames",
	})

	if extensionBase == "" {
		return errors.New("base_domain_name_extension cannot be empty")
	}

	if trunkBase == "" {
		return errors.New("base_domain_name_trunk cannot be empty")
	}

	initialized := false
	initOnce.Do(func() {
		baseDomainNameExtension = extensionBase
		baseDomainNameTrunk = trunkBase
		initialized = true

		log.Infof("Set base domain names. base_domain_name_extension: %s, base_domain_name_trunk: %s", baseDomainNameExtension, baseDomainNameTrunk)
	})

	if !initialized {
		return errors.New("base domain names have already been initialized and cannot be changed")
	}

	return nil
}

func getBaseDomainNameExtension() string {
	return baseDomainNameExtension
}

func getBaseDomainNameTrunk() string {
	return baseDomainNameTrunk
}

// ResetBaseDomainNamesForTest resets the global domain variables and the sync.Once state.
// CAUTION: This function is intended for TESTING PURPOSES ONLY.
// Do not call this in production code.
func ResetBaseDomainNamesForTest() {
	baseDomainNameExtension = ""
	baseDomainNameTrunk = ""

	initOnce = sync.Once{}
}

// GenerateEndpointExtension returns the endpoint of the given customer with extension
func GenerateEndpointExtension(customerID uuid.UUID, extension string) string {
	realm := GenerateRealmExtension(customerID)
	res := fmt.Sprintf("%s@%s", extension, realm)
	return res
}

// GenerateRealmExtension returns the realm of the given customer
func GenerateRealmExtension(customerID uuid.UUID) string {
	res := fmt.Sprintf("%s.%s", customerID.String(), getBaseDomainNameExtension())
	return res
}

// GenerateRealmTrunkDomain returns the realm of the given turnk's domain name
func GenerateRealmTrunkDomain(domainName string) string {
	res := fmt.Sprintf("%s.%s", domainName, getBaseDomainNameTrunk())
	return res
}
