package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/sirupsen/logrus"
)

// processEventCMConferenceUpdated handles the conference-manager's conference_deleted
// event (VOIP-1422: bound to conference_deleted, not conference_updated -- see the
// topicPatterns doc comment in main.go for why). Despite the function name (kept as-is
// to minimize the diff), the dispatch case that reaches this function is keyed on
// EventTypeConferenceDeleted, and the downstream summaryHandler.EventCMConferenceUpdated
// deliberately does not gate on the payload's Status field -- see that function's doc
// comment.
func (h *subscribeHandler) processEventCMConferenceUpdated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConferenceUpdated",
		"event": m,
	})

	evt := cfconference.Conference{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	go h.summaryHandler.EventCMConferenceUpdated(context.Background(), &evt)

	return nil
}
