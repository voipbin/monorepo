package astauth

import (
	"testing"
)

func TestAstAuthStruct(t *testing.T) {
	id := "auth_123"
	authType := "userpass"
	username := "testuser"
	password := "testpass"
	realm := "example.com"
	nonceLifetime := 32
	md5Cred := "md5hash"
	oauthClientID := "client_123"
	oauthSecret := "secret_123"
	refreshToken := "refresh_123"

	auth := AstAuth{
		ID:            &id,
		AuthType:      &authType,
		Username:      &username,
		Password:      &password,
		Realm:         &realm,
		NonceLifetime: &nonceLifetime,
		MD5Cred:       &md5Cred,
		OAuthClientID: &oauthClientID,
		OAuthSecret:   &oauthSecret,
		RefreshToken:  &refreshToken,
	}

	if auth.ID == nil || *auth.ID != id {
		t.Errorf("AstAuth.ID = %v, expected %v", auth.ID, &id)
	}
	if auth.AuthType == nil || *auth.AuthType != authType {
		t.Errorf("AstAuth.AuthType = %v, expected %v", auth.AuthType, &authType)
	}
	if auth.Username == nil || *auth.Username != username {
		t.Errorf("AstAuth.Username = %v, expected %v", auth.Username, &username)
	}
	if auth.Password == nil || *auth.Password != password {
		t.Errorf("AstAuth.Password = %v, expected %v", auth.Password, &password)
	}
	if auth.Realm == nil || *auth.Realm != realm {
		t.Errorf("AstAuth.Realm = %v, expected %v", auth.Realm, &realm)
	}
	if auth.NonceLifetime == nil || *auth.NonceLifetime != nonceLifetime {
		t.Errorf("AstAuth.NonceLifetime = %v, expected %v", auth.NonceLifetime, &nonceLifetime)
	}
	if auth.MD5Cred == nil || *auth.MD5Cred != md5Cred {
		t.Errorf("AstAuth.MD5Cred = %v, expected %v", auth.MD5Cred, &md5Cred)
	}
	if auth.OAuthClientID == nil || *auth.OAuthClientID != oauthClientID {
		t.Errorf("AstAuth.OAuthClientID = %v, expected %v", auth.OAuthClientID, &oauthClientID)
	}
	if auth.OAuthSecret == nil || *auth.OAuthSecret != oauthSecret {
		t.Errorf("AstAuth.OAuthSecret = %v, expected %v", auth.OAuthSecret, &oauthSecret)
	}
	if auth.RefreshToken == nil || *auth.RefreshToken != refreshToken {
		t.Errorf("AstAuth.RefreshToken = %v, expected %v", auth.RefreshToken, &refreshToken)
	}
}

func TestAstAuthStructWithNilFields(t *testing.T) {
	auth := AstAuth{}

	if auth.ID != nil {
		t.Errorf("AstAuth.ID should be nil, got %v", auth.ID)
	}
	if auth.AuthType != nil {
		t.Errorf("AstAuth.AuthType should be nil, got %v", auth.AuthType)
	}
	if auth.Username != nil {
		t.Errorf("AstAuth.Username should be nil, got %v", auth.Username)
	}
	if auth.Password != nil {
		t.Errorf("AstAuth.Password should be nil, got %v", auth.Password)
	}
	if auth.Realm != nil {
		t.Errorf("AstAuth.Realm should be nil, got %v", auth.Realm)
	}
}
