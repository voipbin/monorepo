package schedulehandler

import (
	"context"
	"encoding/json"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

// isValidTargetMethod reports whether the given method is an allowed
// dispatch method (design §6: POST/GET/PUT/DELETE whitelist).
func isValidTargetMethod(method string) bool {
	switch sock.RequestMethod(method) {
	case sock.RequestMethodPost, sock.RequestMethodGet, sock.RequestMethodPut, sock.RequestMethodDelete:
		return true
	default:
		return false
	}
}

// toInt normalizes the numeric representations an update-field value can
// arrive as (native int from Go callers, float64 from JSON unmarshal).
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// isValidTargetQueue reports whether the given queue is one of the declared
// *.request queue constants (design §6: the allowlist cannot drift from the
// models/outline enumeration).
func isValidTargetQueue(targetQueue string) bool {
	for _, q := range commonoutline.QueueNameRequestAll() {
		if string(q) == targetQueue {
			return true
		}
	}

	return false
}

// validateNameUnique enforces active-name uniqueness per customer (design
// §4.1). excludeID allows an update to keep its own name.
func (h *scheduleHandler) validateNameUnique(ctx context.Context, customerID uuid.UUID, name string, excludeID uuid.UUID) error {
	existing, err := h.db.ScheduleGetByCustomerIDName(ctx, customerID, name)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil
		}
		return errors.Wrap(err, "could not check the schedule name uniqueness")
	}

	if existing.ID == excludeID {
		return nil
	}

	return cerrors.AlreadyExists(
		commonoutline.ServiceNameSchedulerManager,
		"SCHEDULE_NAME_DUPLICATED",
		"A schedule with the given name already exists.",
	)
}

// Create creates a new schedule. tm_next_run is left NULL — the dispatcher
// initializes it on the next tick (design §5.2).
func (h *scheduleHandler) Create(
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
		"func":        "Create",
		"customer_id": customerID,
		"name":        name,
	})
	log.Debug("Creating a new schedule.")

	if name == "" {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_NAME_EMPTY",
			"The schedule name must not be empty.",
		)
	}

	if errCron := schedule.ValidateCron(cronExpr); errCron != nil {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_CRON_INVALID",
			"The cron expression is invalid or never matches.",
		).Wrap(errCron)
	}

	if !isValidTargetMethod(targetMethod) {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_TARGET_METHOD_INVALID",
			"The target method must be one of POST, GET, PUT, DELETE.",
		)
	}

	if !isValidTargetQueue(targetQueue) {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_TARGET_QUEUE_INVALID",
			"The target queue is not a known request queue.",
		)
	}

	if timeoutMS <= 0 {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_TIMEOUT_INVALID",
			"The timeout_ms must be greater than 0.",
		)
	}

	if retryMax < 0 {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_RETRY_MAX_INVALID",
			"The retry_max must not be negative.",
		)
	}

	if errName := h.validateNameUnique(ctx, customerID, name, uuid.Nil); errName != nil {
		return nil, errName
	}

	res, err := h.dbCreate(ctx, customerID, name, detail, cronExpr, targetQueue, targetURI, targetMethod, targetDataType, targetData, timeoutMS, retryMax, enabled)
	if err != nil {
		log.Errorf("Could not create a schedule. err: %v", err)
		return nil, errors.Wrap(err, "could not create a schedule")
	}

	return res, nil
}

// Get returns schedule info.
func (h *scheduleHandler) Get(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"schedule_id": id,
	})

	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get schedule info. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_NOT_FOUND",
				"The schedule was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	return res, nil
}

// Gets returns schedules.
func (h *scheduleHandler) Gets(ctx context.Context, size uint64, token string, filters map[schedule.Field]any) ([]*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.dbList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get schedules info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates the schedule with the given fields. Cron/queue/method are
// re-validated when present. A cron or enabled change also resets
// tm_next_run to NULL so the dispatcher re-initializes it (design §5.2).
func (h *scheduleHandler) Update(ctx context.Context, id uuid.UUID, fields map[schedule.Field]any) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Update",
		"schedule_id": id,
		"fields":      fields,
	})

	cur, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if errValidate := h.validateUpdateFields(ctx, cur, fields); errValidate != nil {
		return nil, errValidate
	}

	// a cron or enabled change forces the dispatcher to re-initialize the
	// next-run slot (design §5.2)
	if _, ok := fields[schedule.FieldCron]; ok {
		fields[schedule.FieldTMNextRun] = nil
	}
	if _, ok := fields[schedule.FieldEnabled]; ok {
		fields[schedule.FieldTMNextRun] = nil
	}

	res, err := h.dbUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update the schedule. err: %v", err)
		return nil, errors.Wrap(err, "could not update the schedule")
	}

	return res, nil
}

// validateUpdateFields re-validates the updatable fields (design §6).
func (h *scheduleHandler) validateUpdateFields(ctx context.Context, cur *schedule.Schedule, fields map[schedule.Field]any) error {
	if v, ok := fields[schedule.FieldType]; ok {
		t, okType := v.(schedule.Type)
		if !okType {
			s, okStr := v.(string)
			if !okStr {
				return cerrors.InvalidArgument(
					commonoutline.ServiceNameSchedulerManager,
					"SCHEDULE_TYPE_INVALID",
					"The schedule type is invalid.",
				)
			}
			t = schedule.Type(s)
		}
		if t != schedule.TypeRPC {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_TYPE_INVALID",
				"Only the rpc schedule type is supported.",
			)
		}
	}

	if v, ok := fields[schedule.FieldCron]; ok {
		cronExpr, okStr := v.(string)
		if !okStr {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_CRON_INVALID",
				"The cron expression is invalid.",
			)
		}
		if errCron := schedule.ValidateCron(cronExpr); errCron != nil {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_CRON_INVALID",
				"The cron expression is invalid or never matches.",
			).Wrap(errCron)
		}
	}

	if v, ok := fields[schedule.FieldTargetMethod]; ok {
		method, okStr := v.(string)
		if !okStr || !isValidTargetMethod(method) {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_TARGET_METHOD_INVALID",
				"The target method must be one of POST, GET, PUT, DELETE.",
			)
		}
	}

	if v, ok := fields[schedule.FieldTargetQueue]; ok {
		targetQueue, okStr := v.(string)
		if !okStr || !isValidTargetQueue(targetQueue) {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_TARGET_QUEUE_INVALID",
				"The target queue is not a known request queue.",
			)
		}
	}

	if v, ok := fields[schedule.FieldTimeoutMS]; ok {
		timeoutMS, okInt := toInt(v)
		if !okInt || timeoutMS <= 0 {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_TIMEOUT_INVALID",
				"The timeout_ms must be greater than 0.",
			)
		}
	}

	if v, ok := fields[schedule.FieldRetryMax]; ok {
		retryMax, okInt := toInt(v)
		if !okInt || retryMax < 0 {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_RETRY_MAX_INVALID",
				"The retry_max must not be negative.",
			)
		}
	}

	if v, ok := fields[schedule.FieldName]; ok {
		name, okStr := v.(string)
		if !okStr || name == "" {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_NAME_EMPTY",
				"The schedule name must not be empty.",
			)
		}
		if errName := h.validateNameUnique(ctx, cur.CustomerID, name, cur.ID); errName != nil {
			return errName
		}
	}

	return nil
}

// Delete soft-deletes the schedule and returns the deleted row.
func (h *scheduleHandler) Delete(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"schedule_id": id,
	})
	log.Debug("Deleting the schedule info.")

	if _, err := h.Get(ctx, id); err != nil {
		return nil, err
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the schedule. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the schedule")
	}

	return res, nil
}
