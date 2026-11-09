package devicehandler

//go:generate mockgen -package devicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/pkg/dbhandler"
)

// maxDevicesPerAgent is the most devices of one agent. Registering one more deletes the oldest by refresh time.
const maxDevicesPerAgent = 10

// maxRegisterTries is the most tries of a registration that keeps losing a duplicate key race.
const maxRegisterTries = 3

// listPageToken is the page token of the internal lists: a time after every record.
const listPageToken = "9999-12-31T23:59:59.999999Z"

// DeviceHandler interface
type DeviceHandler interface {
	// Register registers the token for the agent, or refreshes it when the agent has it already. A token that
	// belongs to another agent or customer moves to this agent.
	Register(ctx context.Context, customerID, agentID uuid.UUID, platform device.Platform, token string) (*device.Device, error)
	Get(ctx context.Context, id uuid.UUID) (*device.Device, error)
	Delete(ctx context.Context, id uuid.UUID) (*device.Device, error)
	// List returns the live devices that match the filters.
	List(ctx context.Context, size uint64, token string, filters map[device.Field]any) ([]*device.Device, error)

	EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error
	EventAgentDeleted(ctx context.Context, a *amagent.Agent) error
}

type deviceHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler

	// now returns the current time. It is a field so the tests can fix it.
	now func() time.Time
}

// NewDeviceHandler return DeviceHandler interface
func NewDeviceHandler(db dbhandler.DBHandler) DeviceHandler {
	return &deviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		now:         time.Now,
	}
}
