package variablehandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

//go:generate mockgen -package variablehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// variableHandler struct
type variableHandler struct {
	db dbhandler.DBHandler
}

// VariableHandler interface
type VariableHandler interface {
	Create(ctx context.Context, activeflowID uuid.UUID, variables map[string]string) (*variable.Variable, error)
	Get(ctx context.Context, id uuid.UUID) (*variable.Variable, error)
	Set(ctx context.Context, t *variable.Variable) error

	SetVariable(ctx context.Context, id uuid.UUID, variables map[string]string) error
	DeleteVariable(ctx context.Context, id uuid.UUID, key string) error

	Substitute(ctx context.Context, id uuid.UUID, data string) (string, error)
	SubstituteString(ctx context.Context, data string, v *variable.Variable) string
	SubstituteByte(ctx context.Context, data []byte, v *variable.Variable) []byte
}

// NewVariableHandler return VariableHandler
func NewVariableHandler(
	db dbhandler.DBHandler,
) VariableHandler {
	h := &variableHandler{
		db: db,
	}

	return h
}
