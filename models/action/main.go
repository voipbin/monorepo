package action

import (
	"encoding/json"

	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Action struct for client show
type Action struct {
	Type   string          `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
}

// ConvertAction return converted action.Action
func ConvertAction(r *fmaction.Action) *Action {
	return &Action{
		Type:   string(r.Type),
		Option: r.Option,
	}
}

// CreateAction returns created fmaction from the action.Action.
func CreateAction(a *Action) *fmaction.Action {
	return &fmaction.Action{
		Type:   fmaction.Type(a.Type),
		Option: a.Option,
	}
}
