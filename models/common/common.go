package common

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

// list of domain defines
const (
	DomainConference      = "conference.voipbin.net"
	DomainPSTN            = "pstn.voipbin.net"
	DomainTrunkSuffix     = ".trunk.voipbin.net"
	DomainRegistrarSuffix = ".registrar.voipbin.net"
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
