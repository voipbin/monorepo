package toolhandler

//go:generate mockgen -package toolhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"monorepo/bin-ai-manager/models/tool"
)

// ToolHandler defines the interface for tool operations
type ToolHandler interface {
	GetAll() []tool.Tool
	GetByNames(names []tool.ToolName) []tool.Tool
}

type toolHandler struct{}

// NewToolHandler creates a new ToolHandler
func NewToolHandler() ToolHandler {
	return &toolHandler{}
}

// GetAll returns all available tool definitions
func (h *toolHandler) GetAll() []tool.Tool {
	return toolDefinitions
}

// GetByNames returns tool definitions filtered by the given names
// If names contains ToolNameAll, all tools are returned
func (h *toolHandler) GetByNames(names []tool.ToolName) []tool.Tool {
	if len(names) == 0 {
		return nil
	}

	// Check for "all"
	for _, name := range names {
		if name == tool.ToolNameAll {
			return toolDefinitions
		}
	}

	// Filter to requested tools
	var result []tool.Tool
	for _, t := range toolDefinitions {
		for _, name := range names {
			if t.Name == name {
				result = append(result, t)
				break
			}
		}
	}
	return result
}
