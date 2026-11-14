package providerhandler

//go:generate mockgen -package providerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
)

// Provider is the operations a number provider implements.
type Provider interface {
	GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	NumberPurchase(num string) (*providernumber.ProviderNumber, error)
	NumberRelease(ctx context.Context, num *number.Number) error
	NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error
}

// Factory creates a Provider from the resolved credentials.
type Factory func(c provider.Credentials) (Provider, error)

// Descriptor describes a provider known to the code. Only the providers registered here
// have a meaning in number_providers.
type Descriptor struct {
	Factory Factory

	// AllowedConfig is the list of the non-secret config keys the admin may set.
	AllowedConfig []string

	// RequiredConfig is the list of config keys a row needs. A provider whose row lacks one is not
	// usable (so never a purchase candidate), while it can still serve release and search.
	RequiredConfig []string

	// EnvSecret is the environment variable that holds the secret of the provider.
	EnvSecret string

	// EnvConfig maps a config key to the environment variable that fills it when the row has no
	// value for the key.
	EnvConfig map[string]string

	// Purchasable is false when the provider cannot buy numbers (it is never a purchase candidate).
	Purchasable bool

	// DefaultEnabled and DefaultPriority apply when there is no row.
	DefaultEnabled  bool
	DefaultPriority int
}

// Candidate is a provider to try for a purchase.
type Candidate struct {
	Name     number.ProviderName
	Provider Provider
}

// ProviderHandler resolves the providers and manages their settings.
type ProviderHandler interface {
	// PurchaseCandidates returns the usable and purchasable providers, ordered by priority then name.
	PurchaseCandidates(ctx context.Context) ([]*Candidate, error)

	// ForRelease returns the provider of the name for release, tag and search operations.
	// It only needs a secret and ignores the enabled flag.
	ForRelease(ctx context.Context, name number.ProviderName) (Provider, error)

	// ForSearch returns the provider used for the available number search.
	ForSearch(ctx context.Context) (Provider, error)

	List(ctx context.Context) ([]*provider.Info, error)
	Get(ctx context.Context, name number.ProviderName) (*provider.Info, error)
	Update(ctx context.Context, name number.ProviderName, actor uuid.UUID, enabled *bool, priority *int, config map[string]string) (*provider.Info, error)
	UpdateSecret(ctx context.Context, name number.ProviderName, actor uuid.UUID, secret string) (*provider.Info, error)
	DeleteSecret(ctx context.Context, name number.ProviderName, actor uuid.UUID) (*provider.Info, error)
}

type providerHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler
	crypto      *secretcrypto.Crypto

	registry map[number.ProviderName]Descriptor
}

// limits of the admin input
const (
	maxPriority       = 1000000
	maxConfigKeys     = 8
	maxConfigKeyLen   = 32
	maxConfigValueLen = 256
)

var (
	metricsNamespace = "number_manager"

	promProviderUnusable = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "provider_unusable_total",
			Help:      "Total number of the resolutions of an enabled provider that could not be used.",
		},
		[]string{"provider", "reason"},
	)

	promProviderPurchase = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "provider_purchase_total",
			Help:      "Total number of the purchase attempts by provider and result.",
		},
		[]string{"provider", "result"},
	)
)

func init() {
	prometheus.MustRegister(
		promProviderUnusable,
		promProviderPurchase,
	)
}

// ObservePurchase counts a purchase attempt of the provider. result is success, rejected or unconfirmed.
func ObservePurchase(name number.ProviderName, result string) {
	promProviderPurchase.WithLabelValues(string(name), result).Inc()
}

// NewProviderHandler returns a ProviderHandler. The credentials of a provider come from its row in
// number_providers only.
func NewProviderHandler(
	r requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	crypto *secretcrypto.Crypto,
) ProviderHandler {
	return &providerHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		crypto:      crypto,
		registry:    newRegistry(r, db),
	}
}

// newRegistry returns the providers known to the code.
func newRegistry(r requesthandler.RequestHandler, db dbhandler.DBHandler) map[number.ProviderName]Descriptor {
	return map[number.ProviderName]Descriptor{
		number.ProviderNameTelnyx: {
			Factory: func(c provider.Credentials) (Provider, error) {
				return numberhandlertelnyx.NewNumberHandler(r, db, c.Config[configConnectionID], c.Config[configProfileID], c.Secret), nil
			},
			EnvSecret:       envTelnyxAPIKey,
			EnvConfig:       map[string]string{configConnectionID: envTelnyxConnectionID, configProfileID: envTelnyxProfileID},
			AllowedConfig:   []string{configConnectionID, configProfileID},
			RequiredConfig:  []string{configConnectionID},
			Purchasable:     true,
			DefaultEnabled:  true,
			DefaultPriority: 10,
		},
		number.ProviderNameTwilio: {
			Factory: func(c provider.Credentials) (Provider, error) {
				// the sid is a part of every API path of the twilio, so it is checked here for release and
				// search too. The trunk_sid is only used by a purchase, which checks it before any request.
				// (the handler removes the surrounding whitespace of the stored values)
				if err := numberhandlertwilio.ValidateSID(strings.TrimSpace(c.Config[configSID])); err != nil {
					return nil, err
				}
				return numberhandlertwilio.NewNumberHandler(r, db, c.Config[configSID], c.Config[configTrunkSID], c.Secret), nil
			},
			EnvSecret:       envTwilioAuthToken,
			EnvConfig:       map[string]string{configSID: envTwilioAccountSID},
			AllowedConfig:   []string{configSID, configTrunkSID},
			RequiredConfig:  []string{configSID, configTrunkSID},
			Purchasable:     true,
			DefaultEnabled:  false,
			DefaultPriority: 20,
		},
	}
}

// config keys
const (
	configConnectionID = "connection_id"
	configProfileID    = "profile_id"
	configSID          = "sid"
	configTrunkSID     = "trunk_sid"
)

// environment variables of the platform credentials
const (
	envTelnyxAPIKey       = "TELNYX_API_KEY"
	envTelnyxConnectionID = "TELNYX_CONNECTION_ID"
	envTelnyxProfileID    = "TELNYX_PROFILE_ID"
	envTwilioAccountSID   = "TWILIO_ACCOUNT_SID"
	envTwilioAuthToken    = "TWILIO_AUTH_TOKEN"
)
