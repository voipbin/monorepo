package common

import (
	"time"
)

const (
	TokenExpiration = time.Hour * 24 * 7 // 1 week(7 days)
)

// JSON alias type
type JSON = map[string]interface{}
