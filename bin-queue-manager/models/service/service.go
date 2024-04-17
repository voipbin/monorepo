package service

import (
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// Service define
type Service struct {
	ID          uuid.UUID         `json:"id"`           // represent started service id "id": "e53c3df6-4f85-4714-9980-1cca63caf4f6"
	Type        Type              `json:"type"`         // represent started service type.
	PushActions []fmaction.Action `json:"push_actions"` // represent the push actions for the service requested resource.
}

// Type define
type Type string

// list of types
const (
	TypeQueuecall Type = "queuecall"
)
