package variablehandler

//go:generate mockgen -package variablehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"regexp"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

const (
	constVariableReferenceData = "voipbin.flow.reference_data"
)

// variableHandler struct
type variableHandler struct {
	db             dbhandler.DBHandler
	requestHandler requesthandler.RequestHandler
}

// VariableHandler interface
type VariableHandler interface {
	Create(ctx context.Context, activeflowID uuid.UUID, variables map[string]string) (*variable.Variable, error)
	Get(ctx context.Context, id uuid.UUID) (*variable.Variable, error)
	Set(ctx context.Context, t *variable.Variable) error

	SetVariable(ctx context.Context, id uuid.UUID, variables map[string]string) error
	DeleteVariable(ctx context.Context, id uuid.UUID, key string) error

	Substitute(ctx context.Context, id uuid.UUID, data string) (string, error)
	SubstituteOption(ctx context.Context, data map[string]any, vars *variable.Variable)
}

// NewVariableHandler return VariableHandler
func NewVariableHandler(
	db dbhandler.DBHandler,
	requestHandler requesthandler.RequestHandler,
) VariableHandler {
	h := &variableHandler{
		db:             db,
		requestHandler: requestHandler,
	}

	return h
}

var (
	regexVariable = regexp.MustCompile(`\${(.*?)}`)
)
