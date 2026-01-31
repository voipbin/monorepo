package toolhandler

//go:generate mockgen -package toolhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	aitool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// ToolHandler manages fetching and caching tool definitions from ai-manager
type ToolHandler interface {
	// FetchTools fetches all tools from ai-manager and caches them
	FetchTools(ctx context.Context) error

	// GetAll returns all cached tools
	GetAll() []aitool.Tool

	// GetByNames returns tools filtered by the given tool names
	// If names contains "all", returns all tools
	// If names is empty or nil, returns empty slice
	GetByNames(names []aitool.ToolName) []aitool.Tool
}

type toolHandler struct {
	requestHandler requesthandler.RequestHandler

	mu    sync.RWMutex
	tools []aitool.Tool
}

// NewToolHandler creates a new ToolHandler instance
func NewToolHandler(requestHandler requesthandler.RequestHandler) ToolHandler {
	return &toolHandler{
		requestHandler: requestHandler,
		tools:          []aitool.Tool{},
	}
}

// FetchTools fetches all tools from ai-manager and caches them
func (h *toolHandler) FetchTools(ctx context.Context) error {
	log := logrus.WithField("func", "FetchTools")
	log.Info("Fetching tools from ai-manager...")

	tools, err := h.requestHandler.AIV1ToolsGet(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch tools from ai-manager")
	}

	h.mu.Lock()
	h.tools = tools
	h.mu.Unlock()

	log.WithField("tool_count", len(tools)).Info("Successfully fetched and cached tools")
	return nil
}

// GetAll returns all cached tools
func (h *toolHandler) GetAll() []aitool.Tool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]aitool.Tool, len(h.tools))
	copy(result, h.tools)
	return result
}

// GetByNames returns tools filtered by the given tool names
func (h *toolHandler) GetByNames(names []aitool.ToolName) []aitool.Tool {
	if len(names) == 0 {
		return []aitool.Tool{}
	}

	// Check if "all" is in the names
	for _, name := range names {
		if name == aitool.ToolNameAll {
			return h.GetAll()
		}
	}

	// Create a set of requested names for O(1) lookup
	nameSet := make(map[aitool.ToolName]bool, len(names))
	for _, name := range names {
		nameSet[name] = true
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]aitool.Tool, 0, len(names))
	for _, tool := range h.tools {
		if nameSet[tool.Name] {
			result = append(result, tool)
		}
	}

	return result
}
