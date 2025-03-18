package engine_dialogflow_handler

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"

	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	dialogflowpb "cloud.google.com/go/dialogflow/apiv2/dialogflowpb"
)

func (h *engineDialogflowHandler) send(ctx context.Context, client *dialogflow.SessionsClient, req *dialogflowpb.DetectIntentRequest) (*dialogflowpb.DetectIntentResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second
	expBackoff.MaxElapsedTime = 1 * time.Minute

	var resp *dialogflowpb.DetectIntentResponse
	var err error
	operation := func() error {
		var err error
		resp, err = client.DetectIntent(ctx, req)
		if err != nil {
			return err
		}
		return nil
	}

	if errRetry := backoff.Retry(operation, expBackoff); errRetry != nil {
		return nil, err
	}

	return resp, nil
}
