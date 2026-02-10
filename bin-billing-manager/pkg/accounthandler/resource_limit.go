package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
)

// IsValidResourceLimitByCustomerID checks if the given customer has not exceeded its plan limit for the given resource type.
func (h *accountHandler) IsValidResourceLimitByCustomerID(ctx context.Context, customerID uuid.UUID, resourceType account.ResourceType) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "IsValidResourceLimitByCustomerID",
		"customer_id":   customerID,
		"resource_type": resourceType,
	})

	a, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	res, err := h.IsValidResourceLimit(ctx, a.ID, resourceType)
	if err != nil {
		log.Errorf("Could not validate the account's resource limit. err: %v", err)
		return false, errors.Wrap(err, "could not validate the account's resource limit")
	}

	return res, nil
}

// IsValidResourceLimit checks if the given account has not exceeded its plan limit for the given resource type.
// It fetches the account's plan, gets the current resource count from the appropriate manager,
// and compares against the plan limit.
func (h *accountHandler) IsValidResourceLimit(ctx context.Context, accountID uuid.UUID, resourceType account.ResourceType) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "IsValidResourceLimit",
		"account_id":    accountID,
		"resource_type": resourceType,
	})

	a, err := h.Get(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, fmt.Errorf("could not get account info: %w", err)
	}
	log.WithField("account", a).Debugf("Retrieved account info. account_id: %s", a.ID)

	if a.TMDelete != nil {
		log.Debugf("The account has been deleted already. account_id: %s", a.ID)
		return false, nil
	}

	planLimits, ok := account.PlanLimitMap[a.PlanType]
	if !ok {
		log.Errorf("Unknown plan type. plan_type: %s", a.PlanType)
		return false, fmt.Errorf("unknown plan type: %s", a.PlanType)
	}

	limit := planLimits.GetLimit(resourceType)

	// 0 means unlimited
	if limit == 0 {
		return true, nil
	}

	currentCount, err := h.getResourceCount(ctx, a.CustomerID, resourceType)
	if err != nil {
		log.Errorf("Could not get resource count. err: %v", err)
		return false, fmt.Errorf("could not get resource count: %w", err)
	}
	log.Debugf("Current resource count. resource_type: %s, count: %d, limit: %d", resourceType, currentCount, limit)

	if currentCount >= limit {
		log.Infof("Resource limit reached. resource_type: %s, count: %d, limit: %d", resourceType, currentCount, limit)
		return false, nil
	}

	return true, nil
}

// getResourceCount gets the current count of the given resource type for the customer.
func (h *accountHandler) getResourceCount(ctx context.Context, customerID uuid.UUID, resourceType account.ResourceType) (int, error) {
	switch resourceType {
	case account.ResourceTypeExtension:
		return h.reqHandler.RegistrarV1ExtensionCountByCustomerID(ctx, customerID)
	case account.ResourceTypeTrunk:
		return h.reqHandler.RegistrarV1TrunkCountByCustomerID(ctx, customerID)
	case account.ResourceTypeAgent:
		return h.reqHandler.AgentV1AgentCountByCustomerID(ctx, customerID)
	case account.ResourceTypeQueue:
		return h.reqHandler.QueueV1QueueCountByCustomerID(ctx, customerID)
	case account.ResourceTypeFlow:
		return h.reqHandler.FlowV1FlowCountByCustomerID(ctx, customerID)
	case account.ResourceTypeConference:
		return h.reqHandler.ConferenceV1ConferenceCountByCustomerID(ctx, customerID)
	case account.ResourceTypeVirtualNumber:
		return h.reqHandler.NumberV1VirtualNumberCountByCustomerID(ctx, customerID)
	default:
		return 0, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
