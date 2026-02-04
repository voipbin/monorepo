package contacthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_List(t *testing.T) {
	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[contact.Field]any

		responseContacts []*contact.Contact
	}{
		{
			name: "normal",

			size:  10,
			token: "2020-04-18T03:22:17.995000Z",
			filters: map[contact.Field]any{
				contact.FieldCustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
				contact.FieldDeleted:    false,
			},

			responseContacts: []*contact.Contact{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a0c95b3e-2a00-11ee-a3cd-3307849aa505"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseContacts, nil)
			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContacts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContacts, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactGet(ctx, tt.id).Return(tt.responseContact, nil)
			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContact, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContact, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {
	tests := []struct {
		name string

		contact *contact.Contact

		responseUUID    uuid.UUID
		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contact: &contact.Contact{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
			},

			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
					CustomerID: uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.responseUUID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

			res, err := h.Create(ctx, tt.contact)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.responseUUID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUUID, res.ID)
			}
		})
	}
}

func Test_Update(t *testing.T) {
	tests := []struct {
		name string

		id     uuid.UUID
		fields map[contact.Field]any

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
			fields: map[contact.Field]any{
				contact.FieldFirstName:   "Updated",
				contact.FieldDisplayName: "Updated Name",
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f6a7ef6-2a01-11ee-8594-87f2ee5140ed"),
				},
				FirstName:   "Updated",
				DisplayName: "Updated Name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactUpdate(ctx, tt.id, tt.fields).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.id).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())
			res, err := h.Update(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContact, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContact, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// First ContactGet to verify existence
			mockDB.EXPECT().ContactGet(ctx, tt.id).Return(tt.responseContact, nil)
			mockDB.EXPECT().ContactDelete(ctx, tt.id).Return(nil)
			// Second ContactGet after deletion
			mockDB.EXPECT().ContactGet(ctx, tt.id).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactDeleted, gomock.Any())

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContact, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContact, res)
			}
		})
	}
}

func Test_AddPhoneNumber(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		phone     *contact.PhoneNumber

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			phone: &contact.PhoneNumber{
				Number:     "+1-555-123-4567",
				NumberE164: "+15551234567",
				Type:       "mobile",
				IsPrimary:  true,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
			mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.AddPhoneNumber(ctx, tt.contactID, tt.phone)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_AddEmail(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		email     *contact.Email

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			email: &contact.Email{
				Address:   "test@example.com",
				Type:      "work",
				IsPrimary: true,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"))
			mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.AddEmail(ctx, tt.contactID, tt.email)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_AddTag(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		tagID     uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
			tagID:     uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
					CustomerID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagAssignmentCreate(ctx, tt.contactID, tt.tagID).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.AddTag(ctx, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_LookupByPhone(t *testing.T) {
	tests := []struct {
		name string

		customerID  uuid.UUID
		phoneE164   string

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			phoneE164:   "+15551234567",

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactLookupByPhone(ctx, tt.customerID, tt.phoneE164).Return(tt.responseContact, nil)

			res, err := h.LookupByPhone(ctx, tt.customerID, tt.phoneE164)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContact, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContact, res)
			}
		})
	}
}

func Test_RemovePhoneNumber(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		phoneID   uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			phoneID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().PhoneNumberDelete(ctx, tt.phoneID).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.RemovePhoneNumber(ctx, tt.contactID, tt.phoneID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_RemoveEmail(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		emailID   uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			emailID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailDelete(ctx, tt.emailID).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.RemoveEmail(ctx, tt.contactID, tt.emailID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_RemoveTag(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		tagID     uuid.UUID

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			contactID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
			tagID:     uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
					CustomerID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TagAssignmentDelete(ctx, tt.contactID, tt.tagID).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.RemoveTag(ctx, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.contactID, res.ID)
			}
		})
	}
}

func Test_LookupByEmail(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		email      string

		responseContact *contact.Contact
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
			email:      "test@example.com",

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ContactLookupByEmail(ctx, tt.customerID, tt.email).Return(tt.responseContact, nil)

			res, err := h.LookupByEmail(ctx, tt.customerID, tt.email)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseContact, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseContact, res)
			}
		})
	}
}

func Test_EventCustomerDeleted(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			customer := &cmcustomer.Customer{
				ID: tt.customerID,
			}

			mockDB.EXPECT().ContactDeleteByCustomerID(ctx, tt.customerID).Return(nil)

			err := h.EventCustomerDeleted(ctx, customer)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestNewContactHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewContactHandler(nil, mockDB, mockNotify)
	if h == nil {
		t.Error("NewContactHandler() returned nil")
	}
}

// Error cases

func Test_Create_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
	}

	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"))
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	_, err := h.Create(ctx, c)
	if err == nil {
		t.Error("Create() expected error")
	}
}

func Test_Create_WithPhoneEmailTags(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
		PhoneNumbers: []contact.PhoneNumber{
			{Number: "+1-555-123-4567", NumberE164: "+15551234567"},
		},
		Emails: []contact.Email{
			{Address: "test@example.com"},
		},
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		},
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Test",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"))
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

func Test_Get_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	mockDB.EXPECT().ContactGet(ctx, id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Get(ctx, id)
	if err == nil {
		t.Error("Get() expected error")
	}
}

func Test_List_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	mockDB.EXPECT().ContactList(ctx, uint64(10), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	_, err := h.List(ctx, 10, "token", nil)
	if err == nil {
		t.Error("List() expected error")
	}
}

func Test_Update_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	fields := map[contact.Field]any{contact.FieldFirstName: "Updated"}

	mockDB.EXPECT().ContactUpdate(ctx, id, fields).Return(fmt.Errorf("database error"))

	_, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Error("Update() expected error")
	}
}

func Test_Delete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{ID: id},
	}

	mockDB.EXPECT().ContactGet(ctx, id).Return(responseContact, nil)
	mockDB.EXPECT().ContactDelete(ctx, id).Return(fmt.Errorf("database error"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Error("Delete() expected error")
	}
}

func Test_Delete_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	mockDB.EXPECT().ContactGet(ctx, id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Error("Delete() expected error for not found")
	}
}

func Test_AddPhoneNumber_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phone := &contact.PhoneNumber{Number: "+1-555-123-4567"}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	_, err := h.AddPhoneNumber(ctx, contactID, phone)
	if err == nil {
		t.Error("AddPhoneNumber() expected error")
	}
}

func Test_AddEmail_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	email := &contact.Email{Address: "test@example.com"}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	_, err := h.AddEmail(ctx, contactID, email)
	if err == nil {
		t.Error("AddEmail() expected error")
	}
}

func Test_AddTag_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tagID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, tagID).Return(fmt.Errorf("database error"))

	_, err := h.AddTag(ctx, contactID, tagID)
	if err == nil {
		t.Error("AddTag() expected error")
	}
}

func Test_RemovePhoneNumber_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phoneID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().PhoneNumberDelete(ctx, phoneID).Return(fmt.Errorf("database error"))

	_, err := h.RemovePhoneNumber(ctx, contactID, phoneID)
	if err == nil {
		t.Error("RemovePhoneNumber() expected error")
	}
}

func Test_RemoveEmail_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	emailID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().EmailDelete(ctx, emailID).Return(fmt.Errorf("database error"))

	_, err := h.RemoveEmail(ctx, contactID, emailID)
	if err == nil {
		t.Error("RemoveEmail() expected error")
	}
}

func Test_RemoveTag_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tagID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().TagAssignmentDelete(ctx, contactID, tagID).Return(fmt.Errorf("database error"))

	_, err := h.RemoveTag(ctx, contactID, tagID)
	if err == nil {
		t.Error("RemoveTag() expected error")
	}
}

func Test_EventCustomerDeleted_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customer := &cmcustomer.Customer{ID: customerID}

	mockDB.EXPECT().ContactDeleteByCustomerID(ctx, customerID).Return(fmt.Errorf("database error"))

	err := h.EventCustomerDeleted(ctx, customer)
	if err == nil {
		t.Error("EventCustomerDeleted() expected error")
	}
}

func Test_Create_GetAfterCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.Create(ctx, c)
	if err == nil {
		t.Error("Create() expected error when get after create fails")
	}
}

func Test_Update_GetAfterUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	fields := map[contact.Field]any{contact.FieldFirstName: "Updated"}

	mockDB.EXPECT().ContactUpdate(ctx, id, fields).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, id).Return(nil, fmt.Errorf("database error"))

	_, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Error("Update() expected error when get after update fails")
	}
}

func Test_Delete_GetAfterDeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{ID: id},
	}

	mockDB.EXPECT().ContactGet(ctx, id).Return(responseContact, nil)
	mockDB.EXPECT().ContactDelete(ctx, id).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, id).Return(nil, fmt.Errorf("database error"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Error("Delete() expected error when get after delete fails")
	}
}

func Test_AddPhoneNumber_ContactGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phone := &contact.PhoneNumber{Number: "+1-555-123-4567"}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("not found"))

	_, err := h.AddPhoneNumber(ctx, contactID, phone)
	if err == nil {
		t.Error("AddPhoneNumber() expected error when contact not found")
	}
}

func Test_AddPhoneNumber_GetAfterCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phone := &contact.PhoneNumber{Number: "+1-555-123-4567"}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.AddPhoneNumber(ctx, contactID, phone)
	if err == nil {
		t.Error("AddPhoneNumber() expected error when get after create fails")
	}
}

func Test_AddEmail_ContactGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	email := &contact.Email{Address: "test@example.com"}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("not found"))

	_, err := h.AddEmail(ctx, contactID, email)
	if err == nil {
		t.Error("AddEmail() expected error when contact not found")
	}
}

func Test_AddEmail_GetAfterCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	email := &contact.Email{Address: "test@example.com"}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.AddEmail(ctx, contactID, email)
	if err == nil {
		t.Error("AddEmail() expected error when get after create fails")
	}
}

func Test_AddTag_GetAfterCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tagID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, tagID).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.AddTag(ctx, contactID, tagID)
	if err == nil {
		t.Error("AddTag() expected error when get after create fails")
	}
}

func Test_RemovePhoneNumber_GetAfterDeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phoneID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().PhoneNumberDelete(ctx, phoneID).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.RemovePhoneNumber(ctx, contactID, phoneID)
	if err == nil {
		t.Error("RemovePhoneNumber() expected error when get after delete fails")
	}
}

func Test_RemoveEmail_GetAfterDeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	emailID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().EmailDelete(ctx, emailID).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.RemoveEmail(ctx, contactID, emailID)
	if err == nil {
		t.Error("RemoveEmail() expected error when get after delete fails")
	}
}

func Test_RemoveTag_GetAfterDeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tagID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	mockDB.EXPECT().TagAssignmentDelete(ctx, contactID, tagID).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("database error"))

	_, err := h.RemoveTag(ctx, contactID, tagID)
	if err == nil {
		t.Error("RemoveTag() expected error when get after delete fails")
	}
}

func Test_LookupByPhone_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	phoneE164 := "+15551234567"

	mockDB.EXPECT().ContactLookupByPhone(ctx, customerID, phoneE164).Return(nil, fmt.Errorf("not found"))

	_, err := h.LookupByPhone(ctx, customerID, phoneE164)
	if err == nil {
		t.Error("LookupByPhone() expected error")
	}
}

func Test_LookupByEmail_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	email := "test@example.com"

	mockDB.EXPECT().ContactLookupByEmail(ctx, customerID, email).Return(nil, fmt.Errorf("not found"))

	_, err := h.LookupByEmail(ctx, customerID, email)
	if err == nil {
		t.Error("LookupByEmail() expected error")
	}
}

func Test_LookupByEmail_NormalizesEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			CustomerID: customerID,
		},
	}

	// Expect normalized email (lowercase and trimmed)
	mockDB.EXPECT().ContactLookupByEmail(ctx, customerID, "test@example.com").Return(responseContact, nil)

	// Pass email with uppercase and spaces
	res, err := h.LookupByEmail(ctx, customerID, "  TEST@EXAMPLE.COM  ")
	if err != nil {
		t.Errorf("LookupByEmail() error = %v", err)
	}
	if res.ID != responseContact.ID {
		t.Errorf("LookupByEmail() ID = %v, want %v", res.ID, responseContact.ID)
	}
}

func Test_Create_WithDefaultSource(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
		// Source is empty, should default to "manual"
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Test",
		Source:    contact.SourceManual,
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.Source != contact.SourceManual {
		t.Errorf("Create() Source = %v, want %v", res.Source, contact.SourceManual)
	}
}

func Test_Create_WithExistingID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID, // ID already set
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
		Source:    "api",
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Test",
		Source:    "api",
	}

	// UUIDCreate should NOT be called since ID is already set
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

func Test_Create_WithPhoneEmailTagErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		FirstName: "Test",
		PhoneNumbers: []contact.PhoneNumber{
			{Number: "+1-555-123-4567", NumberE164: "+15551234567"},
		},
		Emails: []contact.Email{
			{Address: "test@example.com"},
		},
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		},
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Test",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"))
	// PhoneNumberCreate fails but Create should still succeed
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(fmt.Errorf("duplicate phone"))
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"))
	// EmailCreate fails but Create should still succeed
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(fmt.Errorf("duplicate email"))
	// TagAssignmentCreate fails but Create should still succeed
	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, gomock.Any()).Return(fmt.Errorf("tag error"))
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	// Create should succeed even if phone/email/tag creation fails
	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

// Test_Create_WithMultiplePhones tests creating a contact with multiple phone numbers
func Test_Create_WithMultiplePhones(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
		},
		FirstName: "Multiple",
		LastName:  "Phones",
		PhoneNumbers: []contact.PhoneNumber{
			{Number: "+1-555-111-1111", NumberE164: "+15551111111", Type: "mobile", IsPrimary: true},
			{Number: "+1-555-222-2222", NumberE164: "+15552222222", Type: "work", IsPrimary: false},
			{Number: "+1-555-333-3333", NumberE164: "+15553333333", Type: "home", IsPrimary: false},
		},
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Multiple",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	// Three phone numbers
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a4444444-4444-4444-4444-444444444444"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"))
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

// Test_Create_WithMultipleEmails tests creating a contact with multiple emails
func Test_Create_WithMultipleEmails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("b1111111-1111-1111-1111-111111111111")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("b2222222-2222-2222-2222-222222222222"),
		},
		FirstName: "Multiple",
		LastName:  "Emails",
		Emails: []contact.Email{
			{Address: "primary@example.com", Type: "work", IsPrimary: true},
			{Address: "secondary@example.com", Type: "personal", IsPrimary: false},
		},
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Multiple",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	// Two emails
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("b3333333-3333-3333-3333-333333333333"))
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("b4444444-4444-4444-4444-444444444444"))
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

// Test_Create_WithMultipleTags tests creating a contact with multiple tags
func Test_Create_WithMultipleTags(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("c2222222-2222-2222-2222-222222222222"),
		},
		FirstName: "Multiple",
		LastName:  "Tags",
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
			uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444"),
			uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555"),
		},
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: c.CustomerID,
		},
		FirstName: "Multiple",
	}

	mockUtil.EXPECT().UUIDCreate().Return(contactID)
	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	// Three tags
	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333")).Return(nil)
	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444")).Return(nil)
	mockDB.EXPECT().TagAssignmentCreate(ctx, contactID, uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555")).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, c)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}
	if res.ID != contactID {
		t.Errorf("Create() ID = %v, want %v", res.ID, contactID)
	}
}

// Test_LookupByEmail_WithSpaces tests that email lookup trims spaces
func Test_LookupByEmail_WithSpaces(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111")
	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
			CustomerID: customerID,
		},
	}

	// Expect normalized email (lowercase and trimmed)
	mockDB.EXPECT().ContactLookupByEmail(ctx, customerID, "user@domain.com").Return(responseContact, nil)

	// Pass email with leading/trailing spaces
	res, err := h.LookupByEmail(ctx, customerID, "   user@domain.com   ")
	if err != nil {
		t.Errorf("LookupByEmail() error = %v", err)
	}
	if res.ID != responseContact.ID {
		t.Errorf("LookupByEmail() ID = %v, want %v", res.ID, responseContact.ID)
	}
}

// Test_AddEmail_NormalizesAddress tests that email addresses are normalized
func Test_AddEmail_NormalizesAddress(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("e1111111-1111-1111-1111-111111111111")
	email := &contact.Email{
		Address:   "  USER@EXAMPLE.COM  ", // uppercase with spaces
		Type:      "work",
		IsPrimary: true,
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("e2222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("e3333333-3333-3333-3333-333333333333"))
	// Verify that the email address passed to EmailCreate is normalized
	mockDB.EXPECT().EmailCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, e *contact.Email) error {
		if e.Address != "user@example.com" {
			return fmt.Errorf("expected normalized email 'user@example.com', got '%s'", e.Address)
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.AddEmail(ctx, contactID, email)
	if err != nil {
		t.Errorf("AddEmail() error = %v", err)
	}
}

// Test_AddPhoneNumber_TrimsSpaces tests that phone numbers are trimmed
func Test_AddPhoneNumber_TrimsSpaces(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("f1111111-1111-1111-1111-111111111111")
	phone := &contact.PhoneNumber{
		Number:     "+1-555-123-4567",
		NumberE164: "  +15551234567  ", // with spaces
		Type:       "mobile",
		IsPrimary:  true,
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("f2222222-2222-2222-2222-222222222222"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("f3333333-3333-3333-3333-333333333333"))
	// Verify that the phone number is trimmed
	mockDB.EXPECT().PhoneNumberCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p *contact.PhoneNumber) error {
		if p.NumberE164 != "+15551234567" {
			return fmt.Errorf("expected trimmed phone '+15551234567', got '%s'", p.NumberE164)
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.AddPhoneNumber(ctx, contactID, phone)
	if err != nil {
		t.Errorf("AddPhoneNumber() error = %v", err)
	}
}

// Test_Update_EmptyFields tests update with empty fields map
func Test_Update_EmptyFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("01111111-1111-1111-1111-111111111111")
	fields := map[contact.Field]any{} // Empty map

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID: id,
		},
	}

	mockDB.EXPECT().ContactUpdate(ctx, id, fields).Return(nil)
	mockDB.EXPECT().ContactGet(ctx, id).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	res, err := h.Update(ctx, id, fields)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
	if res.ID != id {
		t.Errorf("Update() ID = %v, want %v", res.ID, id)
	}
}

// Test_List_EmptyResult tests listing when no contacts match
func Test_List_EmptyResult(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	filters := map[contact.Field]any{
		contact.FieldCustomerID: uuid.FromStringOrNil("02222222-2222-2222-2222-222222222222"),
	}

	mockDB.EXPECT().ContactList(ctx, uint64(10), "", filters).Return([]*contact.Contact{}, nil)

	res, err := h.List(ctx, 10, "", filters)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(res) != 0 {
		t.Errorf("List() count = %v, want 0", len(res))
	}
}

// Test_List_MultipleContacts tests listing multiple contacts
func Test_List_MultipleContacts(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("03333333-3333-3333-3333-333333333333")
	filters := map[contact.Field]any{
		contact.FieldCustomerID: customerID,
	}

	contacts := []*contact.Contact{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("04444444-4444-4444-4444-444444444444"),
				CustomerID: customerID,
			},
			FirstName: "First",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("05555555-5555-5555-5555-555555555555"),
				CustomerID: customerID,
			},
			FirstName: "Second",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("06666666-6666-6666-6666-666666666666"),
				CustomerID: customerID,
			},
			FirstName: "Third",
		},
	}

	mockDB.EXPECT().ContactList(ctx, uint64(10), "", filters).Return(contacts, nil)

	res, err := h.List(ctx, 10, "", filters)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(res) != 3 {
		t.Errorf("List() count = %v, want 3", len(res))
	}
}
