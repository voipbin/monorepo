package common

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/projectconfig"
)

// Domain variables initialized once from project config
var (
	DomainConference      = projectconfig.Get().DomainConference
	DomainPSTN            = projectconfig.Get().DomainPSTN
	DomainTrunkSuffix     = projectconfig.Get().DomainTrunkSuffix
	DomainRegistrarSuffix = projectconfig.Get().DomainRegistrarSuffix
)

// ParseSIPURI parses the sip uri.
func ParseSIPURI(uri string) (uuid.UUID, string, error) {
	tmp := strings.Split(uri, "@")
	if len(tmp) < 2 {
		return uuid.Nil, "", fmt.Errorf("could not parse the endpoint")
	}

	extension := tmp[0]
	domain := tmp[1]

	tmpCustomerID := strings.TrimSuffix(domain, DomainRegistrarSuffix)
	customerID := uuid.FromStringOrNil(tmpCustomerID)

	return customerID, extension, nil
}
