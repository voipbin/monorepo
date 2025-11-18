package emailhandler

// func Test_engineMailgun_Send(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		email *email.Email

// 		responseProviderReferenceID string
// 		expectRes                   string
// 	}{
// 		{
// 			name: "normal",

// 			email: &email.Email{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("902d678a-c403-11f0-ad71-1b29c304715b"),
// 				},

// 				Source: &commonaddress.Address{
// 					Type:   commonaddress.TypeEmail,
// 					Target: "service@mailgun.voipbin.net",
// 				},
// 				Destinations: []commonaddress.Address{
// 					{
// 						Type:   commonaddress.TypeEmail,
// 						Target: "sungtae@voipbin.net",
// 					},
// 					{
// 						Type:   commonaddress.TypeEmail,
// 						Target: "pchero21@gmail.com",
// 					},
// 				},

// 				Subject: "Test Email from voipbin",
// 				Content: "This is a test email sent via Mailgun.",
// 			},

// 			responseProviderReferenceID: "a1d1f67e-00c5-11f0-b69e-1fa7a77d151f",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockUtil := utilhandler.NewMockUtilHandler(mc)

// 			h := &engineMailgun{
// 				reqHandler:  mockReq,
// 				utilHandler: mockUtil,
// 			}
// 			ctx := context.Background()

// 			// mg := mailgun.NewMailgun("<put your test mailgun domain>", "<put your mailgun api key here>")
// 			mg := mailgun.NewMailgun(defaultMailgunDomain, "<put your mailgun api key here>")
// 			h.client = mg

// 			res, err := h.Send(ctx, tt.email)
// 			if err != nil {
// 				t.Errorf("engineMailgun.Send() error = %v", err)
// 			}

// 			if res != tt.expectRes {
// 				t.Errorf("engineMailgun.Send() = %v, want %v", res, tt.expectRes)
// 			}
// 		})
// 	}
// }
