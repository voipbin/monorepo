package toolhandler

//go:generate mockgen -package toolhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	amai "monorepo/bin-ai-manager/models/ai"
	aitool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// ToolHandler manages fetching and caching tool definitions from ai-manager.
//
// GetAll (all cached tools, no type filtering) is intentionally NOT part of
// this interface -- see docs/plans/
// 2026-07-30-case-insight-assistant-tool-expansion-design.md §2.4. Every
// caller must go through GetByNames so the AIType whitelist (models/ai.
// AllowedToolNames) is re-applied at expansion time regardless of what a
// stored tool_names value claims, closing both the "Normal `all` leaks
// Insight tools" bug and a fail-open path a future caller could otherwise
// silently reintroduce by calling an unfiltered GetAll() directly.
type ToolHandler interface {
	// FetchTools fetches all tools from ai-manager and caches them
	FetchTools(ctx context.Context) error

	// GetByNames returns the cached tools matching names, filtered through
	// amai.AllowedToolNames(aiType) first -- a tool is only ever returned if
	// it is both a member of the AIType's whitelist AND requested (directly
	// by name, or via the "all" selector in names). aiType is resolved
	// fresh by the caller from the AI record already in hand on every call,
	// never cached across calls.
	GetByNames(aiType amai.Type, names []aitool.ToolName) []aitool.Tool
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

	tools, err := h.requestHandler.AIV1ToolList(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch tools from ai-manager")
	}

	h.mu.Lock()
	h.tools = tools
	h.mu.Unlock()

	log.WithField("tool_count", len(tools)).Info("Successfully fetched and cached tools")
	return nil
}

// containsName reports whether target is present in names.
func containsName(names []aitool.ToolName, target aitool.ToolName) bool {
	for _, n := range names {
		if n == target {
			return true
		}
	}
	return false
}

// GetByNames returns tools filtered by the given tool names, with the
// AIType whitelist re-applied at expansion time regardless of what names
// contains (defense-in-depth, design §2.3): whether names is an explicit
// list or contains the "all" selector, every returned tool must first be a
// member of amai.AllowedToolNames(aiType). This closes the pre-existing gap
// where Normal AI's tool_names=["all"] returned Insight-only tools
// alongside every Normal tool -- the type separation ValidateToolNames
// enforces at save time was not previously re-checked here.
func (h *toolHandler) GetByNames(aiType amai.Type, names []aitool.ToolName) []aitool.Tool {
	allowed := amai.AllowedToolNames(aiType)
	hasAll := containsName(names, aitool.ToolNameAll)

	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]aitool.Tool, 0, len(h.tools))
	for _, t := range h.tools {
		if !allowed[t.Name] {
			continue // concrete-name check only; "all" is never itself checked for set membership
		}
		if hasAll || containsName(names, t.Name) {
			result = append(result, t)
		}
	}

	return result
}
