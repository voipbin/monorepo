package utilhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package utilhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"net/url"
	"time"

	"github.com/gofrs/uuid"
)

// UtilHandler defines
type UtilHandler interface {
	UUIDCreate() uuid.UUID

	TimeGetCurTime() string
	TimeGetCurTimeAdd(duration time.Duration) string
	TimeGetCurTimeRFC3339() string
	TimeParse(timeString string) time.Time

	URLParseFilters(u *url.URL) map[string]string
}

type utilHandler struct{}

// NewUtilHandler defines
func NewUtilHandler() UtilHandler {
	return &utilHandler{}
}
