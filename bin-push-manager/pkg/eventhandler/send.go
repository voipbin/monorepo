package eventhandler

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

// sendAll sends the notification to every agent, at most h.parallelism sends at a time, and returns the number
// of agents that were not tried because the context ended first. The customer of the event scopes every send:
// an agent of another customer has no device there. A failed send is logged by the notification handler and
// is not retried; a panic of a send is logged and does not stop the others.
func (h *eventHandler) sendAll(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, kind notification.Kind, data map[string]string) int {
	log := logrus.WithFields(logrus.Fields{
		"func":        "sendAll",
		"customer_id": customerID,
		"kind":        kind,
	})

	parallelism := h.parallelism
	if parallelism < 1 {
		parallelism = 1
	}

	slots := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	skipped := 0

	for _, agentID := range agentIDs {
		slots <- struct{}{} // wait for a free slot

		// the context is checked after the wait: the wait can be long
		if ctx.Err() != nil {
			<-slots
			skipped++
			continue
		}

		wg.Add(1)
		go func(agentID uuid.UUID) {
			defer wg.Done()
			defer func() { <-slots }()
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("The send panicked. agent_id: %s, recovered: %v", agentID, fmt.Sprint(r))
				}
			}()

			if _, err := h.notificationHandler.Send(ctx, customerID, agentID, uuid.Nil, kind, data); err != nil {
				log.Errorf("Could not send the notification. agent_id: %s, err: %v", agentID, err)
			}
		}(agentID)
	}
	wg.Wait()

	if skipped > 0 {
		log.Warnf("The event ran out of time. The agents that were not tried: %d", skipped)
	}

	return skipped
}
