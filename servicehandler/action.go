package servicehandler

import "gitlab.com/voipbin/bin-manager/api-manager/models/action"

func (h *serviceHandler) ValidateAction(a *action.Action) bool {

	return true
}
