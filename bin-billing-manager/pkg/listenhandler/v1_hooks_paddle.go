package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
}

// paddleSubscriptionData is the minimal subscription payload.
type paddleSubscriptionData struct {
	ID         string            `json:"id"`
	CustomerID string            `json:"customer_id"` // Paddle customer ID
	CustomData *paddleCustomData `json:"custom_data"`
	Items      []struct {
		Price struct {
			ProductID string `json:"product_id"`
		} `json:"price"`
	} `json:"items"`
}

// parsePaddleAmountToMicros converts a Paddle decimal amount string to micros.
// Paddle v2 sends amounts as decimal strings: "10.00" = $10.00 = 10,000,000 micros.
// Uses integer-only arithmetic to avoid floating-point precision issues in billing.
func parsePaddleAmountToMicros(amountStr string) (int64, error) {
	// Split on decimal point for integer-only arithmetic
	parts := strings.SplitN(amountStr, ".", 2)

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse amount %q: %w", amountStr, err)
	}

	var cents int64
	if len(parts) == 2 {
		// Pad or truncate fractional part to exactly 2 digits
		frac := parts[1]
		if len(frac) == 0 {
			cents = 0
		} else if len(frac) == 1 {
			c, err := strconv.ParseInt(frac, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse fractional amount %q: %w", amountStr, err)
			}
			cents = c * 10
		} else {
			c, err := strconv.ParseInt(frac[:2], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse fractional amount %q: %w", amountStr, err)
			}
			cents = c
		}
	}

	return dollars*1_000_000 + cents*10_000, nil
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

	log.WithField("event_type", event.EventType).Debugf("Received Paddle event. event_id: %s", event.EventID)

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
		log.Debugf("Transaction is subscription renewal. subscription_id: %s", *txn.SubscriptionID)
		if err := h.accountHandler.PaddleSubscriptionRenew(ctx, *txn.SubscriptionID, event.EventID); err != nil {
			log.Errorf("Could not process subscription renewal: %v", err)
			return simpleResponse(500), nil
		}
		return simpleResponse(200), nil
	}

	// One-time credit purchase: needs custom_data with customer_id
	if txn.CustomData == nil || txn.CustomData.CustomerID == "" {
		log.Infof("Missing customer_id in custom_data, skipping. event_id: %s", event.EventID)
		return simpleResponse(200), nil
	}

	customerID := uuid.FromStringOrNil(txn.CustomData.CustomerID)
	if customerID == uuid.Nil {
		log.Errorf("Invalid customer_id in custom_data: %s", txn.CustomData.CustomerID)
		return simpleResponse(400), nil
	}

	amountMicros, err := parsePaddleAmountToMicros(txn.Details.Totals.Total)
	if err != nil {
		log.Errorf("Could not parse transaction amount: %v", err)
		return simpleResponse(400), nil
	}

	if err := h.accountHandler.PaddleCreditTopUp(ctx, customerID, amountMicros, event.EventID); err != nil {
		log.Errorf("Could not process credit top-up: %v", err)
		return simpleResponse(500), nil
	}
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
		log.Infof("Missing customer_id in custom_data, skipping. event_id: %s", event.EventID)
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

	if err := h.accountHandler.PaddleSubscriptionCreate(ctx, customerID, planType, sub.ID, sub.CustomerID, event.EventID); err != nil {
		log.Errorf("Could not process subscription create: %v", err)
		return simpleResponse(500), nil
	}
	return simpleResponse(200), nil
}

// handlePaddleSubscriptionUpdated handles subscription.updated events (plan change).
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

	if sub.CustomData == nil || sub.CustomData.PlanType == "" {
		log.Infof("Missing plan_type in custom_data, skipping. event_id: %s", event.EventID)
		return simpleResponse(200), nil
	}

	newPlanType := account.PlanType(sub.CustomData.PlanType)
	if _, ok := account.PlanTokenMap[newPlanType]; !ok {
		log.Errorf("Unknown plan_type in custom_data: %s", sub.CustomData.PlanType)
		return simpleResponse(400), nil
	}

	if err := h.accountHandler.PaddleSubscriptionUpdate(ctx, sub.ID, newPlanType, event.EventID); err != nil {
		log.Errorf("Could not process subscription update: %v", err)
		return simpleResponse(500), nil
	}
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

	if err := h.accountHandler.PaddleSubscriptionCancel(ctx, sub.ID, event.EventID); err != nil {
		log.Errorf("Could not process subscription cancel: %v", err)
		return simpleResponse(500), nil
	}
	return simpleResponse(200), nil
}

// handlePaddleTransactionRefunded handles transaction.refunded events.
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

	if txn.CustomData == nil || txn.CustomData.CustomerID == "" {
		log.Infof("Missing customer_id in custom_data, skipping. event_id: %s", event.EventID)
		return simpleResponse(200), nil
	}

	customerID := uuid.FromStringOrNil(txn.CustomData.CustomerID)
	if customerID == uuid.Nil {
		log.Errorf("Invalid customer_id in custom_data: %s", txn.CustomData.CustomerID)
		return simpleResponse(400), nil
	}

	amountMicros, err := parsePaddleAmountToMicros(txn.Details.Totals.Total)
	if err != nil {
		log.Errorf("Could not parse refund amount: %v", err)
		return simpleResponse(400), nil
	}

	if err := h.accountHandler.PaddleRefund(ctx, customerID, amountMicros, event.EventID); err != nil {
		log.Errorf("Could not process refund: %v", err)
		return simpleResponse(500), nil
	}
	return simpleResponse(200), nil
}
