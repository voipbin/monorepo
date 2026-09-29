package toolhandler

//go:generate mockgen -package toolhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
)

// ToolHandler defines the interface for tool operations
type ToolHandler interface {
	GetAll() []tool.Tool

	// GetByNames returns the built-in tool definitions matching names,
	// filtered through ai.AllowedToolNames(aiType) first -- a tool is only
	// ever returned if it is both a member of aiType's whitelist AND
	// requested (directly by name, or via the "all" selector in names).
	// This brings ai-manager's own resolver to parity with
	// bin-pipecat-manager/pkg/toolhandler.GetByNames, which already
	// re-applies the whitelist regardless of what names claims (B4):
	// without it, an Insight AI storing tool_names:["all"] resolved every
	// Normal-only built-in tool too.
	GetByNames(aiType ai.Type, names []tool.ToolName) []tool.Tool
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

// GetByNames returns tool definitions filtered by the given names, whose
// caller-claimed selection is re-filtered against aiType's whitelist
// (ai.AllowedToolNames) regardless of what names contains (B4).
func (h *toolHandler) GetByNames(aiType ai.Type, names []tool.ToolName) []tool.Tool {
	if len(names) == 0 {
		return nil
	}

	allowed := ai.AllowedToolNames(aiType)

	// Check for "all"
	hasAll := false
	for _, name := range names {
		if name == tool.ToolNameAll {
			hasAll = true
			break
		}
	}

	var result []tool.Tool
	for _, t := range toolDefinitions {
		if !allowed[t.Name] {
			continue
		}
		if hasAll {
			result = append(result, t)
			continue
		}
		for _, name := range names {
			if t.Name == name {
				result = append(result, t)
				break
			}
		}
	}
	return result
}
