package interaction

// InteractionListResponse is the response envelope for all interaction list
// endpoints (GET /v1/interactions, GET /v1/interactions/unresolved).
//
// This type lives in models/interaction (not pkg/contacthandler) to avoid a
// circular import: pkg/contacthandler imports bin-common-handler/pkg/requesthandler,
// and bin-common-handler/pkg/requesthandler must import this type to deserialise
// responses. Placing it here keeps the import boundary at the models layer.
//
// NextPageToken is empty when no further pages exist. Callers forward it
// verbatim as page_token on the next request; no client-side decoding is needed.
type InteractionListResponse struct {
	Items         []*Interaction `json:"items"`
	NextPageToken string         `json:"next_page_token"`
}
