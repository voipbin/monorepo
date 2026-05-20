package whatsapphandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/internal/convtitle"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// whatsappPayload is the top-level structure of a WhatsApp Cloud API webhook payload.
type whatsappPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					PhoneNumberID     string `json:"phone_number_id"`
					DisplayPhoneNumber string `json:"display_phone_number"`
				} `json:"metadata"`
				Messages []struct {
					ID   string `json:"id"`   // wamid
					From string `json:"from"` // sender's phone number (wa_id)
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// Hook validates the HMAC-SHA256 signature, parses the WhatsApp Cloud API webhook payload,
// and creates a conversation+message for each inbound text message.
// It is fail-closed: if app_secret is empty the request is rejected.
func (h *whatsappHandler) Hook(ctx context.Context, ac *account.Account, rawData []byte, signature string) ([]*HookResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "whatsapphandler.Hook",
		"account_id": ac.ID,
	})

	// --- Decode provider_data ---
	var pd account.WhatsAppProviderData
	if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
		return nil, fmt.Errorf("whatsapphandler.Hook: invalid provider_data: %w", err)
	}

	// --- Fail-closed HMAC verification ---
	if pd.AppSecret == "" {
		return nil, fmt.Errorf("whatsapphandler.Hook: app_secret is required for signature verification")
	}
	if err := verifySignature(pd.AppSecret, rawData, signature); err != nil {
		return nil, errors.Wrap(err, "whatsapphandler.Hook: signature verification failed")
	}

	// --- Parse payload ---
	var payload whatsappPayload
	if err := json.Unmarshal(rawData, &payload); err != nil {
		return nil, fmt.Errorf("whatsapphandler.Hook: unmarshal payload: %w", err)
	}

	// --- Build sender name lookup from contacts ---
	senderNames := map[string]string{}
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, contact := range change.Value.Contacts {
				if contact.WaID != "" {
					senderNames[contact.WaID] = contact.Profile.Name
				}
			}
		}
	}

	results := []*HookResult{}
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" {
					log.Debugf("Skipping non-text WhatsApp message. type: %s, wamid: %s", msg.Type, msg.ID)
					continue
				}

				selfTarget := change.Value.Metadata.DisplayPhoneNumber
				r, err := h.hookTextMessage(ctx, ac, msg.ID, msg.From, selfTarget, msg.Text.Body, senderNames[msg.From])
				if err != nil {
					log.Errorf("Could not handle WhatsApp text message. wamid: %s, err: %v", msg.ID, err)
					continue
				}
				if r != nil {
					results = append(results, r)
				}
			}
		}
	}

	return results, nil
}

// hookTextMessage processes a single inbound WhatsApp text message.
func (h *whatsappHandler) hookTextMessage(
	ctx context.Context,
	ac *account.Account,
	wamid string,
	waID string,
	selfTarget string,
	text string,
	senderName string,
) (*HookResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "whatsapphandler.hookTextMessage",
		"account_id": ac.ID,
		"wamid":      wamid,
		"wa_id":      waID,
	})

	// --- Dedup: skip if wamid already recorded ---
	existing, err := h.reqHandler.ConversationV1MessageList(ctx, "", 1, map[message.Field]any{
		message.FieldTransactionID: wamid,
	})
	if err != nil {
		return nil, errors.Wrap(err, "hookTextMessage: check duplicate wamid")
	}
	if len(existing) > 0 {
		log.Debugf("Duplicate wamid — skipping. wamid: %s", wamid)
		return nil, nil
	}

	// --- Get or create conversation ---
	cv, err := h.getOrCreateConversation(ctx, ac, waID, selfTarget, senderName)
	if err != nil {
		return nil, errors.Wrap(err, "hookTextMessage: get or create conversation")
	}

	// --- Create message ---
	m, err := h.reqHandler.ConversationV1MessageCreate(
		ctx,
		uuid.Nil,
		cv.CustomerID,
		cv.ID,
		message.DirectionIncoming,
		message.StatusDone,
		message.ReferenceTypeWhatsApp,
		uuid.Nil,
		wamid,
		text,
		[]media.Media{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "hookTextMessage: create message")
	}
	log.Debugf("Created WhatsApp message. message_id: %s", m.ID)

	return &HookResult{
		Conversation: cv,
		Message:      m,
	}, nil
}

// getOrCreateConversation returns the existing conversation for the given waID,
// or creates a new one and immediately sets its AccountID.
func (h *whatsappHandler) getOrCreateConversation(
	ctx context.Context,
	ac *account.Account,
	waID string,
	selfTarget string,
	senderName string,
) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "whatsapphandler.getOrCreateConversation",
		"account_id": ac.ID,
		"wa_id":      waID,
	})

	cvs, err := h.reqHandler.ConversationV1ConversationList(ctx, "", 1, map[conversation.Field]any{
		conversation.FieldType:     conversation.TypeWhatsApp,
		conversation.FieldDialogID: waID,
		conversation.FieldDeleted:  false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getOrCreateConversation: list conversations")
	}
	if len(cvs) > 0 {
		log.Debugf("Found existing conversation. conversation_id: %s", cvs[0].ID)
		cv := cvs[0]
		return &cv, nil
	}

	// Build peer address
	peer := commonaddress.Address{
		Type:       commonaddress.TypeWhatsApp,
		Target:     waID,
		TargetName: senderName,
	}
	self := commonaddress.Address{
		Type:       commonaddress.TypeWhatsApp,
		Target:     selfTarget,
		TargetName: "Me",
	}

	name, detail := convtitle.Build(conversation.TypeWhatsApp, peer)

	cv, err := h.reqHandler.ConversationV1ConversationCreate(
		ctx,
		ac.CustomerID,
		name,
		detail,
		conversation.TypeWhatsApp,
		waID,
		self,
		peer,
	)
	if err != nil {
		return nil, errors.Wrap(err, "getOrCreateConversation: create conversation")
	}
	log.Debugf("Created new conversation. conversation_id: %s", cv.ID)

	// Persist account_id — ConversationCreate does not accept it directly.
	updated, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, cv.ID, map[conversation.Field]any{
		conversation.FieldAccountID: ac.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getOrCreateConversation: update conversation account_id")
	}

	return updated, nil
}

// verifySignature validates the X-Hub-Signature-256 header produced by Meta.
// Expected format: "sha256=<lowercase-hex>".
func verifySignature(appSecret string, body []byte, signature string) error {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return fmt.Errorf("signature missing sha256= prefix")
	}
	sigHex := strings.TrimPrefix(signature, prefix)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expected)) {
		return fmt.Errorf("HMAC mismatch")
	}
	return nil
}
