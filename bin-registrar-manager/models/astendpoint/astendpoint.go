package astendpoint

// AstEndpoint for ast_endpoint table
type AstEndpoint struct {
	ID         *string // id
	Transport  *string // transport
	AORs       *string // aors
	Auth       *string // auth
	Context    *string // context
	IdentifyBy *string //
	FromDomain *string //
}
