package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/models/auth"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

var (
	dsAllowedID = uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-00000000beef")
	dsOtherID   = uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-00000000dead")
)

func dsDirect(allowed uuid.UUID) *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "ai",
		AllowedResourceTypes: []string{"aicall"},
		AllowedResourceID:    allowed,
	})
}

// dsRun wires the middleware behind a route and reports whether the handler was
// reached and what status came back.
func dsRun(t *testing.T, route, path string, identity any, setIdentity bool) (reached bool, status int) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setIdentity {
			c.Set("auth_identity", identity)
		}
		c.Next()
	})
	r.Use(DirectResourceScope())
	r.GET(route, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return reached, w.Code
}

func Test_DirectResourceScope(t *testing.T) {
	tests := []struct {
		name         string
		route        string
		path         string
		identity     any
		setIdentity  bool
		expectReach  bool
		expectStatus int
	}{
		{
			name: "no identity in context is unauthenticated, not a pass-through",
			// Fail closed: a chain misconfiguration must not become an open door.
			route: "/aicalls/:id", path: "/aicalls/" + dsAllowedID.String(),
			setIdentity: false, expectReach: false, expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "wrong type in the identity slot is unauthenticated",
			route: "/aicalls/:id", path: "/aicalls/" + dsAllowedID.String(),
			identity: "not an identity", setIdentity: true,
			expectReach: false, expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "agent identity is untouched",
			route: "/aicalls/:id", path: "/aicalls/" + dsOtherID.String(),
			identity: &auth.AuthIdentity{Type: auth.TypeAgent}, setIdentity: true,
			expectReach: true, expectStatus: http.StatusOK,
		},
		{
			name:  "no path parameters passes through to the handler overwrite",
			route: "/aicalls", path: "/aicalls",
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: true, expectStatus: http.StatusOK,
		},
		{
			// The fail-closed step. A route shaped like this today does not
			// exist; if one is added, it must not silently bypass the check.
			name:  "parameters but none named id is refused",
			route: "/aicalls/:aicall_id/foo", path: "/aicalls/" + dsAllowedID.String() + "/foo",
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
		{
			name:  "unparseable id is refused",
			route: "/aicalls/:id", path: "/aicalls/not-a-uuid",
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
		{
			name:  "matching id is allowed",
			route: "/aicalls/:id", path: "/aicalls/" + dsAllowedID.String(),
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: true, expectStatus: http.StatusOK,
		},
		{
			// The whole point: another visitor of the same public link.
			name:  "another visitor's id is refused",
			route: "/aicalls/:id", path: "/aicalls/" + dsOtherID.String(),
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
		{
			// Handlers parse with FromStringOrNil, which accepts these forms.
			// A string comparison here would disagree with the handler.
			name:  "upper case id still matches",
			route: "/aicalls/:id", path: "/aicalls/AAAAAAAA-0000-0000-0000-00000000BEEF",
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: true, expectStatus: http.StatusOK,
		},
		{
			name:  "braced id still matches",
			route: "/aicalls/:id", path: "/aicalls/{" + dsAllowedID.String() + "}",
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: true, expectStatus: http.StatusOK,
		},
		{
			// Without the explicit Nil guard an all-zeros path id would compare
			// equal to an unbound token and be let through.
			name:  "unbound token with an all-zeros path id is refused",
			route: "/aicalls/:id", path: "/aicalls/00000000-0000-0000-0000-000000000000",
			identity: dsDirect(uuid.Nil), setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
		{
			name:  "direct identity with a nil scope is refused",
			route: "/aicalls/:id", path: "/aicalls/" + dsAllowedID.String(),
			identity: &auth.AuthIdentity{Type: auth.TypeDirect}, setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
		{
			// A message id can never equal a conversation id, so routes of a
			// different resource kind fall out for free.
			name:  "a different resource kind's id is refused",
			route: "/aimessages/:id", path: "/aimessages/" + dsOtherID.String(),
			identity: dsDirect(dsAllowedID), setIdentity: true,
			expectReach: false, expectStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached, status := dsRun(t, tt.route, tt.path, tt.identity, tt.setIdentity)
			if reached != tt.expectReach {
				t.Errorf("Wrong handler reach. expect: %v, got: %v", tt.expectReach, reached)
			}
			if status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, status)
			}
		})
	}
}
