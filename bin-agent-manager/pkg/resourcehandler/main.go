package resourcehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package resourcehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-agent-manager/pkg/dbhandler"
)

// ResourceHandler interface
type ResourceHandler interface {
	// Create(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tagIDs []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	// Delete(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	// Get(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	// Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*agent.Agent, error)
	// Login(ctx context.Context, username, password string) (*agent.Agent, error)
	// UpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	// UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) (*agent.Agent, error)
	// UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error)
	// UpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error)
	// UpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error)
	// UpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) (*agent.Agent, error)

	// EventGroupcallCreated(ctx context.Context, groupcall *cmgroupcall.Groupcall) error
	// EventGroupcallProgressing(ctx context.Context, groupcall *cmgroupcall.Groupcall) error
	// EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
}

type resourceHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewResourceHandler return ResourceHandler interface
func NewResourceHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) ResourceHandler {
	return &resourceHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}

// // checkHash returns true if the given hashstring is correct
// func checkHash(password, hashString string) bool {
// 	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
// 		return false
// 	}

// 	return true
// }

// // GenerateHash generates hash from auth
// func generateHash(password string) (string, error) {
// 	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
// 	return string(bytes), err
// }

// // isEmailValid checks if the email provided is valid by regex.
// // get from https://stackoverflow.com/questions/66624011/how-to-validate-an-email-address-in-go
// func isEmailValid(e string) bool {
// 	emailRegex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
// 	return emailRegex.MatchString(e)
// }
