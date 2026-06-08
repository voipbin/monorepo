package eventhandler

import (
	"context"
	"sort"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// maxCorrelationResources caps the number of resources returned in a single
// correlation graph. A single activeflow rarely links more than this; overflow
// is signaled via the Truncated flag. Raising this requires a code change.
const maxCorrelationResources = 100

// ResourceCorrelationGet resolves a resource id to its activeflow and returns
// the correlation graph of all resources sharing that activeflow, grouped by
// publisher. Returns an empty graph (ResourceFound distinguishes the cases)
// when the resource has no activeflow.
func (h *eventHandler) ResourceCorrelationGet(ctx context.Context, resourceID uuid.UUID) (*response.V1DataResourceCorrelationGet, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ResourceCorrelationGet",
		"resource_id": resourceID,
	})

	if resourceID == uuid.Nil {
		return nil, errors.New("resource_id is required")
	}

	// 1. resource_id -> activeflow_id reverse lookup (deterministic).
	activeflowIDStr, err := h.db.ResourceActiveflowIDGet(ctx, resourceID.String())
	if err != nil {
		log.Errorf("Could not get activeflow_id for resource. err: %v", err)
		return nil, errors.Wrap(err, "could not get activeflow_id for resource")
	}

	if activeflowIDStr == "" {
		// No activeflow: either the resource was never seen, or it exists but
		// carries no activeflow. ResourceExists distinguishes the two.
		found, errExists := h.db.ResourceExists(ctx, resourceID.String())
		if errExists != nil {
			log.Errorf("Could not check resource existence. err: %v", errExists)
			return nil, errors.Wrap(errExists, "could not check resource existence")
		}
		return &response.V1DataResourceCorrelationGet{
			ResourceID:    resourceID,
			ResourceFound: found,
			ActiveflowID:  uuid.Nil,
			Truncated:     false,
			Resources:     []*event.PublisherGroup{},
		}, nil
	}

	activeflowID, err := uuid.FromString(activeflowIDStr)
	if err != nil {
		log.Errorf("Could not parse activeflow_id. activeflow_id: %s, err: %v", activeflowIDStr, err)
		return nil, errors.Wrap(err, "could not parse activeflow_id")
	}

	// 2. activeflow_id -> aggregated resource list (limit+1 to detect truncation).
	rows, err := h.db.CorrelatedResourceList(ctx, activeflowIDStr, maxCorrelationResources+1)
	if err != nil {
		log.Errorf("Could not list correlated resources. err: %v", err)
		return nil, errors.Wrap(err, "could not list correlated resources")
	}

	truncated := len(rows) > maxCorrelationResources
	if truncated {
		rows = rows[:maxCorrelationResources]
	}

	// 3. group by publisher with stable ordering.
	groups := groupByPublisher(rows)

	return &response.V1DataResourceCorrelationGet{
		ResourceID:    resourceID,
		ResourceFound: true,
		ActiveflowID:  activeflowID,
		Truncated:     truncated,
		Resources:     groups,
	}, nil
}

// groupByPublisher converts aggregated rows into publisher-grouped, stably
// sorted output. Rows whose resource_id fails to parse as a UUID are skipped
// (with a warning) so a single malformed payload does not fail the whole call.
func groupByPublisher(rows []*event.CorrelatedRow) []*event.PublisherGroup {
	log := logrus.WithField("func", "groupByPublisher")

	byPublisher := map[string][]*event.CorrelatedResource{}
	for _, r := range rows {
		id, err := uuid.FromString(r.ResourceID)
		if err != nil {
			log.Warnf("Skipping row with unparseable resource_id. resource_id: %s, err: %v", r.ResourceID, err)
			continue
		}

		eventTypes := make([]string, len(r.EventTypes))
		copy(eventTypes, r.EventTypes)
		sort.Strings(eventTypes)

		byPublisher[r.Publisher] = append(byPublisher[r.Publisher], &event.CorrelatedResource{
			ID:         id,
			DataType:   r.DataType,
			EventTypes: eventTypes,
			FirstSeen:  r.FirstSeen,
			LastSeen:   r.LastSeen,
		})
	}

	publishers := make([]string, 0, len(byPublisher))
	for p := range byPublisher {
		publishers = append(publishers, p)
	}
	sort.Strings(publishers)

	groups := make([]*event.PublisherGroup, 0, len(publishers))
	for _, p := range publishers {
		resources := byPublisher[p]
		sort.SliceStable(resources, func(i, j int) bool {
			return resources[i].FirstSeen.Before(resources[j].FirstSeen)
		})
		groups = append(groups, &event.PublisherGroup{
			Publisher: p,
			Resources: resources,
		})
	}

	return groups
}
