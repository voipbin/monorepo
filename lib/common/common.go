package common

import (
	"strings"
	"time"
)

// JSON alias type
type JSON = map[string]interface{}

// GetCurTime return current utc time string
func GetCurTime() string {
	date := time.Now().UTC()

	res := date.String()
	res = strings.TrimSuffix(res, " +0000 UTC")

	return res
}
