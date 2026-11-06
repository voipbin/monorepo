package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

const (
	updateBackoffMax = 50 * time.Millisecond // random delay between two tries of a conflicting write

	callbackWriteTries  = 12
	callbackWriteBudget = 2 * time.Second // reads, writes and retries of one per target callback
	callbackMaxFailures = 3               // consecutive failed callbacks after which the callbacks of a dispatch are skipped

	receiptBudget   = time.Second // all the work of one receipt
	eventReadBudget = time.Second // the read of the row for an event
)

// budgets of the two phases of the final write; variables so that a test can shorten them.
var (
	finalWriteCASBudget   = 20 * time.Second // compare and set phase
	finalWriteMergeBudget = 10 * time.Second // unconditional merge phase
)

var (
	// errNoChange is returned by a change function that has nothing to write.
	errNoChange = errors.New("no change")
	// errTargetsMismatch is returned when the stored targets are not the targets that the writer works on.
	errTargetsMismatch = errors.New("the stored targets do not match")
	// errUpdateConflict is returned when the write lost the race on every try.
	errUpdateConflict = errors.New("the targets update conflicted")
)

// updateOpts are the limits of updateTargets.
type updateOpts struct {
	maxTries int                   // 0 means as many tries as the budget allows
	budget   time.Duration         // total time of all the tries
	provider *message.ProviderName // set the provider name too when not nil
}

// updateTargets changes the targets of a message. It reads the row from the database (not the cache), applies fn to a
// copy of the targets and writes it only if nobody wrote the row in between (compare on tm_update). On a conflict it
// reads again and retries after a random delay of up to updateBackoffMax. fn returns errNoChange to skip the write.
// It returns true if a row was written.
func (h *messageHandler) updateTargets(ctx context.Context, id uuid.UUID, opts updateOpts, fn func(targets []target.Target) error) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, opts.budget)
	defer cancel()

	for try := 1; ; try++ {
		m, err := h.db.MessageGetFromDB(ctx, id)
		if err != nil {
			return false, fmt.Errorf("could not get the message: %w", err)
		}

		targets := append([]target.Target(nil), m.Targets...)
		if errFn := fn(targets); errFn != nil {
			if errors.Is(errFn, errNoChange) {
				return false, nil
			}
			return false, errFn
		}

		ok, err := h.db.MessageUpdateTargetsIf(ctx, id, m.TMUpdate, opts.provider, targets)
		if err != nil {
			return false, fmt.Errorf("could not update the targets: %w", err)
		}
		if ok {
			return true, nil
		}

		if opts.maxTries > 0 && try >= opts.maxTries {
			return false, errUpdateConflict
		}

		select {
		case <-ctx.Done():
			return false, fmt.Errorf("%w: %w", errUpdateConflict, ctx.Err())
		case <-time.After(time.Duration(rand.Int63n(int64(updateBackoffMax) + 1))): //nolint:gosec // retry jitter, not a security value
		}
	}
}

// mergeTargets is the last resort of the final write: it reads the row, applies fn and writes the result without the
// comparison. A receipt that is committed between the read and the write could be overwritten.
func (h *messageHandler) mergeTargets(ctx context.Context, id uuid.UUID, provider message.ProviderName, fn func(targets []target.Target) error) error {
	m, err := h.db.MessageGetFromDB(ctx, id)
	if err != nil {
		return fmt.Errorf("could not get the message: %w", err)
	}

	targets := append([]target.Target(nil), m.Targets...)
	if errFn := fn(targets); errFn != nil {
		if errors.Is(errFn, errNoChange) {
			return nil
		}
		return errFn
	}

	if errUpdate := h.db.MessageUpdateTargets(ctx, id, provider, targets); errUpdate != nil {
		return fmt.Errorf("could not update the targets: %w", errUpdate)
	}

	return nil
}

// publishUpdated publishes the message_updated event with the latest row from the database (not the cache, which is
// refreshed on a best effort). The call is detached from the cancellation of ctx because the customer webhook is sent
// asynchronously with the context. The internal provider keys are removed from the published copy.
func (h *messageHandler) publishUpdated(ctx context.Context, id uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "publishUpdated",
		"message_id": id,
	})

	pubCtx := context.WithoutCancel(ctx)

	readCtx, cancel := context.WithTimeout(pubCtx, eventReadBudget)
	defer cancel()

	m, err := h.db.MessageGetFromDB(readCtx, id)
	if err != nil {
		promEventDroppedTotal.Inc()
		log.Errorf("Could not read the message for the event. The event is dropped. err: %v", err)
		return
	}

	h.notifyHandler.PublishWebhookEvent(pubCtx, m.CustomerID, message.EventTypeMessageUpdated, m.EventCopy())
}
