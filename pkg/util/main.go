package util

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package util -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// Util defines
type Util interface {
	CreateUUID() uuid.UUID

	GetCurTime() string
	GetCurTimeRFC3339() string
}

type util struct{}

// NewUtil defines
func NewUtil() Util {
	return &util{}
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// GetCurTimeRFC3339 return current utc time string in a RFC3339 format
func GetCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// CreateUUID returns a new uuid v4
func CreateUUID() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}

// GetCurTime return current utc time string
func (h *util) GetCurTime() string {
	return GetCurTime()
}

// GetCurTimeRFC3339 return current utc time string in a RFC3339 format
func (h *util) GetCurTimeRFC3339() string {
	return GetCurTimeRFC3339()
}

// CreateUUID returns a new uuid v4
func (h *util) CreateUUID() uuid.UUID {
	return CreateUUID()
}
