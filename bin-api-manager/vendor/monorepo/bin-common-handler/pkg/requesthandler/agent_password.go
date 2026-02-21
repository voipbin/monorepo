package requesthandler

import (
	"context"
	"encoding/json"

	amrequest "monorepo/bin-agent-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// AgentV1PasswordForgot sends a request to agent-manager
// to generate a password reset token and send the reset email.
// timeout: milliseconds
func (r *requestHandler) AgentV1PasswordForgot(ctx context.Context, timeout int, username string) error {
	uri := "/v1/password-forgot"

	req := &amrequest.V1DataPasswordForgotPost{
		Username: username,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPost, "agent/password-forgot", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AgentV1PasswordReset sends a request to agent-manager
// to validate a password reset token and update the password.
// timeout: milliseconds
func (r *requestHandler) AgentV1PasswordReset(ctx context.Context, timeout int, token string, password string) error {
	uri := "/v1/password-reset"

	req := &amrequest.V1DataPasswordResetPost{
		Token:    token,
		Password: password,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPost, "agent/password-reset", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
