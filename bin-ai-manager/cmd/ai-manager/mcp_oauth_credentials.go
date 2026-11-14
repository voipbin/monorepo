package main

import (
	"context"

	"monorepo/bin-common-handler/pkg/confighandler"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
)

// mcpOAuthCredentials adapts the config handler to the function the MCP OAuth handler calls each
// time a request needs the OAuth app of a vendor. It adds nothing: the pair comes from the database
// only, and an error is passed on as it is, so a failed read is never answered with an empty value.
func mcpOAuthCredentials(h confighandler.Handler) mcpoauthhandler.CredentialsFunc {
	return func(ctx context.Context, vendor string) (mcpoauthhandler.Credentials, error) {
		id, secret, err := aiconfig.MCPOAuthClient(ctx, h, vendor)
		if err != nil {
			return mcpoauthhandler.Credentials{}, err
		}

		return mcpoauthhandler.Credentials{ClientID: id, ClientSecret: secret}, nil
	}
}
