package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"monorepo/bin-common-handler/models/sock"
	hmhook "monorepo/bin-hook-manager/models/hook"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
)

// paddleEvent is the minimal Paddle v2 event envelope.
type paddleEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// paddleCustomData contains the VoIPBin customer ID embedded in Paddle's custom_data.
type paddleCustomData struct {
	CustomerID string `json:"customer_id"`
	PlanType   string `json:"plan_type"`
}

// paddleAdjustment represents a single refund/chargeback adjustment on a transaction.
type paddleAdjustment struct {
	Totals struct {
		Total string `json:"total"`
	} `json:"totals"`
}

// paddleTransactionData is the minimal transaction payload.
type paddleTransactionData struct {
	ID             string            `json:"id"`
	SubscriptionID *string           `json:"subscription_id"`
	CustomData     *paddleCustomData `json:"custom_data"`
	Details        struct {
		Totals struct {
			Total string `json:"total"`
		} `json:"totals"`
	} `json:"details"`
	// Adjustments contains refund/chargeback entries. Present in transaction.refunded events.
	// Each entry's totals.total is the refund amount for that adjustment.
	Adjustments []paddleAdjustment `json:"adjustments"`
}

// paddleScheduledChange represents a scheduled change on a Paddle subscription.
type paddleScheduledChange struct {
	Action      string `json:"action"`
	EffectiveAt string `json:"effective_at"`
}

// paddleSubscriptionData is the minimal subscription payload.
type paddleSubscriptionData struct {
	ID         string            `json:"id"`
	CustomerID string            `json:"customer_id"` // Paddle customer ID
	CustomData *paddleCustomData `json:"custom_data"`
	Items      []struct {
		Price struct {
			ID        string `json:"id"`
			ProductID string `json:"product_id"`
		} `json:"price"`
	} `json:"items"`
	ScheduledChange *paddleScheduledChange `json:"scheduled_change"`
}

// parsePaddleCentsToMicros converts a Paddle v2 amount string to internal micros.
// Paddle v2 sends all monetary amounts as strings representing integers in the
// lowest currency denomination (cents for USD). For example, "1000" means $10.00.
// Internal micros use 1 USD = 1,000,000 micros, so: micros = cents × 10,000.
func parsePaddleCentsToMicros(amountStr string) (int64, error) {
	cents, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse Paddle amount %q: %w", amountStr, err)
	}
	return cents * 10_000, nil
}

// processV1HooksPaddlePost handles POST /v1/hooks/paddle
func (h *listenHandler) processV1HooksPaddlePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksPaddlePost",
		"request": m,
	})

	// Unmarshal Hook envelope from RPC data
	var hook hmhook.Hook
	if err := json.Unmarshal(m.Data, &hook); err != nil {
		log.Errorf("Could not unmarshal hook data: %v", err)
		return simpleResponse(400), nil
	}

	// Parse the Paddle event from the raw webhook body
	var event paddleEvent
	if err := json.Unmarshal(hook.ReceivedData, &event); err != nil {
		log.Errorf("Could not parse Paddle event: %v", err)
		return simpleResponse(400), nil
	}
	if event.EventID == "" {
		log.Errorf("Missing event_id in Paddle event")
		return simpleResponse(400), nil
	}

	log.Infof("Received Paddle event. event_type: %s, event_id: %s", event.EventType, event.EventID)

	switch event.EventType {
	case "transaction.completed":
		return h.handlePaddleTransactionCompleted(ctx, &event)

	case "subscription.created":
		return h.handlePaddleSubscriptionCreated(ctx, &event)

	case "subscription.updated":
		return h.handlePaddleSubscriptionUpdated(ctx, &event)

	case "subscription.canceled":
		return h.handlePaddleSubscriptionCanceled(ctx, &event)

	case "transaction.refunded":
		return h.handlePaddleTransactionRefunded(ctx, &event)

	case "transaction.payment_failed":
		log.Errorf("Paddle payment failed. event_id: %s", event.EventID)
		return simpleResponse(200), nil

	default:
		log.Debugf("Unhandled Paddle event type, returning 200. event_type: %s, event_id: %s", event.EventType, event.EventID)
		return simpleResponse(200), nil
	}
}

// handlePaddleTransactionCompleted handles transaction.completed events.
// If subscription_id is present → subscription renewal; otherwise → one-time credit purchase.
func (h *listenHandler) handlePaddleTransactionCompleted(ctx context.Context, event *paddleEvent) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "handlePaddleTransactionCompleted",
		"event_id": event.EventID,
	})

	var txn paddleTransactionData
	if err := json.Unmarshal(event.Data, &txn); err != nil {
		log.Errorf("Could not parse transaction data: %v", err)
		return simpleResponse(400), nil
	}

	// Subscription renewal: has subscription_id
	if txn.SubscriptionID != nil && *txn.SubscriptionID != "" {
		log.Infof("Processing subscription renewal. transaction_id: %s, subscription_id: %s", txn.ID, *txn.SubscriptionID)
		if err := h.accountHandler.PaddleSubscriptionRenew(ctx, *txn.SubscriptionID, event.EventID); err != nil {
			log.Errorf("Could not process subscription renewal: %v", err)
			return simpleResponse(500), nil
		}
		log.Infof("Subscription renewal completed. transaction_id: %s, subscription_id: %s", txn.ID, *txn.SubscriptionID)
		return simpleResponse(200), nil
	}

	// One-time credit purchase: needs custom_data with customer_id
	if txn.CustomData == nil || txn.CustomData.CustomerID == "" {
		log.Infof("Missing customer_id in custom_data, skipping. transaction_id: %s", txn.ID)
		return simpleResponse(200), nil
	}

	customerID := uuid.FromStringOrNil(txn.CustomData.CustomerID)
	if customerID == uuid.Nil {
		log.Errorf("Invalid customer_id in custom_data: %s", txn.CustomData.CustomerID)
		return simpleResponse(400), nil
	}

	amountMicros, err := parsePaddleCentsToMicros(txn.Details.Totals.Total)
	if err != nil {
		log.Errorf("Could not parse transaction amount: %v", err)
		return simpleResponse(400), nil
	}

	log.Infof("Processing credit top-up. transaction_id: %s, customer_id: %s, amount_cents: %s, amount_micros: %d", txn.ID, txn.CustomData.CustomerID, txn.Details.Totals.Total, amountMicros)
	if err := h.accountHandler.PaddleCreditTopUp(ctx, customerID, amountMicros, event.EventID); err != nil {
		log.Errorf("Could not process credit top-up: %v", err)
		return simpleResponse(500), nil
	}
	log.Infof("Credit top-up completed. transaction_id: %s, customer_id: %s, amount_micros: %d", txn.ID, txn.CustomData.CustomerID, amountMicros)
	return simpleResponse(200), nil
}

// handlePaddleSubscriptionCreated handles subscription.created events.
func (h *listenHandler) handlePaddleSubscriptionCreated(ctx context.Context, event *paddleEvent) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "handlePaddleSubscriptionCreated",
		"event_id": event.EventID,
	})

	var sub paddleSubscriptionData
	if err := json.Unmarshal(event.Data, &sub); err != nil {
		log.Errorf("Could not parse subscription data: %v", err)
		return simpleResponse(400), nil
	}

	if sub.CustomData == nil || sub.CustomData.CustomerID == "" {
		log.Infof("Missing customer_id in custom_data, skipping. subscription_id: %s", sub.ID)
		return simpleResponse(200), nil
	}

	customerID := uuid.FromStringOrNil(sub.CustomData.CustomerID)
	if customerID == uuid.Nil {
		log.Errorf("Invalid customer_id in custom_data: %s", sub.CustomData.CustomerID)
		return simpleResponse(400), nil
	}

	planType := account.PlanType(sub.CustomData.PlanType)
	if _, ok := account.PlanTokenMap[planType]; !ok {
		log.Errorf("Unknown plan_type in custom_data: %s", sub.CustomData.PlanType)
		return simpleResponse(400), nil
	}

	log.Infof("Processing subscription create. subscription_id: %s, customer_id: %s, plan_type: %s, paddle_customer_id: %s", sub.ID, sub.CustomData.CustomerID, planType, sub.CustomerID)
	if err := h.accountHandler.PaddleSubscriptionCreate(ctx, customerID, planType, sub.ID, sub.CustomerID, event.EventID); err != nil {
		log.Errorf("Could not process subscription create: %v", err)
		return simpleResponse(500), nil
	}
	log.Infof("Subscription create completed. subscription_id: %s, customer_id: %s, plan_type: %s", sub.ID, sub.CustomData.CustomerID, planType)
	return simpleResponse(200), nil
}

// handlePaddleSubscriptionUpdated handles subscription.updated events.
// Handles two cases:
//  1. Scheduled cancellation: scheduled_change.action == "cancel" → sets plan_status to "canceling"
//  2. Plan change: resolves plan type from price ID (priority 1) or custom_data (priority 2)
func (h *listenHandler) handlePaddleSubscriptionUpdated(ctx context.Context, event *paddleEvent) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "handlePaddleSubscriptionUpdated",
		"event_id": event.EventID,
	})

	var sub paddleSubscriptionData
	if err := json.Unmarshal(event.Data, &sub); err != nil {
		log.Errorf("Could not parse subscription data: %v", err)
		return simpleResponse(400), nil
	}

	// Case 1: Scheduled cancellation
	if sub.ScheduledChange != nil && sub.ScheduledChange.Action == "cancel" {
		log.Infof("Processing scheduled cancellation. subscription_id: %s, effective_at: %s", sub.ID, sub.ScheduledChange.EffectiveAt)
		if err := h.accountHandler.PaddleSubscriptionScheduleCancel(ctx, sub.ID, event.EventID); err != nil {
			log.Errorf("Could not process scheduled cancellation: %v", err)
			return simpleResponse(500), nil
		}
		log.Infof("Scheduled cancellation recorded. subscription_id: %s", sub.ID)
		return simpleResponse(200), nil
	}

	// Case 2: Plan change — resolve plan type from price ID, then fall back to custom_data
	var newPlanType account.PlanType

	// Priority 1: Match items[].price.id against configured price IDs
	if len(sub.Items) > 0 && sub.Items[0].Price.ID != "" {
		pt, err := h.paddleHandler.GetPlanTypeByPriceID(sub.Items[0].Price.ID)
		if err == nil {
			newPlanType = pt
		} else {
			log.Debugf("Price ID not mapped, trying custom_data fallback. price_id: %s", sub.Items[0].Price.ID)
		}
	}

	// Priority 2: Fall back to custom_data.plan_type
	if newPlanType == "" && sub.CustomData != nil && sub.CustomData.PlanType != "" {
		pt := account.PlanType(sub.CustomData.PlanType)
		if _, ok := account.PlanTokenMap[pt]; ok {
			newPlanType = pt
		}
	}

	// Neither resolved — skip
	if newPlanType == "" {
		log.Infof("Could not resolve plan type, skipping. subscription_id: %s", sub.ID)
		return simpleResponse(200), nil
	}

	log.Infof("Processing subscription update. subscription_id: %s, new_plan_type: %s", sub.ID, newPlanType)
	if err := h.accountHandler.PaddleSubscriptionUpdate(ctx, sub.ID, newPlanType, event.EventID); err != nil {
		log.Errorf("Could not process subscription update: %v", err)
		return simpleResponse(500), nil
	}
	log.Infof("Subscription update completed. subscription_id: %s, new_plan_type: %s", sub.ID, newPlanType)
	return simpleResponse(200), nil
}

// handlePaddleSubscriptionCanceled handles subscription.canceled events.
func (h *listenHandler) handlePaddleSubscriptionCanceled(ctx context.Context, event *paddleEvent) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "handlePaddleSubscriptionCanceled",
		"event_id": event.EventID,
	})

	var sub paddleSubscriptionData
	if err := json.Unmarshal(event.Data, &sub); err != nil {
		log.Errorf("Could not parse subscription data: %v", err)
		return simpleResponse(400), nil
	}

	log.Infof("Processing subscription cancel. subscription_id: %s", sub.ID)
	if err := h.accountHandler.PaddleSubscriptionCancel(ctx, sub.ID, event.EventID); err != nil {
		log.Errorf("Could not process subscription cancel: %v", err)
		return simpleResponse(500), nil
	}
	log.Infof("Subscription cancel completed. subscription_id: %s", sub.ID)
	return simpleResponse(200), nil
}

// handlePaddleTransactionRefunded handles transaction.refunded events.
// Uses the adjustments array to determine the actual refund amount (sum of all adjustments).
// Falls back to paddle_subscription_id lookup when custom_data is missing.
func (h *listenHandler) handlePaddleTransactionRefunded(ctx context.Context, event *paddleEvent) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "handlePaddleTransactionRefunded",
		"event_id": event.EventID,
	})

	var txn paddleTransactionData
	if err := json.Unmarshal(event.Data, &txn); err != nil {
		log.Errorf("Could not parse transaction data: %v", err)
		return simpleResponse(400), nil
	}

	// Parse refund amount from adjustments array.
	// Each adjustment entry represents a refund; sum them for the total refund amount.
	if len(txn.Adjustments) == 0 {
		log.Errorf("No adjustments found in transaction.refunded event. event_id: %s", event.EventID)
		return simpleResponse(400), nil
	}

	var totalRefundMicros int64
	for _, adj := range txn.Adjustments {
		micros, err := parsePaddleCentsToMicros(adj.Totals.Total)
		if err != nil {
			log.Errorf("Could not parse adjustment amount: %v", err)
			return simpleResponse(400), nil
		}
		totalRefundMicros += micros
	}

	// Paddle may send negative adjustment amounts (money returned to customer).
	// PaddleRefund expects a positive amount, so take the absolute value.
	if totalRefundMicros < 0 {
		totalRefundMicros = -totalRefundMicros
	}

	// Resolve customer ID: prefer custom_data, fall back to paddle_subscription_id lookup.
	customerID := uuid.Nil
	if txn.CustomData != nil && txn.CustomData.CustomerID != "" {
		customerID = uuid.FromStringOrNil(txn.CustomData.CustomerID)
	} else if txn.SubscriptionID != nil && *txn.SubscriptionID != "" {
		acc, err := h.accountHandler.GetByPaddleSubscriptionID(ctx, *txn.SubscriptionID)
		if err != nil {
			log.Errorf("Could not look up account by paddle_subscription_id for refund: %v", err)
			return simpleResponse(500), nil
		}
		customerID = acc.CustomerID
	}

	if customerID == uuid.Nil {
		log.Infof("Could not resolve customer for refund, skipping. transaction_id: %s", txn.ID)
		return simpleResponse(200), nil
	}

	log.Infof("Processing refund. transaction_id: %s, customer_id: %s, refund_micros: %d, adjustments: %d", txn.ID, customerID, totalRefundMicros, len(txn.Adjustments))
	if err := h.accountHandler.PaddleRefund(ctx, customerID, totalRefundMicros, event.EventID); err != nil {
		log.Errorf("Could not process refund: %v", err)
		return simpleResponse(500), nil
	}
	log.Infof("Refund completed. transaction_id: %s, customer_id: %s, refund_micros: %d", txn.ID, customerID, totalRefundMicros)
	return simpleResponse(200), nil
}
