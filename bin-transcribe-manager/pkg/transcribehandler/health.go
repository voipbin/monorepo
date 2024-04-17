package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
)

// HealthCheck checks the given transcribe is still vaild
// and stop the transcribe if the transcribe is not valid and over the default retry count.
func (h *transcribeHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "HealthCheck",
		"transcribe_id": id,
		"retry_count":   retryCount,
	})

	if retryCount > defaultHealthMaxRetryCount {
		log.Errorf("The health check exceeded max retry count. Stopping the transcribe. retry_count: %d", retryCount)
		_, _ = h.Stop(ctx, id)
		return
	}

	// validate the transcribe.
	tr, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return
	}
	if tr.Status == transcribe.StatusDone || tr.TMDelete < dbhandler.DefaultTimeStamp {
		// the call is done already. no need to check the health anymore.
		return
	}

	// validate reference
	switch tr.ReferenceType {
	case transcribe.ReferenceTypeCall:
		c, err := h.reqHandler.CallV1CallGet(ctx, tr.ReferenceID)
		if err != nil {
			log.Errorf("Could not get reference call info. Stopping the transcribe. err: %v", err)
			_, _ = h.Stop(ctx, id)
			return
		}

		if c.Status == call.StatusHangup || c.TMDelete < dbhandler.DefaultTimeStamp || c.TMHangup < dbhandler.DefaultTimeStamp {
			// the call is done already. no need to check the health anymore.
			retryCount++
		} else {
			retryCount = 0
		}

	case transcribe.ReferenceTypeConfbridge:
		cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, tr.ReferenceID)
		if err != nil {
			log.Errorf("Could not get reference confbridge info. Stopping the transcribe. err: %v", err)
			_, _ = h.Stop(ctx, id)
			return
		}

		if cb.TMDelete < dbhandler.DefaultTimeStamp {
			retryCount++
		} else {
			retryCount = 0
		}
	}

	go func() {
		_ = h.reqHandler.TranscribeV1TranscribeHealthCheck(ctx, id, defaultHealthDelay, retryCount)
	}()
}
