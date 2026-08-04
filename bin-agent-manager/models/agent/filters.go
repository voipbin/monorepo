package agent

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Agent queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID         uuid.UUID  `filter:"id"`
	CustomerID uuid.UUID  `filter:"customer_id"`
	Username   string     `filter:"username"`
	Name       string     `filter:"name"`
	RingMethod RingMethod `filter:"ring_method"`
	Status     Status     `filter:"status"`
	Deleted    bool       `filter:"deleted"`
	// TagIDs is a comma-joined, normalized (lowercase, uuid.String()) list of
	// tag ids. It cannot be a []uuid.UUID here because ConvertFilters passes
	// values through untyped; the JSON-array containment query itself is
	// built by hand in dbhandler.AgentList (agents.tag_ids is a JSON column,
	// not a plain equality-comparable one). See FormatTagIDsFilter/ParseTagIDsFilter.
	TagIDs string `filter:"tag_ids"`
}
