package schedulehandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-schedule-manager/models/schedule"
)

// dbCreate creates a new schedule record. The type is forced to rpc (design
// §4.1) and tm_next_run stays NULL for the dispatcher to initialize.
func (h *scheduleHandler) dbCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	cronExpr string,
	targetQueue string,
	targetURI string,
	targetMethod string,
	targetDataType string,
	targetData json.RawMessage,
	timeoutMS int,
	retryMax int,
	enabled bool,
) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbCreate",
		"customer_id": customerID,
		"name":        name,
	})

	id := h.utilHandler.UUIDCreate()
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Name:   name,
		Detail: detail,

		Type: schedule.TypeRPC,
		Cron: cronExpr,

		TargetQueue:    targetQueue,
		TargetURI:      targetURI,
		TargetMethod:   targetMethod,
		TargetDataType: targetDataType,
		TargetData:     targetData,

		TimeoutMS: timeoutMS,
		RetryMax:  retryMax,
		Enabled:   enabled,
	}
	log = log.WithField("schedule_id", id)

	if err := h.db.ScheduleCreate(ctx, s); err != nil {
		log.Errorf("Could not create a new schedule. err: %v", err)
		return nil, err
	}

	res, err := h.db.ScheduleGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created schedule info. err: %v", err)
		return nil, err
	}

	h.notifyScheduleCreated(ctx, res)
	log.WithField("schedule", res).Debug("Created a new schedule.")

	return res, nil
}

// dbGet returns schedule info.
func (h *scheduleHandler) dbGet(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbGet",
		"schedule_id": id,
	})

	res, err := h.db.ScheduleGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get schedule info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbList returns schedules.
func (h *scheduleHandler) dbList(ctx context.Context, size uint64, token string, filters map[schedule.Field]any) ([]*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.ScheduleList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get schedules info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbUpdate updates the schedule with the given fields.
func (h *scheduleHandler) dbUpdate(ctx context.Context, id uuid.UUID, fields map[schedule.Field]any) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbUpdate",
		"schedule_id": id,
	})

	if err := h.db.ScheduleUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the schedule. err: %v", err)
		return nil, err
	}

	res, err := h.db.ScheduleGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated schedule info. err: %v", err)
		return nil, err
	}

	h.notifyScheduleUpdated(ctx, res)

	return res, nil
}

// dbDelete soft-deletes the schedule info.
func (h *scheduleHandler) dbDelete(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbDelete",
		"schedule_id": id,
	})

	if err := h.db.ScheduleDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the schedule. err: %v", err)
		return nil, err
	}

	res, err := h.db.ScheduleGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted schedule info. err: %v", err)
		return nil, err
	}

	h.notifyScheduleDeleted(ctx, res)

	return res, nil
}
