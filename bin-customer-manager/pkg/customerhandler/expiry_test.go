package customerhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_cleanupFrozenExpired(t *testing.T) {
	tests := []struct {
		name string

		responseFrozenExpired []*customer.Customer
		responseFrozenErr     error

		expectAnonymize       bool
		anonymizeErr          error
		expectGet             bool
		getResponse           *customer.Customer
		getErr                error
		expectPublish         bool
	}{
		{
			name: "expired frozen customers - anonymized and event published",
			responseFrozenExpired: []*customer.Customer{
				{
					ID:    uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
					Email: "test@example.com",
				},
			},
			responseFrozenErr: nil,
			expectAnonymize:   true,
			anonymizeErr:      nil,
			expectGet:         true,
			getResponse: &customer.Customer{
				ID:    uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Name:  "deleted_user_4cd23368",
				Email: "deleted_4cd23368@removed.voipbin.net",
			},
			getErr:        nil,
			expectPublish: true,
		},
		{
			name:                  "no frozen customers - no action",
			responseFrozenExpired: []*customer.Customer{},
			responseFrozenErr:     nil,
			expectAnonymize:       false,
			expectGet:             false,
			expectPublish:         false,
		},
		{
			name:                  "list frozen expired error - no action",
			responseFrozenExpired: nil,
			responseFrozenErr:     fmt.Errorf("db error"),
			expectAnonymize:       false,
			expectGet:             false,
			expectPublish:         false,
		},
		{
			name: "anonymize error - continues to next",
			responseFrozenExpired: []*customer.Customer{
				{
					ID:    uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
					Email: "test@example.com",
				},
			},
			responseFrozenErr: nil,
			expectAnonymize:   true,
			anonymizeErr:      fmt.Errorf("anonymize error"),
			expectGet:         false,
			expectPublish:     false,
		},
		{
			name: "get after anonymize error - continues to next",
			responseFrozenExpired: []*customer.Customer{
				{
					ID:    uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
					Email: "test@example.com",
				},
			},
			responseFrozenErr: nil,
			expectAnonymize:   true,
			anonymizeErr:      nil,
			expectGet:         true,
			getResponse:       nil,
			getErr:            fmt.Errorf("get error"),
			expectPublish:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerListFrozenExpired(gomock.Any(), gomock.Any()).Return(tt.responseFrozenExpired, tt.responseFrozenErr)

			if tt.expectAnonymize {
				id := tt.responseFrozenExpired[0].ID
				shortID := id.String()[:8]
				anonName := fmt.Sprintf("deleted_user_%s", shortID)
				anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)
				mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id, anonName, anonEmail).Return(tt.anonymizeErr)
			}

			if tt.expectGet {
				id := tt.responseFrozenExpired[0].ID
				mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(tt.getResponse, tt.getErr)
			}

			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, tt.getResponse).Return()
			}

			h.cleanupFrozenExpired(ctx)
		})
	}
}

func Test_cleanupFrozenExpired_MultipleCustomers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id1 := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")
	id2 := uuid.FromStringOrNil("5de34479-8dc8-22fd-a577-9429fg6b8236")

	frozenCustomers := []*customer.Customer{
		{ID: id1, Email: "user1@example.com"},
		{ID: id2, Email: "user2@example.com"},
	}

	anonymized1 := &customer.Customer{
		ID:    id1,
		Name:  "deleted_user_" + id1.String()[:8],
		Email: "deleted_" + id1.String()[:8] + "@removed.voipbin.net",
	}
	anonymized2 := &customer.Customer{
		ID:    id2,
		Name:  "deleted_user_" + id2.String()[:8],
		Email: "deleted_" + id2.String()[:8] + "@removed.voipbin.net",
	}

	mockDB.EXPECT().CustomerListFrozenExpired(gomock.Any(), gomock.Any()).Return(frozenCustomers, nil)

	// First customer
	mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id1, "deleted_user_"+id1.String()[:8], "deleted_"+id1.String()[:8]+"@removed.voipbin.net").Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id1).Return(anonymized1, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, anonymized1).Return()

	// Second customer
	mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id2, "deleted_user_"+id2.String()[:8], "deleted_"+id2.String()[:8]+"@removed.voipbin.net").Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id2).Return(anonymized2, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, anonymized2).Return()

	h.cleanupFrozenExpired(ctx)
}

func Test_expiryConstants(t *testing.T) {
	if expiryCheckInterval != 24*time.Hour {
		t.Errorf("expiryCheckInterval = %v, expected %v", expiryCheckInterval, 24*time.Hour)
	}
	if gracePeriod != 30*24*time.Hour {
		t.Errorf("gracePeriod = %v, expected %v", gracePeriod, 30*24*time.Hour)
	}
}
