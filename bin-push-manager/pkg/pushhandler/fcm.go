package pushhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"monorepo/bin-push-manager/models/notification"
)

const (
	fcmScope    = "https://www.googleapis.com/auth/firebase.messaging"
	fcmEndpoint = "https://fcm.googleapis.com"

	// fcmErrorCodeUnregistered is the FCM error code of a token that is not valid any more.
	fcmErrorCodeUnregistered = "UNREGISTERED"

	// tokenRefreshTimeout is the longest wait for the OAuth token endpoint.
	tokenRefreshTimeout = 6 * time.Second

	// maxErrorBody is the most bytes read of an error response.
	maxErrorBody = 64 * 1024
)

// fcmSender sends the notifications through the FCM HTTP v1 API.
type fcmSender struct {
	client    *http.Client
	endpoint  string
	projectID string
}

// newFCMSenderFromJSON builds the sender from the content of a service account JSON. The project id is the
// project_id of the JSON. The returned error never contains the content.
func newFCMSenderFromJSON(content []byte) (Sender, error) {
	// the OAuth token refresh does not take the context of a send, so its http client has its own timeout
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: tokenRefreshTimeout})

	creds, err := google.CredentialsFromJSONWithType(ctx, content, google.ServiceAccount, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("could not parse the service account")
	}

	if creds.ProjectID == "" {
		return nil, fmt.Errorf("the service account has no project id")
	}

	return newFCMSender(oauth2.NewClient(ctx, creds.TokenSource), fcmEndpoint, creds.ProjectID), nil
}

// newFCMSender builds the sender with the given http client and endpoint (the test constructor).
func newFCMSender(client *http.Client, endpoint, projectID string) *fcmSender {
	return &fcmSender{
		client:    client,
		endpoint:  endpoint,
		projectID: projectID,
	}
}

type fcmAndroid struct {
	Priority string `json:"priority"`
	TTL      string `json:"ttl"`
}

type fcmMessage struct {
	Token   string            `json:"token"`
	Data    map[string]string `json:"data"`
	Android fcmAndroid        `json:"android"`
}

type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmErrorResponse struct {
	Error struct {
		Details []struct {
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

// buildBody builds the FCM request body: a data only message with the kind added. The priority and the TTL
// follow the kind.
func buildBody(token string, kind notification.Kind, data map[string]string) ([]byte, error) {
	d := make(map[string]string, len(data)+1)
	for k, v := range data {
		d[k] = v
	}
	d[notification.DataKeyKind] = string(kind)

	policy := notification.PolicyOf(kind)

	return json.Marshal(&fcmRequest{
		Message: fcmMessage{
			Token:   token,
			Data:    d,
			Android: fcmAndroid{Priority: policy.Priority, TTL: policy.TTL},
		},
	})
}

// Send delivers the notification to the token.
func (s *fcmSender) Send(ctx context.Context, token string, kind notification.Kind, data map[string]string) error {
	body, err := buildBody(token, kind, data)
	if err != nil {
		return fmt.Errorf("could not build the request body")
	}

	url := fmt.Sprintf("%s/v1/projects/%s/messages:send", s.endpoint, s.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not build the request")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		// the url of the error has the project id only: the token is in the body, never in the url
		return fmt.Errorf("could not send the request. err: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxErrorBody))
		return nil
	}

	raw, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBody))
	var errRes fcmErrorResponse
	if errParse := json.Unmarshal(raw, &errRes); errParse == nil {
		for _, d := range errRes.Error.Details {
			if d.ErrorCode == fcmErrorCodeUnregistered {
				return ErrUnregistered
			}
		}
	}

	return fmt.Errorf("fcm send failed. status: %d", res.StatusCode)
}
