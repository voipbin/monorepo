package request

import (
	hmhook "gitlab.com/voipbin/bin-manager/hook-manager.git/models/hook"
)

// V1DataHooksPost is
// v1 data type request struct for
// /v1/hooks POST
type V1DataHooksPost struct {
	hmhook.Hook
}
