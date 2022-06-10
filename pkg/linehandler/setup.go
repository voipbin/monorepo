package linehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Setup sets up the given customer's line webhook
func (h *lineHandler) Setup(ctx context.Context, customerID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Setup",
		},
	)

	c, err := h.getClient(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	// create webhook uri
	// "https://hook.voipbin.net/v1.0/conversation/customers/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c/line",
	uri := fmt.Sprintf("https://hook.voipbin.net/v1.0/conversation/customers/%s/line", customerID)
	_, err = c.SetWebhookEndpointURL(uri).WithContext(ctx).Do()
	if err != nil {
		log.Errorf("Could not set webhook uri. err: %v", err)
		return err
	}

	return nil
}
