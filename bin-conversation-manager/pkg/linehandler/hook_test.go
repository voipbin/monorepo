package linehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
		data    []byte

		// responseAccount *account.Account

		expectResConversations []*conversation.Conversation
		expectResMessages      []*message.Message
	}{
		{
			name: "received message",

			account: &account.Account{
				ID:         uuid.FromStringOrNil("5c1e2020-ff15-11ed-9f7c-5fdc6685e3e2"),
				CustomerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
			},
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			expectResMessages: []*message.Message{
				{
					CustomerID:    uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
					Status:        message.StatusReceived,
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:   "Hello",
					Medias: []media.Media{},
				},
			},
		},
		{
			name: "received follow",

			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "follow",
						"webhookEventId": "01G49KGV3YYCWA0CPZHP9AA6H9",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884873348,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "44b7e0b5fa034a58bfd75c9e256ad2ed",
						"mode": "active"
					}
				]
			}`),

			account: &account.Account{
				ID:         uuid.FromStringOrNil("c62c6260-ff15-11ed-9a69-bb517f07e453"),
				CustomerID: uuid.FromStringOrNil("c65bdc84-ff15-11ed-b5e5-eb878c571347"),
				Type:       account.TypeLine,
				Secret:     "ba5f0575d826d5b4a052a43145ef1391",
				Token:      "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},

			expectResConversations: []*conversation.Conversation{
				{
					CustomerID: uuid.FromStringOrNil("c65bdc84-ff15-11ed-b5e5-eb878c571347"),
					AccountID:  uuid.FromStringOrNil("c62c6260-ff15-11ed-9a69-bb517f07e453"),

					Name:   "Sungtae Kim",
					Detail: "Conversation with Sungtae Kim",

					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

					Source: &commonaddress.Address{
						Type:       commonaddress.TypeLine,
						Target:     "",
						TargetName: "Me",
					},
					Participants: []commonaddress.Address{
						{
							Type:       commonaddress.TypeLine,
							Target:     "",
							TargetName: "Me",
						},
						{
							Type:       commonaddress.TypeLine,
							Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
							TargetName: "Sungtae Kim",
						},
					},
				},
			},
		},
		{
			name: "received follow and message",

			account: &account.Account{
				ID:         uuid.FromStringOrNil("110d5ed8-ff16-11ed-aefb-0b546f35699e"),
				CustomerID: uuid.FromStringOrNil("11391898-ff16-11ed-a3ca-ef45e5a17de8"),
				Type:       account.TypeLine,
				Secret:     "ba5f0575d826d5b4a052a43145ef1391",
				Token:      "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "follow",
						"webhookEventId": "01G49KGV3YYCWA0CPZHP9AA6H9",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884873348,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "44b7e0b5fa034a58bfd75c9e256ad2ed",
						"mode": "active"
					},
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			expectResConversations: []*conversation.Conversation{
				{
					CustomerID: uuid.FromStringOrNil("11391898-ff16-11ed-a3ca-ef45e5a17de8"),
					AccountID:  uuid.FromStringOrNil("110d5ed8-ff16-11ed-aefb-0b546f35699e"),

					Name:   "Sungtae Kim",
					Detail: "Conversation with Sungtae Kim",

					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

					Source: &commonaddress.Address{
						Type:       commonaddress.TypeLine,
						Target:     "",
						TargetName: "Me",
					},
					Participants: []commonaddress.Address{
						{
							Type:       commonaddress.TypeLine,
							Target:     "",
							TargetName: "Me",
						},
						{
							Type:       commonaddress.TypeLine,
							Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
							TargetName: "Sungtae Kim",
						},
					},
				},
			},
			expectResMessages: []*message.Message{
				{
					ID:             [16]byte{},
					CustomerID:     uuid.FromStringOrNil("11391898-ff16-11ed-a3ca-ef45e5a17de8"),
					ConversationID: [16]byte{},
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:   "Hello",
					Medias: []media.Media{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			h := lineHandler{}
			ctx := context.Background()

			resConversations, resMessages, err := h.Hook(ctx, tt.account, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(tt.expectResConversations) > 0 && !reflect.DeepEqual(resConversations, tt.expectResConversations) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectResConversations[0], resConversations[0])
			}
			if len(tt.expectResMessages) > 0 && !reflect.DeepEqual(resMessages, tt.expectResMessages) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectResMessages[0], resMessages[0])
			}
		})
	}
}

func Test_getParticipant(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
		id      string

		expectRes *commonaddress.Address
	}{
		{
			name: "normal",

			account: &account.Account{
				ID:         uuid.FromStringOrNil("59e02406-ff16-11ed-ae5e-87a8da5c72e4"),
				CustomerID: uuid.FromStringOrNil("5a1ccef6-ff16-11ed-9538-eb049dc1f23e"),
				Secret:     "ba5f0575d826d5b4a052a43145ef1391",
				Token:      "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},
			id: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TargetName: "Sungtae Kim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			h := lineHandler{}

			ctx := context.Background()

			res, err := h.GetParticipant(ctx, tt.account, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectRes, res)
			}
		})
	}
}
