package emailhandler

import (
	"context"
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

		responseSubject             string
		responseContent             string
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

			responseSubject:             "updated subject",
			responseContent:             "updated content",
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
				Subject: "updated subject",
				Content: "updated content",
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
				Subject: "updated subject",
				Content: "updated content",
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

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}

			ctx := context.Background()

			if tt.activeflowID != uuid.Nil {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.subject).Return(tt.responseSubject, nil)
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.content).Return(tt.responseContent, nil)
			}

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

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[string]string

		responseEmails []*email.Email
		expectRes      []*email.Email
	}{
		{
			name: "normal",

			token: "2025-03-14 03:23:20.995000",
			size:  10,
			filters: map[string]string{
				"deleted": "false",
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

			mockDB.EXPECT().EmailGets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseEmails, nil)

			res, err := h.Gets(ctx, tt.token, tt.size, tt.filters)
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
