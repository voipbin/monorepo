package schedulehandler

import (
	"context"

	"monorepo/bin-scheduler-manager/models/schedule"
)

// notifyScheduleCreated publishes the internal schedule_created event.
// Phase 1 has no webhook publishing: every schedule is nil-customer and
// customer-visible events arrive with Phase 3 (design §6).
func (h *scheduleHandler) notifyScheduleCreated(ctx context.Context, s *schedule.Schedule) {
	h.notifyHandler.PublishEvent(ctx, schedule.EventTypeScheduleCreated, s)
}

// notifyScheduleUpdated publishes the internal schedule_updated event.
func (h *scheduleHandler) notifyScheduleUpdated(ctx context.Context, s *schedule.Schedule) {
	h.notifyHandler.PublishEvent(ctx, schedule.EventTypeScheduleUpdated, s)
}

// notifyScheduleDeleted publishes the internal schedule_deleted event.
func (h *scheduleHandler) notifyScheduleDeleted(ctx context.Context, s *schedule.Schedule) {
	h.notifyHandler.PublishEvent(ctx, schedule.EventTypeScheduleDeleted, s)
}
