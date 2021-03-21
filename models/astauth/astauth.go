package astauth

// AstAuth struct for Asterisk's auth
type AstAuth struct {
	ID *string

	AuthType *string

	Username *string
	Password *string
	Realm    *string

	NonceLifetime *int
	MD5Cred       *string

	OAuthClientID *string
	OAuthSecret   *string

	RefreshToken *string
}
