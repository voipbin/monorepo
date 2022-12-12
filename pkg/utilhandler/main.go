package utilhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package utilhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"github.com/gofrs/uuid"
)

// UtilHandler defines
type UtilHandler interface {
	CreateUUID() uuid.UUID

	GetCurTime() string
	GetCurTimeRFC3339() string
}

type utilHandler struct{}

// NewUtilHandler defines
func NewUtilHandler() UtilHandler {
	return &utilHandler{}
}
