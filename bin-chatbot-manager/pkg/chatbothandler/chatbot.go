package chatbothandler

import (
	"context"
	"fmt"
	"monorepo/bin-chatbot-manager/models/chatbot"

	"github.com/gofrs/uuid"
)

func (h *chatbotHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType chatbot.EngineType,
	engineModel chatbot.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*chatbot.Chatbot, error) {

	target := chatbot.GetEngineModelTarget(engineModel)
	if target == chatbot.EngineModelTargetNone {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	return h.dbCreate(ctx, customerID, name, detail, engineType, engineModel, engineData, initPrompt)
}

// Update updates the chatbot info
func (h *chatbotHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineType chatbot.EngineType,
	engineModel chatbot.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*chatbot.Chatbot, error) {

	target := chatbot.GetEngineModelTarget(engineModel)
	if target == chatbot.EngineModelTargetNone {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	return h.dbUpdate(ctx, id, name, detail, engineType, engineModel, engineData, initPrompt)
}
