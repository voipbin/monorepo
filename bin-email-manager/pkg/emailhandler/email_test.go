package emailhandler

import (
	"context"
	"errors"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		destinations []commonaddress.Address
		subject      string
		content      string
		attachments  []email.Attachment

		responseUUID                uuid.UUID
		responseProviderReferenceID string

		expectEmail *email.Email
		expectRes   *email.Email
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("86e92dc4-0083-11f0-a8c4-0f10cc7d6102"),
			activeflowID: uuid.FromStringOrNil("871bddaa-0083-11f0-8d2f-a32510551821"),
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			subject: "test subject",
			content: "test content",
			attachments: []email.Attachment{
				{
					ReferenceType: email.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("87363f1a-0083-11f0-a6e9-ff59ce3dd60d"),
				},
			},

			responseUUID:                uuid.FromStringOrNil("8756a2dc-0083-11f0-a7bd-bf369e143079"),
			responseProviderReferenceID: "877ba028-0083-11f0-9831-3b71bd00b552",

			expectEmail: &email.Email{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("8756a2dc-0083-11f0-a7bd-bf369e143079"),
					CustomerID: uuid.FromStringOrNil("86e92dc4-0083-11f0-a8c4-0f10cc7d6102"),
				},
				ActiveflowID: uuid.FromStringOrNil("871bddaa-0083-11f0-8d2f-a32510551821"),
				ProviderType: email.ProviderTypeSendgrid,
				Source:       defaultSource,
				Destinations: []commonaddress.Address{
					{
						Type:       commonaddress.TypeEmail,
						Target:     "test@voipbin.net",
						TargetName: "test name",
					},
				},
				Status:  email.StatusInitiated,
				Subject: "test subject",
				Content: "test content",
				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("87363f1a-0083-11f0-a6e9-ff59ce3dd60d"),
					},
				},
			},
			expectRes: &email.Email{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("8756a2dc-0083-11f0-a7bd-bf369e143079"),
					CustomerID: uuid.FromStringOrNil("86e92dc4-0083-11f0-a8c4-0f10cc7d6102"),
				},
				ActiveflowID: uuid.FromStringOrNil("871bddaa-0083-11f0-8d2f-a32510551821"),
				ProviderType: email.ProviderTypeSendgrid,
				Source:       defaultSource,
				Destinations: []commonaddress.Address{
					{
						Type:       commonaddress.TypeEmail,
						Target:     "test@voipbin.net",
						TargetName: "test name",
					},
				},
				Status:  email.StatusInitiated,
				Subject: "test subject",
				Content: "test content",
				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("87363f1a-0083-11f0-a6e9-ff59ce3dd60d"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)
			mockMailgun := NewMockEngineMailgun(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
				engineMailgun:  mockMailgun,
			}

			ctx := context.Background()

			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeEmail, "", len(tt.destinations)).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().EmailCreate(ctx, tt.expectEmail).Return(nil)
			mockDB.EXPECT().EmailGet(ctx, tt.responseUUID).Return(tt.expectEmail, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectEmail.CustomerID, email.EventTypeCreated, tt.expectEmail)

			mockSendgrid.EXPECT().Send(ctx, tt.expectEmail).Return(tt.responseProviderReferenceID, nil)
			mockDB.EXPECT().EmailUpdateProviderReferenceID(ctx, tt.expectEmail.ID, tt.responseProviderReferenceID).Return(nil)

			res, err := h.Create(ctx, tt.customerID, tt.activeflowID, tt.destinations, tt.subject, tt.content, tt.attachments)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}

func Test_Create_InsufficientBalance(t *testing.T) {
	tests := []struct {
		name         string
		customerID   uuid.UUID
		activeflowID uuid.UUID
		destinations []commonaddress.Address
	}{
		{
			name:         "insufficient balance returns error",
			customerID:   uuid.FromStringOrNil("86e92dc4-0083-11f0-a8c4-0f10cc7d6102"),
			activeflowID: uuid.FromStringOrNil("871bddaa-0083-11f0-8d2f-a32510551821"),
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeEmail, "", len(tt.destinations)).Return(false, nil)

			_, err := h.Create(ctx, tt.customerID, tt.activeflowID, tt.destinations, "test subject", "test content", []email.Attachment{})
			if err == nil {
				t.Errorf("Expected error for insufficient balance")
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[email.Field]any

		responseEmails []*email.Email
		expectRes      []*email.Email
	}{
		{
			name: "normal",

			token: "2025-03-14T03:23:20.995000Z",
			size:  10,
			filters: map[email.Field]any{
				email.FieldDeleted: "false",
			},

			responseEmails: []*email.Email{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("79bdc86e-0086-11f0-922f-eb1723e9fa90"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("79e2dca8-0086-11f0-ad3a-e75f1e63d84f"),
					},
				},
			},
			expectRes: []*email.Email{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("79bdc86e-0086-11f0-922f-eb1723e9fa90"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("79e2dca8-0086-11f0-ad3a-e75f1e63d84f"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailList(ctx, tt.token, tt.size, tt.filters).Return(tt.responseEmails, nil)

			res, err := h.List(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseEmail *email.Email
		expectRes     *email.Email
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b1109c4c-0086-11f0-86bd-7b43a2310a89"),

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b1109c4c-0086-11f0-86bd-7b43a2310a89"),
				},
			},
			expectRes: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b1109c4c-0086-11f0-86bd-7b43a2310a89"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().EmailGet(ctx, tt.id).Return(tt.responseEmail, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseEmail.CustomerID, email.EventTypeDeleted, tt.responseEmail)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseEmail *email.Email
		expectRes     *email.Email
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b13146fe-0086-11f0-9c44-277c939229a8"),

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b13146fe-0086-11f0-9c44-277c939229a8"),
				},
			},
			expectRes: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b13146fe-0086-11f0-9c44-277c939229a8"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailGet(ctx, tt.id).Return(tt.responseEmail, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		providerType email.ProviderType
		source       *commonaddress.Address
		destinations []commonaddress.Address
		subject      string
		content      string
		attachments  []email.Attachment

		responseUUID uuid.UUID

		expectEmail *email.Email
		expectRes   *email.Email
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("8069b352-0087-11f0-86bb-d71359e78f48"),
			activeflowID: uuid.FromStringOrNil("808a0422-0087-11f0-9410-2bc642df4cc0"),
			providerType: email.ProviderTypeSendgrid,
			source: &commonaddress.Address{
				Type:       commonaddress.TypeEmail,
				Target:     "test_sender@voipbin.net",
				TargetName: "test sender name",
			},
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			subject: "test subject",
			content: "test content",
			attachments: []email.Attachment{
				{
					ReferenceType: email.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("80f369a8-0087-11f0-9db8-c7c13b82654f"),
				},
			},

			responseUUID: uuid.FromStringOrNil("80ce4376-0087-11f0-8862-37ac95ef87e1"),

			expectEmail: &email.Email{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("80ce4376-0087-11f0-8862-37ac95ef87e1"),
					CustomerID: uuid.FromStringOrNil("8069b352-0087-11f0-86bb-d71359e78f48"),
				},
				ActiveflowID: uuid.FromStringOrNil("808a0422-0087-11f0-9410-2bc642df4cc0"),
				ProviderType: email.ProviderTypeSendgrid,
				Source: &commonaddress.Address{
					Type:       commonaddress.TypeEmail,
					Target:     "test_sender@voipbin.net",
					TargetName: "test sender name",
				},
				Destinations: []commonaddress.Address{
					{
						Type:       commonaddress.TypeEmail,
						Target:     "test@voipbin.net",
						TargetName: "test name",
					},
				},
				Status:  email.StatusInitiated,
				Subject: "test subject",
				Content: "test content",
				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("80f369a8-0087-11f0-9db8-c7c13b82654f"),
					},
				},
			},
			expectRes: &email.Email{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("80ce4376-0087-11f0-8862-37ac95ef87e1"),
					CustomerID: uuid.FromStringOrNil("8069b352-0087-11f0-86bb-d71359e78f48"),
				},
				ActiveflowID: uuid.FromStringOrNil("808a0422-0087-11f0-9410-2bc642df4cc0"),
				ProviderType: email.ProviderTypeSendgrid,
				Source: &commonaddress.Address{
					Type:       commonaddress.TypeEmail,
					Target:     "test_sender@voipbin.net",
					TargetName: "test sender name",
				},
				Destinations: []commonaddress.Address{
					{
						Type:       commonaddress.TypeEmail,
						Target:     "test@voipbin.net",
						TargetName: "test name",
					},
				},
				Status:  email.StatusInitiated,
				Subject: "test subject",
				Content: "test content",
				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("80f369a8-0087-11f0-9db8-c7c13b82654f"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().EmailCreate(ctx, tt.expectEmail).Return(nil)
			mockDB.EXPECT().EmailGet(ctx, tt.responseUUID).Return(tt.expectEmail, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectEmail.CustomerID, email.EventTypeCreated, tt.expectEmail)

			res, err := h.create(ctx, tt.customerID, tt.activeflowID, tt.providerType, tt.source, tt.destinations, tt.subject, tt.content, tt.attachments)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}

func Test_UpdateProviderReferenceID(t *testing.T) {

	tests := []struct {
		name string

		id                  uuid.UUID
		providerReferenceID string
	}{
		{
			name: "normal",

			id:                  uuid.FromStringOrNil("d05971b0-0085-11f0-91f8-7399c2e87154"),
			providerReferenceID: "d07e7adc-0085-11f0-99f9-f75941de7e87",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailUpdateProviderReferenceID(ctx, tt.id, tt.providerReferenceID).Return(nil)

			if errUpdate := h.UpdateProviderReferenceID(ctx, tt.id, tt.providerReferenceID); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status email.Status

		responseEmail *email.Email
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("5c42e66e-0088-11f0-b5d6-7b34c7b33dab"),
			status: email.StatusDelivered,

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5c42e66e-0088-11f0-b5d6-7b34c7b33dab"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			mockDB.EXPECT().EmailUpdateStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().EmailGet(ctx, tt.id).Return(tt.responseEmail, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseEmail.CustomerID, email.EventTypeUpdated, tt.responseEmail)

			res, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseEmail) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseEmail, res)
			}
		})
	}
}

func Test_Create_InvalidEmail(t *testing.T) {
	tests := []struct {
		name         string
		destinations []commonaddress.Address
	}{
		{
			name: "fails_with_invalid_email",
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeEmail,
					Target: "invalid-email",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
			activeflowID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

			_, err := h.Create(ctx, customerID, activeflowID, tt.destinations, "subject", "content", []email.Attachment{})
			if err == nil {
				t.Errorf("Expected error for invalid email")
			}
		})
	}
}

func Test_List_DBError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "fails_when_db_fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			dbErr := errors.New("db error")

			mockDB.EXPECT().EmailList(ctx, "", uint64(10), gomock.Any()).Return(nil, dbErr)

			_, err := h.List(ctx, "", 10, map[email.Field]any{})
			if err == nil {
				t.Errorf("Expected error when DB fails")
			}
		})
	}
}

func Test_Delete_DBError(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "fails_when_delete_fails",
			id:   uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			dbErr := errors.New("db error")

			mockDB.EXPECT().EmailDelete(ctx, tt.id).Return(dbErr)

			_, err := h.Delete(ctx, tt.id)
			if err == nil {
				t.Errorf("Expected error when delete fails")
			}
		})
	}
}

func Test_Get_DBError(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "fails_when_get_fails",
			id:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			dbErr := errors.New("db error")

			mockDB.EXPECT().EmailGet(ctx, tt.id).Return(nil, dbErr)

			_, err := h.Get(ctx, tt.id)
			if err == nil {
				t.Errorf("Expected error when get fails")
			}
		})
	}
}

func Test_UpdateProviderReferenceID_DBError(t *testing.T) {
	tests := []struct {
		name                string
		id                  uuid.UUID
		providerReferenceID string
	}{
		{
			name:                "fails_when_update_fails",
			id:                  uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			providerReferenceID: "test-ref-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			dbErr := errors.New("db error")

			mockDB.EXPECT().EmailUpdateProviderReferenceID(ctx, tt.id, tt.providerReferenceID).Return(dbErr)

			err := h.UpdateProviderReferenceID(ctx, tt.id, tt.providerReferenceID)
			if err == nil {
				t.Errorf("Expected error when update fails")
			}
		})
	}
}

func Test_UpdateStatus_DBError(t *testing.T) {
	tests := []struct {
		name   string
		id     uuid.UUID
		status email.Status
	}{
		{
			name:   "fails_when_update_status_fails",
			id:     uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
			status: email.StatusDelivered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()
			dbErr := errors.New("db error")

			mockDB.EXPECT().EmailUpdateStatus(ctx, tt.id, tt.status).Return(dbErr)

			_, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err == nil {
				t.Errorf("Expected error when update status fails")
			}
		})
	}
}
