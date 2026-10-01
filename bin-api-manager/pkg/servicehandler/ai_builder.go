package servicehandler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
)

var (
	// metricBuilderTimeout counts Builder RPCs that ended with a deadline. It is
	// measured against the 55 second RPC wait, so it also counts the case where
	// the caller's own request context ran out; the two cannot be told apart
	// here. It does not count a cancel.
	metricBuilderTimeout = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_manager_builder_timeout_total",
		Help: "Total number of assistant builder requests that ended with a deadline.",
	})

	// metricBuilderCircuitOpen counts Builder RPCs refused because the circuit to
	// ai-manager was open.
	metricBuilderCircuitOpen = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_manager_builder_circuit_open_total",
		Help: "Total number of assistant builder requests refused by an open circuit.",
	})
)

// Status cache windows. A failure is remembered for less time than a success so
// that a short ai-manager hiccup does not hide the Builder for long.
const (
	builderStatusOKTTL       = 30 * time.Second
	builderStatusFailTTL     = 5 * time.Second
	builderStatusRPCDeadline = 3 * time.Second
)

// builderStatusCache is the per-process cache of the Builder status. The zero
// value is ready to use.
type builderStatusCache struct {
	mu      sync.Mutex
	value   *builder.StatusResponse
	expires time.Time
	group   singleflight.Group
	now     func() time.Time // nil means time.Now; tests replace it
}

func (c *builderStatusCache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// canUseBuilder is the one rule for who may use the Builder.
//
// The caller must be a logged-in Agent AND hold the customer admin or manager
// permission. The first is checked on its own because HasPermission grants an
// accesskey and a delegate token the admin bit, and the platform pays for every
// turn: a script holding an accesskey must not be able to spend that budget.
func (h *serviceHandler) canUseBuilder(ctx context.Context, a *auth.AuthIdentity) bool {
	if a == nil || !a.IsAgent() {
		return false
	}
	return h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
}

// AIBuilderChat runs one turn of the assistant builder conversation.
//
// It logs the agent, the customer and a message count, never the conversation
// or the error text of a failed call (a provider error can echo the prompt).
func (h *serviceHandler) AIBuilderChat(ctx context.Context, a *auth.AuthIdentity, req *builder.ChatRequest) (*builder.ChatResponse, error) {
	if !h.canUseBuilder(ctx, a) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AIBuilderChat",
		"agent_id":    a.AgentID(),
		"customer_id": a.CustomerID,
	})

	// The same check ai-manager runs, so a request that can never succeed costs
	// no RPC. ValidateRequest builds its own error and never echoes input.
	if errValidate := builder.ValidateRequest(req); errValidate != nil {
		return nil, errValidate
	}
	log = log.WithField("message_count", len(req.Messages))

	res, err := h.reqHandler.AIV1BuilderChat(ctx, a.CustomerID, req)
	if err != nil {
		return nil, h.mapBuilderRPCError(log, err)
	}

	return res, nil
}

// mapBuilderRPCError turns a failed Builder RPC into what the client sees.
//
// The two transport failures are detected and counted here, in one place, so the
// HTTP handler needs no way to reach the counters and the translator in
// server/error_translate.go (whose generic mapping would give the client a less
// specific reason) is bypassed in one place only.
//
// A typed error from ai-manager is passed through as it is: its reason is what
// the client acts on. Nothing here puts the error text in a log line.
func (h *serviceHandler) mapBuilderRPCError(log *logrus.Entry, err error) error {
	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) {
		log.WithField("reason", ve.Reason).Info("The builder turn did not succeed.")
		return err
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		metricBuilderTimeout.Inc()
		log.Info("The builder turn timed out.")
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, builder.ReasonTimeout, "The assistant builder took too long. Please try again.").Wrap(err)

	case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
		metricBuilderCircuitOpen.Inc()
		log.Info("The builder circuit is open.")
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE", "An upstream service is temporarily unavailable.").Wrap(err)
	}

	// Anything else (a cancel, a bare sentinel) goes on to the generic translator.
	log.Info("The builder turn failed.")
	return err
}

// AIBuilderStatus reports whether the Builder is usable for this caller.
//
// A caller who may not use the Builder, and any failure to find out, get
// available=false, never an error: the client uses this to decide whether to
// show an entry point.
//
// The permission check comes BEFORE the cache, so a cached "available" is never
// shown to someone who may not use it.
func (h *serviceHandler) AIBuilderStatus(ctx context.Context, a *auth.AuthIdentity) (*builder.StatusResponse, error) {
	if !h.canUseBuilder(ctx, a) {
		return &builder.StatusResponse{Available: false}, nil
	}

	return h.builderStatusCached(ctx), nil
}

func (h *serviceHandler) builderStatusCached(ctx context.Context) *builder.StatusResponse {
	c := &h.builderStatus

	c.mu.Lock()
	if c.value != nil && c.clock().Before(c.expires) {
		v := *c.value
		c.mu.Unlock()
		return &v
	}
	c.mu.Unlock()

	// One RPC for all callers that find the cache expired. It runs on its own
	// context, not the first caller's: if the first caller gave up, the others
	// waiting on the same call would otherwise inherit its cancellation, and the
	// outage would be cached for the failure window.
	ch := c.group.DoChan("status", func() (any, error) {
		rpcCtx, cancel := context.WithTimeout(context.Background(), builderStatusRPCDeadline)
		defer cancel()

		res, err := h.reqHandler.AIV1BuilderStatus(rpcCtx)

		c.mu.Lock()
		defer c.mu.Unlock()
		if err != nil || res == nil {
			c.value = &builder.StatusResponse{Available: false}
			c.expires = c.clock().Add(builderStatusFailTTL)
			return *c.value, nil
		}
		c.value = res
		c.expires = c.clock().Add(builderStatusOKTTL)
		return *res, nil
	})

	select {
	case r := <-ch:
		v, _ := r.Val.(builder.StatusResponse)
		return &v
	case <-ctx.Done():
		// This caller gave up. It sees "not available" and nothing is cached on
		// its account; the shared call carries on for the others.
		return &builder.StatusResponse{Available: false}
	}
}
