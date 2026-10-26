package numberhandlertwilio

//go:generate mockgen -package numberhandlertwilio -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/requestexternal"
)

// NumberHandlerTwilio is interface for service handle. It implements the provider operations
// (search, purchase, release and tag update) for the twilio.
type NumberHandlerTwilio interface {
	GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	NumberPurchase(num string) (*providernumber.ProviderNumber, error)
	NumberRelease(ctx context.Context, num *number.Number) error
	NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error
}

// formats of the twilio identifiers and of the number to purchase
var (
	regexpAccountSID = regexp.MustCompile(`^AC[0-9a-fA-F]{32}$`)
	regexpTrunkSID   = regexp.MustCompile(`^TK[0-9a-fA-F]{32}$`)
	regexpNumberSID  = regexp.MustCompile(`^PN[0-9a-fA-F]{32}$`)
	regexpE164       = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	regexpCountry    = regexp.MustCompile(`^[A-Za-z]{2}$`)
)

// ValidateSID checks the format of the account sid. The sid is a part of the API path, so a
// malformed value is refused before any request. The error does not hold the value.
func ValidateSID(sid string) error {
	if !regexpAccountSID.MatchString(sid) {
		return fmt.Errorf("the twilio account sid is not set or malformed")
	}
	return nil
}

// numberHandlerTwilio structure for service handle
type numberHandlerTwilio struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	sid      string
	trunkSID string
	token    string

	requestExternal requestexternal.RequestExternal
}

var (
	metricsNamespace = "number_manager"

	promCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "twilio_number_create_total",
			Help:      "Total number of created number type by twilio.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, sid string, trunkSID string, token string) NumberHandlerTwilio {

	reqExternal := requestexternal.NewRequestExternal()

	h := &numberHandlerTwilio{
		reqHandler: r,
		db:         db,

		sid:      sid,
		trunkSID: trunkSID,
		token:    token,

		requestExternal: reqExternal,
	}

	return h
}
