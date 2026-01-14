package aicall

import (
	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/ai"
)

// FieldStruct defines allowed filters for AIcall queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID    uuid.UUID      `filter:"customer_id"`
	AIID          uuid.UUID      `filter:"ai_id"`
	AIEngineType  ai.EngineType  `filter:"ai_engine_type"`
	AIEngineModel ai.EngineModel `filter:"ai_engine_model"`
	ActiveflowID  uuid.UUID      `filter:"activeflow_id"`
	ReferenceType ReferenceType  `filter:"reference_type"`
	ReferenceID   uuid.UUID      `filter:"reference_id"`
	ConfbridgeID  uuid.UUID      `filter:"confbridge_id"`
	PipecatcallID uuid.UUID      `filter:"pipecatcall_id"`
	Status        Status         `filter:"status"`
	Gender        Gender         `filter:"gender"`
	Language      string         `filter:"language"`
	Deleted       bool           `filter:"deleted"`
}
