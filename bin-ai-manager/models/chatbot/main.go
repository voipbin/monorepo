package chatbot

import (
	"monorepo/bin-common-handler/models/identity"
	"strings"
)

// Chatbot define
type Chatbot struct {
	identity.Identity

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineType  EngineType     `json:"engine_type,omitempty"`
	EngineModel EngineModel    `json:"engine_model,omitempty"`
	EngineData  map[string]any `json:"engine_data,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	// timestamp
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// EngineType define
type EngineType string

// list of engine types
const (
	EngineTypeNone EngineType = ""
)

type EngineModelTarget string

const (
	EngineModelTargetNone       EngineModelTarget = ""
	EngineModelTargetOpenai     EngineModelTarget = "openai"     // openai. https://chat.openai.com/chat
	EngineModelTargetDialogflow EngineModelTarget = "dialogflow" // dialogflow use
)

type EngineModel string

// list of engine models
const (
	EngineModelOpenaiO1Mini            EngineModel = "openai.o1-mini"
	EngineModelOpenaiO1Preview         EngineModel = "openai.o1-preview"
	EngineModelOpenaiO1                EngineModel = "openai.o1"
	EngineModelOpenaiO3Mini            EngineModel = "openai.o3-mini"
	EngineModelOpenaiGPT4O             EngineModel = "openai.gpt-4o"
	EngineModelOpenaiGPT4OMini         EngineModel = "openai.gpt-4o-mini"
	EngineModelOpenaiGPT4Turbo         EngineModel = "openai.gpt-4-turbo"
	EngineModelOpenaiGPT4VisionPreview EngineModel = "openai.gpt-4-vision-preview"
	EngineModelOpenaiGPT4              EngineModel = "openai.gpt-4"
	EngineModelOpenaiGPT3Dot5Turbo     EngineModel = "openai.gpt-3.5-turbo"

	EngineModelDialogflowCX EngineModel = "dialogflow.cx"
	EngineModelDialogflowES EngineModel = "dialogflow.es"
)

func GetEngineModelTarget(engineModel EngineModel) EngineModelTarget {
	mapModelTarget := map[EngineModel]EngineModelTarget{
		EngineModelOpenaiO1Mini:            EngineModelTargetOpenai,
		EngineModelOpenaiO1Preview:         EngineModelTargetOpenai,
		EngineModelOpenaiO1:                EngineModelTargetOpenai,
		EngineModelOpenaiO3Mini:            EngineModelTargetOpenai,
		EngineModelOpenaiGPT4O:             EngineModelTargetOpenai,
		EngineModelOpenaiGPT4OMini:         EngineModelTargetOpenai,
		EngineModelOpenaiGPT4Turbo:         EngineModelTargetOpenai,
		EngineModelOpenaiGPT4VisionPreview: EngineModelTargetOpenai,
		EngineModelOpenaiGPT4:              EngineModelTargetOpenai,
		EngineModelOpenaiGPT3Dot5Turbo:     EngineModelTargetOpenai,

		EngineModelDialogflowCX: EngineModelTargetDialogflow,
		EngineModelDialogflowES: EngineModelTargetDialogflow,
	}

	res, ok := mapModelTarget[engineModel]
	if !ok {
		return EngineModelTargetNone
	}

	return res
}

func GetEngineModelName(engineModel EngineModel) string {
	tmp := strings.Split(string(engineModel), ".")
	if len(tmp) < 2 {
		return ""
	}
	return tmp[1]
}
