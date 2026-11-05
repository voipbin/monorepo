package billinghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_messageBillingTypes(t *testing.T) {
	tests := []struct {
		name string

		messageType mmmessage.Type

		expectReference billing.ReferenceType
		expectCost      billing.CostType
	}{
		{"sms", mmmessage.TypeSMS, billing.ReferenceTypeSMS, billing.CostTypeSMS},
		{"empty type keeps the sms pair", mmmessage.Type(""), billing.ReferenceTypeSMS, billing.CostTypeSMS},
		{"mms", mmmessage.TypeMMS, billing.ReferenceTypeMMS, billing.CostTypeMMS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reference, cost := messageBillingTypes(&mmmessage.Message{Type: tt.messageType})
			if reference != tt.expectReference || cost != tt.expectCost {
				t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", tt.expectReference, tt.expectCost, reference, cost)
			}
		})
	}
}

// Test_EventMMMessageCreated_mms checks that an mms message is charged per target with the mms
// reference type, cost type and rate, never with the sms pair.
func Test_EventMMMessageCreated_mms(t *testing.T) {
	h, mockDB, mockNotify, mockAccount := newTestHandler(t)

	messageID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000002")
	accountID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000003")
	billID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000004")
	tmCreate := time.Date(2026, 11, 6, 1, 2, 3, 0, time.UTC)

	m := &mmmessage.Message{
		Identity: commonidentity.Identity{ID: messageID, CustomerID: customerID},
		Type:     mmmessage.TypeMMS,
		Source:   &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+100****0001"},
		Targets:  []mmtarget.Target{{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+100****0002"}}},
		TMCreate: &tmCreate,
	}
	targetRefID := uuid.NewV5(messageID, fmt.Sprintf("target-%d", 0))

	created := &billing.Billing{
		Identity:          commonidentity.Identity{ID: billID},
		AccountID:         accountID,
		Status:            billing.StatusProgressing,
		ReferenceType:     billing.ReferenceTypeMMS,
		CostType:          billing.CostTypeMMS,
		RateCreditPerUnit: billing.DefaultCreditPerUnitMMS,
	}
	consumed := *created
	consumed.Status = billing.StatusEnd

	ctx := context.Background()
	mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeMMS, targetRefID).Return(nil, dbhandler.ErrNotFound)
	mockAccount.EXPECT().GetByCustomerID(ctx, customerID).Return(&account.Account{Identity: commonidentity.Identity{ID: accountID, CustomerID: customerID}}, nil)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(billID)
	mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, b *billing.Billing) error {
		if b.ReferenceType != billing.ReferenceTypeMMS || b.CostType != billing.CostTypeMMS || b.RateCreditPerUnit != 30000 {
			t.Errorf("Wrong billing record. reference: %s, cost: %s, rate: %d", b.ReferenceType, b.CostType, b.RateCreditPerUnit)
		}
		return nil
	})
	mockDB.EXPECT().BillingGet(ctx, billID).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, created)
	mockDB.EXPECT().BillingConsumeAndRecord(ctx, created, accountID, 1, 0, billing.GetCostInfo(billing.CostTypeMMS), gomock.Any()).Return(&consumed, nil)
	mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, &consumed)

	if err := h.EventMMMessageCreated(ctx, m); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_BillingStart_retry_mms checks that an unfinished mms billing is finished on retry like sms,
// and that an unknown reference type is still rejected.
func Test_BillingStart_retry_mms(t *testing.T) {
	customerID := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000002")
	accountID := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000003")
	billID := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000004")
	start := time.Date(2026, 11, 6, 5, 6, 7, 0, time.UTC)

	t.Run("retry of an unfinished mms billing ends it", func(t *testing.T) {
		h, mockDB, mockNotify, _ := newTestHandler(t)

		existing := &billing.Billing{
			Identity:      commonidentity.Identity{ID: billID},
			AccountID:     accountID,
			Status:        billing.StatusProgressing,
			ReferenceType: billing.ReferenceTypeMMS,
			CostType:      billing.CostTypeMMS,
		}
		ended := *existing
		ended.Status = billing.StatusEnd

		mockDB.EXPECT().BillingGetByReferenceTypeAndID(gomock.Any(), billing.ReferenceTypeMMS, refID).Return(existing, nil)
		mockDB.EXPECT().BillingConsumeAndRecord(gomock.Any(), existing, accountID, 1, 0, billing.GetCostInfo(billing.CostTypeMMS), &start).Return(&ended, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), billing.EventTypeBillingUpdated, &ended)

		err := h.BillingStart(context.Background(), customerID, billing.ReferenceTypeMMS, refID, billing.CostTypeMMS, &start, &commonaddress.Address{}, &commonaddress.Address{})
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}
