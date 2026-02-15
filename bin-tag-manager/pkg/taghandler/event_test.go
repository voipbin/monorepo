package taghandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/dbhandler"
)

func Test_EventCustomerDeleted(t *testing.T) {
	tests := []struct {
		name     string
		customer *cmcustomer.Customer
		tags     []*tag.Tag
		listErr  error
	}{
		{
			name: "deletes_all_customer_tags",
			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			tags: []*tag.Tag{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a0c95b3e-2a00-11ee-a3cd-3307849aa505"),
						CustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
					},
					Name: "tag1",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b0c95b3e-2a00-11ee-a3cd-3307849aa505"),
						CustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
					},
					Name: "tag2",
				},
			},
			listErr: nil,
		},
		{
			name: "handles_list_error",
			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("b082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			tags:    nil,
			listErr: fmt.Errorf("database error"),
		},
		{
			name: "no_tags_to_delete",
			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("c082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			tags:    []*tag.Tag{},
			listErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			filters := map[tag.Field]any{
				tag.FieldCustomerID: tt.customer.ID,
				tag.FieldDeleted:    false,
			}

			mockUtil.EXPECT().TimeGetCurTime().Return("")
			mockDB.EXPECT().TagList(ctx, uint64(999), gomock.Any(), filters).Return(tt.tags, tt.listErr)

			if tt.listErr == nil {
				for _, tg := range tt.tags {
					mockDB.EXPECT().TagDelete(ctx, tg.ID).Return(nil)
					mockDB.EXPECT().TagGet(ctx, tg.ID).Return(tg, nil)
					mockNotify.EXPECT().PublishEvent(ctx, tag.EventTypeTagDeleted, tg)
				}
			}

			err := h.EventCustomerDeleted(ctx, tt.customer)
			if err != nil {
				t.Errorf("EventCustomerDeleted should not return error, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerDeleted_DeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := tagHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyhandler: mockNotify,
	}
	ctx := context.Background()

	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
	}

	tags := []*tag.Tag{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("a0c95b3e-2a00-11ee-a3cd-3307849aa505"),
				CustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},
			Name: "tag1",
		},
	}

	filters := map[tag.Field]any{
		tag.FieldCustomerID: customer.ID,
		tag.FieldDeleted:    false,
	}

	mockUtil.EXPECT().TimeGetCurTime().Return("")
	mockDB.EXPECT().TagList(ctx, uint64(999), gomock.Any(), filters).Return(tags, nil)
	mockDB.EXPECT().TagDelete(ctx, tags[0].ID).Return(fmt.Errorf("delete failed"))

	err := h.EventCustomerDeleted(ctx, customer)
	if err != nil {
		t.Errorf("EventCustomerDeleted should not return error even when delete fails, got: %v", err)
	}
}
