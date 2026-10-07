package servicehandler

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
)

// The Flow Builder's own transport counters. The Assistant Builder's series
// keep their names and labels; these are separate series so a dashboard can
// tell the two apart (VOIP-1573 design doc 5.1).
var (
	metricFlowBuilderTimeout = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_manager_flow_builder_timeout_total",
		Help: "Total number of flow builder requests that ended with a deadline.",
	})

	metricFlowBuilderCircuitOpen = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_manager_flow_builder_circuit_open_total",
		Help: "Total number of flow builder requests refused by an open circuit.",
	})
)

// FlowBuilderChat runs one turn of the flow builder conversation.
//
// Who may use it is the same rule as the Assistant Builder (canUseBuilder: a
// logged-in Agent holding the customer admin or manager permission), which is
// also the permission to create a flow. Availability is the Assistant
// Builder's status (AIBuilderStatus): both depend on the same platform key.
//
// It logs the agent, the customer and a message count, never the conversation,
// the draft or the error text of a failed call.
func (h *serviceHandler) FlowBuilderChat(ctx context.Context, a *auth.AuthIdentity, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
	if !h.canUseBuilder(ctx, a) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowBuilderChat",
		"agent_id":    a.AgentID(),
		"customer_id": a.CustomerID,
	})

	// The same check ai-manager runs, so a request that can never succeed
	// costs no RPC. ValidateRequest builds its own error and never echoes input.
	if errValidate := flowbuilder.ValidateRequest(req); errValidate != nil {
		return nil, errValidate
	}
	log = log.WithField("message_count", len(req.Messages))

	res, err := h.reqHandler.AIV1FlowBuilderChat(ctx, a.CustomerID, req)
	if err != nil {
		return nil, mapBuilderRPCError(log, err, metricFlowBuilderTimeout, metricFlowBuilderCircuitOpen, "flow builder")
	}

	return res, nil
}
