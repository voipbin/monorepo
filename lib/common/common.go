package common

import (
	"strings"
	"time"
)

// JSON alias type
type JSON = map[string]interface{}

// GetCurTime return current utc time string
func GetCurTime() string {
	date := time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC)

	res := date.String()
	res = strings.TrimSuffix(res, " +0000 UTC")

	return res
}
