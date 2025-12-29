package common

import (
	"errors"
	"fmt"
	"regexp"
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

// regex to validate domain format (RFC 1123 compliant mostly)
// Alphanumeric, hyphens, dots. No spaces.
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

func SetBaseDomainNames(extensionBase string, trunkBase string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "SetBaseDomainNames",
	})

	if !isValidDomain(extensionBase) {
		return errors.New("extension base domain name is invalid (check format)")
	}
	if !isValidDomain(trunkBase) {
		return errors.New("trunk base domain name is invalid (check format)")
	}

	initialized := false
	initOnce.Do(func() {
		baseDomainNameExtension = extensionBase
		baseDomainNameTrunk = trunkBase
		initialized = true

		log.Infof("Set base domain names. ext: %s, trunk: %s", baseDomainNameExtension, baseDomainNameTrunk)
	})

	if !initialized {
		return errors.New("base domain names have already been initialized and cannot be changed")
	}

	return nil
}

func getBaseDomainNameExtension() string {
	if baseDomainNameExtension == "" {
		logrus.WithFields(logrus.Fields{
			"func": "getBaseDomainNameExtension",
		}).Panic("baseDomainNameExtension is not initialized; call SetBaseDomainNames before generating realms or endpoints")
	}
	return baseDomainNameExtension
}

func getBaseDomainNameTrunk() string {
	if baseDomainNameTrunk == "" {
		logrus.WithFields(logrus.Fields{
			"func": "getBaseDomainNameTrunk",
		}).Panic("baseDomainNameTrunk is not initialized; call SetBaseDomainNames before generating realms or endpoints")
	}
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

// GenerateRealmTrunkDomain returns the realm of the given trunk's domain name
func GenerateRealmTrunkDomain(domainName string) string {
	res := fmt.Sprintf("%s.%s", domainName, getBaseDomainNameTrunk())
	return res
}
