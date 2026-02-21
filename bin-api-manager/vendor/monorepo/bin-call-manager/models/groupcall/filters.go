package groupcall

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Groupcall queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID        uuid.UUID    `filter:"customer_id"`
	OwnerID           uuid.UUID    `filter:"owner_id"`
	Status            Status       `filter:"status"`
	FlowID            uuid.UUID    `filter:"flow_id"`
	MasterCallID      uuid.UUID    `filter:"master_call_id"`
	MasterGroupcallID uuid.UUID    `filter:"master_groupcall_id"`
	RingMethod        RingMethod   `filter:"ring_method"`
	AnswerMethod      AnswerMethod `filter:"answer_method"`
	AnswerCallID      uuid.UUID    `filter:"answer_call_id"`
	AnswerGroupcallID uuid.UUID    `filter:"answer_groupcall_id"`
	Deleted           bool         `filter:"deleted"`
}
