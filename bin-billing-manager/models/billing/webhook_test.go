package billing

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	id := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002")
	accountID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000003")
	referenceID := uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000004")
	idempotencyKey := uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000005")

	tmBillingStart := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tmBillingEnd := time.Date(2030, 1, 2, 3, 9, 5, 0, time.UTC)
	tmCreate := time.Date(2030, 1, 2, 3, 10, 0, 0, time.UTC)
	tmUpdate := time.Date(2030, 1, 2, 3, 11, 0, 0, time.UTC)
	tmDelete := time.Date(2030, 1, 2, 3, 12, 0, 0, time.UTC)

	// every field carries a value that differs from the others
	src := &Billing{
		Identity:              commonidentity.Identity{ID: id, CustomerID: customerID},
		AccountID:             accountID,
		TransactionType:       TransactionTypeRefund,
		Status:                StatusEnd,
		ReferenceType:         ReferenceTypeCall,
		ReferenceID:           referenceID,
		CostType:              CostTypeCallPSTNOutgoing,
		UsageDuration:         61,
		BillableUnits:         2,
		RateTokenPerUnit:      7,
		RateCreditPerUnit:     8,
		AmountToken:           -9,
		AmountCredit:          -10,
		BalanceTokenSnapshot:  1100,
		BalanceCreditSnapshot: 1200,
		IdempotencyKey:        idempotencyKey,
		TMBillingStart:        &tmBillingStart,
		TMBillingEnd:          &tmBillingEnd,
		TMCreate:              &tmCreate,
		TMUpdate:              &tmUpdate,
		TMDelete:              &tmDelete,
	}

	expect := &WebhookMessage{
		Identity:              commonidentity.Identity{ID: uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001"), CustomerID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002")},
		AccountID:             uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000003"),
		TransactionType:       "refund",
		Status:                "end",
		ReferenceType:         "call",
		ReferenceID:           uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000004"),
		CostType:              "call_pstn_outgoing",
		UsageDuration:         61,
		BillableUnits:         2,
		RateTokenPerUnit:      7,
		RateCreditPerUnit:     8,
		AmountToken:           -9,
		AmountCredit:          -10,
		BalanceTokenSnapshot:  1100,
		BalanceCreditSnapshot: 1200,
		IdempotencyKey:        uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000005"),
		TMBillingStart:        timePtr(2030, 1, 2, 3, 4, 5),
		TMBillingEnd:          timePtr(2030, 1, 2, 3, 9, 5),
		TMCreate:              timePtr(2030, 1, 2, 3, 10, 0),
		TMUpdate:              timePtr(2030, 1, 2, 3, 11, 0),
		TMDelete:              timePtr(2030, 1, 2, 3, 12, 0),
	}

	res := src.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot:    %+v", expect, res)
	}
}

func timePtr(y int, mo time.Month, d, h, mi, s int) *time.Time {
	tm := time.Date(y, mo, d, h, mi, s, 0, time.UTC)
	return &tm
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC)
	src := &Billing{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002"),
		},
		AccountID:    uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000003"),
		Status:       StatusProgressing,
		AmountCredit: -1500,
		TMCreate:     &tmCreate,
	}

	data, err := src.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Wrong match. the event is not valid JSON: %v", err)
	}

	expect := map[string]any{
		"id":            "d1000000-0000-0000-0000-000000000001",
		"customer_id":   "d2000000-0000-0000-0000-000000000002",
		"account_id":    "d3000000-0000-0000-0000-000000000003",
		"status":        "progressing",
		"amount_credit": float64(-1500),
		"tm_create":     "2030-05-06T07:08:09Z",
		"tm_delete":     nil,
	}
	for k, v := range expect {
		if !reflect.DeepEqual(got[k], v) {
			t.Errorf("Wrong match for %s. expect: %v, got: %v", k, v, got[k])
		}
	}
}
