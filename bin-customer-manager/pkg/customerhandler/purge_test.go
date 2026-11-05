package customerhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_purgeConstants(t *testing.T) {
	if expiredRetention != 30*24*time.Hour {
		t.Errorf("expiredRetention = %v, expected %v", expiredRetention, 30*24*time.Hour)
	}
	if purgeBatchSize != 100 {
		t.Errorf("purgeBatchSize = %d, expected %d", purgeBatchSize, 100)
	}
}

func newPurgeTestCustomer(id string) *customer.Customer {
	updated := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	return &customer.Customer{
		ID:               uuid.FromStringOrNil(id),
		Name:             "orig name",
		Email:            "orig@example.com",
		PhoneNumber:      "+15550100",
		Address:          "orig address",
		WebhookURI:       "https://orig.example.com/hook",
		BillingAccountID: uuid.FromStringOrNil("b1111111-0000-4000-8000-000000000001"),
		Status:           customer.StatusExpired,
		TMUpdate:         &updated,
	}
}

func anonymizedNameEmail(id uuid.UUID) (string, string) {
	shortID := id.String()[:8]

	return fmt.Sprintf("deleted_user_%s", shortID), fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)
}

func Test_PurgeExpiredUnverified(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	eventTime := now.Add(time.Second)

	id1 := "4cd23368-7cb7-11ec-9466-8318ef5a7125"
	id2 := "5de34479-7cb7-11ec-9466-8318ef5a7125"

	tests := []struct {
		name string

		listed  []*customer.Customer
		listErr error

		// anonymizeErr is the result of the anonymize for the listed row with the same index
		anonymizeErr []error

		expectPurgedIDs []string
		expectPurged    int
		expectErr       bool
	}{
		{
			name:            "one customer is purged and the event is published",
			listed:          []*customer.Customer{newPurgeTestCustomer(id1)},
			anonymizeErr:    []error{nil},
			expectPurgedIDs: []string{id1},
			expectPurged:    1,
		},
		{
			name:         "empty list",
			listed:       []*customer.Customer{},
			expectPurged: 0,
		},
		{
			name:      "list error is returned",
			listErr:   fmt.Errorf("list error"),
			expectErr: true,
		},
		{
			name:            "ErrNotFound is skipped without an event and the next customer is processed",
			listed:          []*customer.Customer{newPurgeTestCustomer(id1), newPurgeTestCustomer(id2)},
			anonymizeErr:    []error{dbhandler.ErrNotFound, nil},
			expectPurgedIDs: []string{id2},
			expectPurged:    1,
		},
		{
			name:            "other anonymize error is skipped without an event and the next customer is processed",
			listed:          []*customer.Customer{newPurgeTestCustomer(id1), newPurgeTestCustomer(id2)},
			anonymizeErr:    []error{fmt.Errorf("exec failed"), nil},
			expectPurgedIDs: []string{id2},
			expectPurged:    1,
		},
		{
			name:            "two customers are purged",
			listed:          []*customer.Customer{newPurgeTestCustomer(id1), newPurgeTestCustomer(id2)},
			anonymizeErr:    []error{nil, nil},
			expectPurgedIDs: []string{id1, id2},
			expectPurged:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// the cutoff is exactly now - 30 days, taken once before the list
			mockUtil.EXPECT().TimeNow().Return(&now)
			expectCutoff := now.Add(-30 * 24 * time.Hour)
			mockDB.EXPECT().CustomerListExpiredUnverified(gomock.Any(), expectCutoff, uint64(100)).Return(tt.listed, tt.listErr)

			purgedIDs := map[string]bool{}
			for _, id := range tt.expectPurgedIDs {
				purgedIDs[id] = true
			}

			var calls []any
			for i, c := range tt.listed {
				anonName, anonEmail := anonymizedNameEmail(c.ID)
				calls = append(calls, mockDB.EXPECT().CustomerAnonymizeExpiredUnverified(gomock.Any(), c.ID, anonName, anonEmail).Return(tt.anonymizeErr[i]))

				if !purgedIDs[c.ID.String()] {
					continue
				}

				calls = append(calls, mockUtil.EXPECT().TimeNow().Return(&eventTime))

				expectEvent := *c
				expectEvent.Name = anonName
				expectEvent.Email = anonEmail
				expectEvent.PhoneNumber = ""
				expectEvent.Address = ""
				expectEvent.WebhookURI = ""
				expectEvent.Status = customer.StatusDeleted
				expectEvent.TMDelete = &eventTime
				expectEvent.TMUpdate = &eventTime
				calls = append(calls, mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, &expectEvent).Return())
			}
			gomock.InOrder(calls...)

			purged, err := h.PurgeExpiredUnverified(ctx)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if purged != tt.expectPurged {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectPurged, purged)
			}
		})
	}
}

// Test_PurgeExpiredUnverified_eventPayload pins what the published event
// carries: the anonymized values and none of the original personal data, and the
// listed row stays unmodified.
func Test_PurgeExpiredUnverified_eventPayload(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	eventTime := now.Add(time.Second)

	listed := newPurgeTestCustomer("4cd23368-7cb7-11ec-9466-8318ef5a7125")
	original := *listed

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerHandler{utilHandler: mockUtil, db: mockDB, notifyHandler: mockNotify}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CustomerListExpiredUnverified(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{listed}, nil)
	mockDB.EXPECT().CustomerAnonymizeExpiredUnverified(gomock.Any(), listed.ID, gomock.Any(), gomock.Any()).Return(nil)
	mockUtil.EXPECT().TimeNow().Return(&eventTime)

	var got *customer.Customer
	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, data any) {
			got = data.(*customer.Customer)
		})

	if _, err := h.PurgeExpiredUnverified(context.Background()); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if got == nil {
		t.Fatalf("Wrong match. expect an event, got none")
	}
	if got.ID != original.ID || got.BillingAccountID != original.BillingAccountID {
		t.Errorf("Wrong match. expect id and billing account id kept, got: %v, %v", got.ID, got.BillingAccountID)
	}
	if got.Status != customer.StatusDeleted {
		t.Errorf("Wrong match. expect status: %v, got: %v", customer.StatusDeleted, got.Status)
	}
	if got.TMDelete == nil || !got.TMDelete.Equal(eventTime) || got.TMUpdate == nil || !got.TMUpdate.Equal(eventTime) {
		t.Errorf("Wrong match. expect tm_delete and tm_update: %v, got: %v, %v", eventTime, got.TMDelete, got.TMUpdate)
	}
	for _, v := range []string{original.Name, original.Email, original.PhoneNumber, original.Address, original.WebhookURI} {
		if got.Name == v || got.Email == v || got.PhoneNumber == v || got.Address == v || got.WebhookURI == v {
			t.Errorf("Wrong match. expect no original personal data in the event, found: %q", v)
		}
	}

	if !reflect.DeepEqual(*listed, original) {
		t.Errorf("Wrong match. expect the listed row unmodified.\nexpect: %v\ngot: %v", original, *listed)
	}
}
