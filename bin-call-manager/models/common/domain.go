package common

import (
	"fmt"
	"strings"

	"monorepo/bin-call-manager/pkg/projectconfig"
)

// Domain variables initialized once from project config
var (
	DomainConference      = projectconfig.Get().DomainConference
	DomainPSTN            = projectconfig.Get().DomainPSTN
	DomainSIP             = projectconfig.Get().DomainSIP
	DomainTrunkSuffix     = projectconfig.Get().DomainTrunkSuffix
	DomainRegistrarSuffix = projectconfig.Get().DomainRegistrarSuffix
)

// ParseSIPURI splits the given sip uri(<extension>@<domain>) into
// extension and domain parts. It is a pure textual split: the caller is
// responsible for resolving the domain(realm) to a customer.
func ParseSIPURI(uri string) (string, string, error) {
	tmp := strings.Split(uri, "@")
	if len(tmp) < 2 {
		return "", "", fmt.Errorf("could not parse the endpoint")
	}

	extension := tmp[0]
	domain := tmp[1]

	return extension, domain, nil
}
