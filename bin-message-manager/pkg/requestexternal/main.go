package requestexternal

//go:generate mockgen -package requestexternal -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-message-manager/models/messagebird"
	"monorepo/bin-message-manager/models/telnyx"
)

var (
	metricsNamespace = "message_manager"

	promRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "request_external_process_time",
			Help:      "Process time of send/receiv requests",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"target", "resource", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestExternal intreface for ARI request handler
type RequestExternal interface {

	// messagebird
	MessagebirdSendMessage(ctx context.Context, sender string, destinations []string, text string) (*messagebird.Message, error)
	MessagebirdSendMMS(ctx context.Context, sender string, destination string, text string, mediaURLs []string) (*messagebird.Message, error)

	// telnyx
	TelnyxSendMessage(ctx context.Context, source string, destination string, text string) (*telnyx.MessageResponse, error)
	TelnyxSendMMS(ctx context.Context, source string, destination string, text string, mediaURLs []string) (*telnyx.MessageResponse, error)
}

// TokenFunc hands a provider token to the request at the time of use. Nothing is cached by the
// caller, so a token that was changed is used by the next request. An empty token with no error
// means "not configured". An error means that the token could not be read: the request is not
// sent, and the error is never replaced by another token.
type TokenFunc func(ctx context.Context) (string, error)

// ErrTokenUnavailable is wrapped by the error of a request that was not sent because the token
// could not be read. The request never reached the provider, so no message exists there. The
// cause (a database error, a stored value that can not be decrypted, a read that ran out of time)
// is wrapped as well and holds no token text.
var ErrTokenUnavailable = errors.New("the provider token could not be read, so the request was not sent")

// IsTokenUnavailable reports whether the request was not sent because its token could not be read.
func IsTokenUnavailable(err error) bool {
	return errors.Is(err, ErrTokenUnavailable)
}

type requestExternal struct {
	authtokenMessagebird TokenFunc // supplies the authentication token for messagebird
	authtokenTelnyx      TokenFunc // supplies the authentication token for telnyx
}

// NewRequestExternal create RequestExternal. The tokens are asked for on every request.
func NewRequestExternal(authtokenMessagebird TokenFunc, authtokenTelnyx TokenFunc) RequestExternal {
	h := &requestExternal{
		authtokenMessagebird: authtokenMessagebird,
		authtokenTelnyx:      authtokenTelnyx,
	}

	return h
}
