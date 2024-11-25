package utilhandler

//go:generate mockgen -package utilhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"net/url"
	"time"

	"github.com/gofrs/uuid"
)

// UtilHandler defines
type UtilHandler interface {
	// uuid helpers
	UUIDCreate() uuid.UUID

	// time helpers
	TimeGetCurTime() string
	TimeGetCurTimeAdd(duration time.Duration) string
	TimeGetCurTimeRFC3339() string
	TimeParse(timeString string) time.Time

	// url helpers
	URLParseFilters(u *url.URL) map[string]string
	URLMergeFilters(uri string, filters map[string]string) string
}

type utilHandler struct{}

// NewUtilHandler defines
func NewUtilHandler() UtilHandler {
	return &utilHandler{}
}
