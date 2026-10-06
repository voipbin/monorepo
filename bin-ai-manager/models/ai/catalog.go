package ai

// Route defines how a catalog model reaches its provider.
type Route string

const (
	RouteDirect     Route = "direct"
	RouteOpenRouter Route = "openrouter"

	// RouteCustomOpenRouter is the single customer-keyed OpenRouter entry.
	RouteCustomOpenRouter Route = "custom_openrouter"
)

// Key modes describe how the engine key is handled for a model.
const (
	KeyModePlatform     = "platform"       // platform supplies the key
	KeyModeOwnOrDefault = "own_or_default" // customer key, or the platform default when empty
	KeyModeOwnRequired  = "own_required"   // customer key is mandatory
)

// EngineModelPrefixCustomOpenRouter is prepended to the model ID the customer types.
const EngineModelPrefixCustomOpenRouter = "openrouter."

// TagLowCost marks models with a low per-token price.
const TagLowCost = "low-cost"

// ModelEntry is internal. Route and UpstreamSlug never leave the platform.
type ModelEntry struct {
	ID           EngineModel
	Label        string
	Vendor       string
	Route        Route
	UpstreamSlug string
	Recommended  bool
	Tags         []string
	Description  string
	CustomPrefix string // set only for entries where the customer types the model ID
}

// ModelInfo is the customer-facing view returned by GET /ai_models.
type ModelInfo struct {
	ID              EngineModel `json:"id"`
	Label           string      `json:"label"`
	Vendor          string      `json:"vendor"`
	Recommended     bool        `json:"recommended"`
	Tags            []string    `json:"tags"`
	Description     string      `json:"description"`
	PlatformManaged bool        `json:"platform_managed"`
	KeyMode         string      `json:"key_mode"`
	ModelIDPrefix   string      `json:"model_id_prefix,omitempty"`
}

// CatalogView returns the customer-facing view of the catalog.
func CatalogView() []ModelInfo {
	res := make([]ModelInfo, 0, len(catalog))
	for _, e := range catalog {
		tags := e.Tags
		if tags == nil {
			tags = []string{}
		}
		keyMode := KeyModeOwnOrDefault
		switch e.Route {
		case RouteOpenRouter:
			keyMode = KeyModePlatform
		case RouteCustomOpenRouter:
			keyMode = KeyModeOwnRequired
		}

		res = append(res, ModelInfo{
			ID:              e.ID,
			Label:           e.Label,
			Vendor:          e.Vendor,
			Recommended:     e.Recommended,
			Tags:            tags,
			Description:     e.Description,
			PlatformManaged: e.Route == RouteOpenRouter,
			KeyMode:         keyMode,
			ModelIDPrefix:   e.CustomPrefix,
		})
	}
	return res
}

var lowCost = []string{TagLowCost}

// catalog is the curated allow-list of selectable engine models.
var catalog = []ModelEntry{
	// direct models (customer supplies the provider key)
	{ID: EngineModelGeminiGemini2Dot5Flash, Label: "Gemini 2.5 Flash", Vendor: "Google", Route: RouteDirect, Recommended: true, Tags: lowCost, Description: "Fast, balanced model for most voice conversations."},
	{ID: EngineModelGeminiGemini2Dot5Pro, Label: "Gemini 2.5 Pro", Vendor: "Google", Route: RouteDirect, Description: "Higher reasoning quality for complex conversations."},
	{ID: EngineModelGeminiGemini2Dot0Flash, Label: "Gemini 2.0 Flash", Vendor: "Google", Route: RouteDirect, Description: "Previous generation fast model."},
	{ID: EngineModelGeminiGeminiProLatest, Label: "Gemini Pro (latest)", Vendor: "Google", Route: RouteDirect, Description: "Always points to the latest Gemini Pro model."},

	{ID: EngineModelOpenaiGPT5Dot2, Label: "GPT-5.2", Vendor: "OpenAI", Route: RouteDirect, Description: "Latest flagship general-purpose model."},
	{ID: EngineModelOpenaiGPT5Dot1, Label: "GPT-5.1", Vendor: "OpenAI", Route: RouteDirect, Description: "Flagship general-purpose model."},
	{ID: EngineModelOpenaiGPT5, Label: "GPT-5", Vendor: "OpenAI", Route: RouteDirect, Description: "General-purpose model with strong reasoning."},
	{ID: EngineModelOpenaiGPT5Mini, Label: "GPT-5 mini", Vendor: "OpenAI", Route: RouteDirect, Tags: lowCost, Description: "Smaller, faster and cheaper GPT-5 variant."},
	{ID: EngineModelOpenaiGPT5Nano, Label: "GPT-5 nano", Vendor: "OpenAI", Route: RouteDirect, Tags: lowCost, Description: "Smallest and cheapest GPT-5 variant."},

	{ID: EngineModelGrok3, Label: "Grok 3", Vendor: "xAI", Route: RouteDirect, Description: "General-purpose model from xAI."},
	{ID: EngineModelGrok3Mini, Label: "Grok 3 mini", Vendor: "xAI", Route: RouteDirect, Tags: lowCost, Description: "Smaller and cheaper Grok 3 variant."},

	// platform-managed models (no customer key needed)
	{ID: "anthropic.claude-haiku-4.5", Label: "Claude Haiku 4.5", Vendor: "Anthropic", Route: RouteOpenRouter, UpstreamSlug: "anthropic/claude-haiku-4.5", Description: "Fast Claude model for responsive conversations."},
	{ID: "anthropic.claude-sonnet-4.5", Label: "Claude Sonnet 4.5", Vendor: "Anthropic", Route: RouteOpenRouter, UpstreamSlug: "anthropic/claude-sonnet-4.5", Description: "Balanced Claude model with strong instruction following."},
	{ID: "meta.llama-3.3-70b-instruct", Label: "Llama 3.3 70B Instruct", Vendor: "Meta", Route: RouteOpenRouter, UpstreamSlug: "meta-llama/llama-3.3-70b-instruct", Tags: lowCost, Description: "Open-weight 70B model for general conversations."},
	{ID: "meta.llama-4-maverick", Label: "Llama 4 Maverick", Vendor: "Meta", Route: RouteOpenRouter, UpstreamSlug: "meta-llama/llama-4-maverick", Tags: lowCost, Description: "Open-weight Llama 4 model for general conversations."},
	{ID: "deepseek.deepseek-v3.2", Label: "DeepSeek V3.2", Vendor: "DeepSeek", Route: RouteOpenRouter, UpstreamSlug: "deepseek/deepseek-v3.2", Tags: lowCost, Description: "Capable general-purpose model at a low price."},
	{ID: "qwen.qwen3-235b-a22b-2507", Label: "Qwen3 235B A22B", Vendor: "Qwen", Route: RouteOpenRouter, UpstreamSlug: "qwen/qwen3-235b-a22b-2507", Tags: lowCost, Description: "Large Qwen3 model with strong multilingual support."},
	{ID: "qwen.qwen3-30b-a3b-instruct-2507", Label: "Qwen3 30B A3B Instruct", Vendor: "Qwen", Route: RouteOpenRouter, UpstreamSlug: "qwen/qwen3-30b-a3b-instruct-2507", Tags: lowCost, Description: "Compact, fast Qwen3 model."},
	{ID: "mistral.mistral-medium-3.1", Label: "Mistral Medium 3.1", Vendor: "Mistral", Route: RouteOpenRouter, UpstreamSlug: "mistralai/mistral-medium-3.1", Description: "Mid-size Mistral model for general conversations."},
	{ID: "mistral.mistral-small-3.2-24b-instruct", Label: "Mistral Small 3.2 24B Instruct", Vendor: "Mistral", Route: RouteOpenRouter, UpstreamSlug: "mistralai/mistral-small-3.2-24b-instruct", Tags: lowCost, Description: "Small, fast and low-cost Mistral model."},

	// customer-keyed OpenRouter entry (the customer types the model ID)
	{ID: "custom.openrouter", Label: "OpenRouter model (your OpenRouter key)", Vendor: "OpenRouter", Route: RouteCustomOpenRouter, CustomPrefix: EngineModelPrefixCustomOpenRouter, Description: "Enter any model supported by OpenRouter."},
}
