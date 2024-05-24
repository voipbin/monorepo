package file

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`

	URIDownload string `json:"uri_download"` // uri for download

	TMDownloadExpire string `json:"tm_download_expire"`
	TMCreate         string `json:"tm_create"`
	TMUpdate         string `json:"tm_update"`
	TMDelete         string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *File) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		OwnerID:    h.OwnerID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Name:   h.Name,
		Detail: h.Detail,

		Filename: h.Filename,
		Filesize: h.Filesize,

		URIDownload: h.URIDownload,

		TMDownloadExpire: h.TMDownloadExpire,
		TMCreate:         h.TMCreate,
		TMUpdate:         h.TMUpdate,
		TMDelete:         h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *File) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
