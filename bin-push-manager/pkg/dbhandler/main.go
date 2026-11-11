package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/models/notification"
)

// DBHandler interface
type DBHandler interface {
	// DeviceCreate inserts the device. It returns ErrDuplicate when the token already exists.
	DeviceCreate(ctx context.Context, d *device.Device) error
	DeviceGet(ctx context.Context, id uuid.UUID) (*device.Device, error)
	DeviceGetByToken(ctx context.Context, token string) (*device.Device, error)
	DeviceList(ctx context.Context, size uint64, token string, filters map[device.Field]any) ([]*device.Device, error)
	// DeviceTouch sets tm_update of the device to now.
	DeviceTouch(ctx context.Context, id uuid.UUID) error
	DeviceDelete(ctx context.Context, id uuid.UUID) error
	// DeviceDeleteByTokenNotOwner deletes the device that has the token and a different owner.
	DeviceDeleteByTokenNotOwner(ctx context.Context, token string, customerID, agentID uuid.UUID) error
	DeviceDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error
	DeviceDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error

	// KindPrefDisable switches the kind off for the agent. A kind that has no row is on. Repeating it succeeds.
	KindPrefDisable(ctx context.Context, customerID, agentID uuid.UUID, kind notification.Kind) error
	// KindPrefEnable switches the kind on for the agent by deleting the row. Repeating it succeeds.
	KindPrefEnable(ctx context.Context, customerID, agentID uuid.UUID, kind notification.Kind) error
	KindPrefListDisabledByAgent(ctx context.Context, customerID, agentID uuid.UUID) ([]notification.Kind, error)
	// KindPrefListDisabledByAgents returns the agents of the list that switched the kind off.
	KindPrefListDisabledByAgents(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, kind notification.Kind) ([]uuid.UUID, error)
	KindPrefDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error
	KindPrefDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error

	// MuteCreate mutes the chat for the agent. Repeating it succeeds.
	MuteCreate(ctx context.Context, customerID, agentID, chatID uuid.UUID) error
	// MuteDelete unmutes the chat for the agent. Repeating it succeeds.
	MuteDelete(ctx context.Context, customerID, agentID, chatID uuid.UUID) error
	MuteListByAgent(ctx context.Context, customerID, agentID uuid.UUID) ([]uuid.UUID, error)
	// MuteListMutedByAgents returns the agents of the list that muted the chat.
	MuteListMutedByAgents(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, chatID uuid.UUID) ([]uuid.UUID, error)
	MuteDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error
	MuteDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
}

// handler errors
var (
	ErrNotFound  = fmt.Errorf("record not found")
	ErrDuplicate = fmt.Errorf("duplicate record")
)

// mysqlErrDupEntry is the MySQL and MariaDB error number of a unique key violation.
const mysqlErrDupEntry = 1062

// isDuplicate returns true if the error is a unique key violation of the database.
func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == mysqlErrDupEntry
}

// NewHandler creates DBHandler
func NewHandler(db *sql.DB) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
	}
	return h
}
