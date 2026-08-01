package schedulehandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-schedule-manager/models/execution"
)

// pruneBatchSize bounds the number of rows removed per DELETE statement so a
// large backlog cannot hold a long transaction (design §6).
const pruneBatchSize = 1000

// ExecutionGets returns executions.
func (h *scheduleHandler) ExecutionGets(ctx context.Context, size uint64, token string, filters map[execution.Field]any) ([]*execution.Execution, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ExecutionGets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.ExecutionList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get executions info. err: %v", err)
		return nil, errors.Wrap(err, "could not get executions info")
	}

	return res, nil
}

// ExecutionPrune deletes execution rows older than retentionDays, in batches
// of pruneBatchSize until a batch removes nothing (design §6). It returns the
// total number of removed rows.
func (h *scheduleHandler) ExecutionPrune(ctx context.Context, retentionDays int) (int64, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ExecutionPrune",
		"retention_days": retentionDays,
	})

	// guard against a misconfigured retention: 0 or negative days would compute
	// a cutoff of now-or-future and hard-delete the entire audit table
	if retentionDays <= 0 {
		return 0, cerrors.InvalidArgument(
			commonoutline.ServiceNameScheduleManager,
			"EXECUTION_RETENTION_INVALID",
			"The execution retention days must be greater than 0.",
		)
	}

	now := h.utilHandler.TimeNow()
	cutoff := now.AddDate(0, 0, -retentionDays)

	var res int64
	for {
		removed, err := h.db.ExecutionPrune(ctx, cutoff, pruneBatchSize)
		if err != nil {
			log.Errorf("Could not prune the executions. err: %v", err)
			return res, errors.Wrap(err, "could not prune the executions")
		}

		res += removed
		if removed == 0 {
			break
		}
	}

	log.Debugf("Pruned the executions. removed: %d", res)
	return res, nil
}
