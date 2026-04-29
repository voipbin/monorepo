package messagehandler

import (
	"testing"

	"monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
)

func TestCreateOptions_apply(t *testing.T) {
	pcID := uuid.Must(uuid.NewV4())
	var p createParams
	WithPipecatcallID(pcID)(&p)
	WithDeliveryStatus(message.DeliveryStatusPending)(&p)
	if p.pipecatcallID != pcID || p.deliveryStatus != message.DeliveryStatusPending {
		t.Fatalf("options not applied: %+v", p)
	}
}
