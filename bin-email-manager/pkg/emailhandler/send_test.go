package emailhandler

import (
	"context"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Send(t *testing.T) {

	tests := []struct {
		name string

		email *email.Email

		responseProviderReferenceID string
	}{
		{
			name: "normal",

			email: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a1776254-00c5-11f0-a3ac-cb1a5ddc75fd"),
				},
			},

			responseProviderReferenceID: "a1d1f67e-00c5-11f0-b69e-1fa7a77d151f",
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

			mockSendgrid.EXPECT().Send(ctx, tt.email).Return(tt.responseProviderReferenceID, nil)
			mockDB.EXPECT().EmailUpdateProviderReferenceID(ctx, tt.email.ID, tt.responseProviderReferenceID).Return(nil)

			h.Send(ctx, tt.email)
		})
	}
}
