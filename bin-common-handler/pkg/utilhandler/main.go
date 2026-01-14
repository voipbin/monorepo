package utilhandler

//go:generate mockgen -package utilhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"net/url"
	"time"

	"github.com/gofrs/uuid"
)

// UtilHandler defines
type UtilHandler interface {
	// email
	EmailIsValid(e string) bool

	// hash
	HashCheckPassword(password, hashString string) bool
	HashGenerate(org string, cost int) (string, error)

	// uuid helpers
	UUIDCreate() uuid.UUID

	// string helpers
	StringGenerateRandom(size int) (string, error)

	// time helpers
	TimeGetCurTime() string
	TimeGetCurTimeAdd(duration time.Duration) string
	TimeGetCurTimeRFC3339() string
	TimeParse(timeString string) time.Time

	// url helpers
	URLParseFilters(u *url.URL) map[string]string
	URLMergeFilters(uri string, filters map[string]string) string

	// filter helpers
	ParseFiltersFromRequestBody(data []byte) (map[string]any, error)
}

type utilHandler struct{}

// NewUtilHandler defines
func NewUtilHandler() UtilHandler {
	return &utilHandler{}
}

// ParseFiltersFromRequestBody implements UtilHandler
func (h *utilHandler) ParseFiltersFromRequestBody(data []byte) (map[string]any, error) {
	return ParseFiltersFromRequestBody(data)
}
