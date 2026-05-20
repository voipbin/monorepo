package whatsapphandler_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

// validSignature computes a valid X-Hub-Signature-256 header value for the given secret and body.
func validSignature(t *testing.T, secret string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// samplePayload returns a minimal WhatsApp Cloud API webhook payload JSON with one text message.
func samplePayload(wamid, waID, text string) []byte {
	payload := map[string]any{
		"entry": []map[string]any{
			{
				"changes": []map[string]any{
					{
						"value": map[string]any{
							"messaging_product": "whatsapp",
							"metadata":          map[string]string{"phone_number_id": "111"},
							"messages": []map[string]any{
								{
									"id":   wamid,
									"from": waID,
									"type": "text",
									"text": map[string]string{"body": text},
								},
							},
							"contacts": []map[string]any{
								{
									"wa_id":   waID,
									"profile": map[string]string{"name": "Test User"},
								},
							},
						},
					},
				},
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func makeAccount(appSecret string) *account.Account {
	ac := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		},
	}
	pd, _ := json.Marshal(account.WhatsAppProviderData{
		PhoneNumberID: "111",
		AppSecret:     appSecret,
	})
	ac.ProviderData = pd
	return ac
}

func TestHook_MissingAppSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := makeAccount("") // empty app_secret — must fail

	body := samplePayload("wamid.001", "15551111111", "hi")
	sig := "sha256=invalidsig"

	_, err := h.Hook(context.Background(), ac, body, sig)
	if err == nil {
		t.Fatal("expected error when app_secret is empty (fail-closed)")
	}
}

func TestHook_WrongSignature(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := makeAccount("correct-secret")

	body := samplePayload("wamid.002", "15552222222", "hello")
	badSig := validSignature(t, "wrong-secret", body)

	_, err := h.Hook(context.Background(), ac, body, badSig)
	if err == nil {
		t.Fatal("expected error for wrong HMAC signature")
	}
}

func TestHook_DuplicateWamid(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRH := requesthandler.NewMockRequestHandler(mc)
	h := whatsapphandler.NewWhatsAppHandler(mockRH)

	const wamid = "wamid.dup"
	const waID = "15553333333"
	ac := makeAccount("secret")
	body := samplePayload(wamid, waID, "duplicate")
	sig := validSignature(t, "secret", body)

	// Dedup check returns an existing message — no further calls expected.
	existingMsg := message.Message{TransactionID: wamid}
	mockRH.EXPECT().
		ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]message.Message{existingMsg}, nil)

	results, err := h.Hook(context.Background(), ac, body, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for duplicate wamid, got %d", len(results))
	}
}

func TestHook_ValidTextMessage_ExistingConversation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRH := requesthandler.NewMockRequestHandler(mc)
	h := whatsapphandler.NewWhatsAppHandler(mockRH)

	const wamid = "wamid.new1"
	const waID = "15554444444"
	ac := makeAccount("mysecret")
	body := samplePayload(wamid, waID, "hello world")
	sig := validSignature(t, "mysecret", body)

	cvID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")
	existingCV := conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         cvID,
			CustomerID: ac.CustomerID,
		},
		DialogID: waID,
		Type:     conversation.TypeWhatsApp,
	}
	createdMsg := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		},
		TransactionID: wamid,
	}

	// No duplicate
	mockRH.EXPECT().
		ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]message.Message{}, nil)

	// Existing conversation found
	mockRH.EXPECT().
		ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]conversation.Conversation{existingCV}, nil)

	// Message created
	mockRH.EXPECT().
		ConversationV1MessageCreate(
			gomock.Any(),
			uuid.Nil,
			existingCV.CustomerID,
			cvID,
			message.DirectionIncoming,
			message.StatusDone,
			message.ReferenceTypeWhatsApp,
			uuid.Nil,
			wamid,
			"hello world",
			gomock.Any(),
		).
		Return(createdMsg, nil)

	results, err := h.Hook(context.Background(), ac, body, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Message.TransactionID != wamid {
		t.Errorf("expected wamid %s, got %s", wamid, results[0].Message.TransactionID)
	}
}

func TestHook_NonTextMessageSkipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := whatsapphandler.NewWhatsAppHandler(mockReq)

	appSecret := "skip-secret"
	ac := makeAccount(appSecret)

	rawData := []byte(`{
		"entry": [{
			"changes": [{
				"value": {
					"metadata": {"phone_number_id": "12345", "display_phone_number": "+10001112222"},
					"contacts": [{"profile": {"name": "Dave"}}],
					"messages": [{
						"from": "+15557654321",
						"id": "wamid.img1",
						"type": "image",
						"image": {"id": "img-handle-123"}
					}]
				}
			}]
		}]
	}`)

	// Compute valid HMAC signature
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawData)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// No mock expectations — non-text messages are skipped entirely

	results, err := h.Hook(context.Background(), ac, rawData, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-text message, got %d", len(results))
	}
}

func TestHook_CreateOnFirstMessage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRH := requesthandler.NewMockRequestHandler(mc)
	h := whatsapphandler.NewWhatsAppHandler(mockRH)

	const wamid = "wamid.first"
	const waID = "15555555555"
	ac := makeAccount("appsecret")
	body := samplePayload(wamid, waID, "first message")
	sig := validSignature(t, "appsecret", body)

	newCvID := uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff")
	createdCV := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         newCvID,
			CustomerID: ac.CustomerID,
		},
		DialogID: waID,
		Type:     conversation.TypeWhatsApp,
	}
	updatedCV := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         newCvID,
			CustomerID: ac.CustomerID,
		},
		AccountID: ac.ID,
		DialogID:  waID,
		Type:      conversation.TypeWhatsApp,
	}
	createdMsg := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		TransactionID: wamid,
	}

	// No duplicate
	mockRH.EXPECT().
		ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]message.Message{}, nil)

	// No existing conversation
	mockRH.EXPECT().
		ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]conversation.Conversation{}, nil)

	// Create new conversation
	mockRH.EXPECT().
		ConversationV1ConversationCreate(
			gomock.Any(),
			ac.CustomerID,
			gomock.Any(), // name
			gomock.Any(), // detail
			conversation.TypeWhatsApp,
			waID,
			gomock.Any(), // self
			gomock.Any(), // peer
		).
		Return(createdCV, nil)

	// Update with account_id
	mockRH.EXPECT().
		ConversationV1ConversationUpdate(gomock.Any(), newCvID, gomock.Any()).
		Return(updatedCV, nil)

	// Create message
	mockRH.EXPECT().
		ConversationV1MessageCreate(
			gomock.Any(),
			uuid.Nil,
			updatedCV.CustomerID,
			newCvID,
			message.DirectionIncoming,
			message.StatusDone,
			message.ReferenceTypeWhatsApp,
			uuid.Nil,
			wamid,
			"first message",
			gomock.Any(),
		).
		Return(createdMsg, nil)

	results, err := h.Hook(context.Background(), ac, body, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Conversation.AccountID != ac.ID {
		t.Errorf("expected AccountID %s, got %s", ac.ID, results[0].Conversation.AccountID)
	}
	if results[0].Message.TransactionID != wamid {
		t.Errorf("expected wamid %s, got %s", wamid, results[0].Message.TransactionID)
	}
}
