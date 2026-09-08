package middleware

import (
	"monorepo/bin-api-manager/models/auth"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// DirectResourceScope confines a direct token to the single resource it was
// bound to at boot, for every route that names its target in the path.
//
// It exists because the per-handler checks only ever compared the *parent*
// resource (the AI or the widget), which is identical for every visitor of the
// same public link. Comparing the path id against the token's assignment is
// what separates one anonymous visitor from another.
//
// Routes that name their target in the body or the query are deliberately not
// handled here. Matching those would require the middleware to know which field
// is the target on each route, which is a per-route registry that goes stale.
// Those handlers overwrite the client's value with the assignment instead,
// which is stronger: there is no comparison to forget.
//
// Registration order matters. gin snapshots a group's handler chain when a
// route is registered, so this must be added to the v1 group *before*
// RegisterHandlersWithOptions, or it silently applies to nothing.
func DirectResourceScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logrus.WithFields(logrus.Fields{
			"func": "DirectResourceScope",
			"path": c.FullPath(),
		})

		// 0. No identity, or the wrong type in the slot: fail closed rather
		// than fall through, matching EnforceAccountStatus.
		v, exists := c.Get("auth_identity")
		if !exists {
			abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}
		a, ok := v.(*auth.AuthIdentity)
		if !ok || a == nil {
			abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}

		// 1. Every other identity type is out of scope. Agents, accesskeys and
		// delegates keep the permissions they already had.
		if !a.IsDirect() {
			c.Next()
			return
		}

		// reject shares its reason key with buildJWTIdentity's, so the whole
		// direct-scope rejection rate during a deploy is one aggregation rather
		// than free-text grepping across two files.
		reject := func(reason, msg string) {
			log.WithFields(logrus.Fields{
				"reject_reason": reason,
				"customer_id":   a.CustomerID,
			}).Info(msg)
			abortPermissionDenied(c, "DIRECT_SCOPE_VIOLATION", "This endpoint is not available for this token.")
		}

		// 2. No path parameters at all: a create or send route. Those are
		// covered by the handler-side overwrite.
		if len(c.Params) == 0 {
			c.Next()
			return
		}

		// 3. Parameters, but none named "id". This is the fail-closed step: it
		// means the route has a shape this middleware does not understand, so
		// refuse rather than assume it is harmless. Without it, adding a route
		// like /aicalls/:aicall_id/foo later would silently bypass the check.
		raw := c.Param("id")
		if raw == "" {
			reject("direct_route_unrecognized", "Direct request to a route with no id parameter. Rejecting.")
			return
		}

		// 4. Unparseable id. Explicit rather than relying on FromStringOrNil
		// collapsing to uuid.Nil, so this does not depend on step 5 to be safe.
		parsed, err := uuid.FromString(raw)
		if err != nil {
			reject("direct_id_unparseable", "Direct request with an unparseable id. Rejecting.")
			return
		}

		// 5. Guard the dereference, and refuse a Nil assignment outright.
		// buildJWTIdentity already rejects unbound tokens, so neither branch is
		// reachable today; both are kept so this middleware is correct on its
		// own rather than by cross-referencing another file. In particular, a
		// path id of all zeros would compare *equal* to a Nil assignment at
		// step 6.
		if a.DirectScope == nil || a.DirectScope.AllowedResourceID == uuid.Nil {
			reject("direct_unbound", "Direct identity carries no usable scope. Rejecting.")
			return
		}

		// 6. Compare parsed UUIDs, not strings: handlers parse with
		// FromStringOrNil and so accept braced and upper-case forms. A string
		// comparison would make the middleware and the handler disagree about
		// what "the same id" means.
		if parsed != a.DirectScope.AllowedResourceID {
			reject("direct_id_mismatch", "Direct request for a resource outside the token scope. Rejecting.")
			return
		}

		// 7. In scope.
		//
		// Known gap, recorded rather than fixed: a future route shaped
		// /aicalls/:id/<child>/:child_id would match on :id and leave
		// :child_id unchecked. No such route exists today. If one is added,
		// this middleware needs to learn about the second parameter.
		c.Next()
	}
}
