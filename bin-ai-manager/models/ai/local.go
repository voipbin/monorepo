package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Operator-registered local (self-hosted, OpenAI-compatible) models.
// The model list carries no URL and no key; those live in the runner only.

// EngineModelPrefixLocal is the engine model prefix of a local model: local.<endpoint>.<id>.
const EngineModelPrefixLocal = "local."

// TagToolCalling marks a local model that supports tool calling.
const TagToolCalling = "tool-calling"

const (
	maxLocalModelsInputBytes = 64 * 1024
	maxLocalModels           = 200
	localVendor              = "Local"
	localDescription         = "Local model."
)

var (
	localEndpointPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	localIDPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)
	localLabelPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()-]{0,63}$`)
)

// localModelConfig is one element of LOCAL_LLM_MODELS.
type localModelConfig struct {
	Endpoint     string   `json:"endpoint"`
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Capabilities []string `json:"capabilities"`
	Default      bool     `json:"default"`
}

// LocalParseResult is the outcome of ParseLocalModels.
type LocalParseResult struct {
	Entries           []ModelEntry
	Skipped           int
	DuplicateDefaults int
}

// LogFields returns counts only; never URLs, keys or model names.
func (r LocalParseResult) LogFields() map[string]interface{} {
	hasDefault := 0
	for i := range r.Entries {
		if r.Entries[i].Recommended {
			hasDefault = 1
		}
	}

	return map[string]interface{}{
		"local_models_loaded":            len(r.Entries),
		"local_models_skipped":           r.Skipped,
		"local_models_default":           hasDefault,
		"local_models_duplicate_default": r.DuplicateDefaults,
	}
}

// ParseLocalModels validates the LOCAL_LLM_MODELS value. Invalid entries are skipped
// and counted; a malformed or oversized value returns an error and no models.
func ParseLocalModels(raw string) (LocalParseResult, error) {
	res := LocalParseResult{}
	if len(raw) > maxLocalModelsInputBytes {
		return res, errors.New("local llm models value is too large")
	}

	if isBlank(raw) {
		return res, nil
	}

	var items []localModelConfig
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return res, errors.New("local llm models value is not a valid json list")
	}

	seen := map[string]bool{}
	defaultTaken := false
	for _, it := range items {
		if len(res.Entries) >= maxLocalModels {
			res.Skipped++
			continue
		}

		id := EngineModel(fmt.Sprintf("%s%s.%s", EngineModelPrefixLocal, it.Endpoint, it.ID))
		if !localEndpointPattern.MatchString(it.Endpoint) ||
			!localIDPattern.MatchString(it.ID) ||
			!localLabelPattern.MatchString(it.Label) ||
			len(id) > maxEngineModelLen ||
			seen[string(id)] {
			res.Skipped++
			continue
		}
		seen[string(id)] = true

		tags := []string{}
		for _, c := range it.Capabilities {
			if (c == TagToolCalling || c == TagLowCost) && !containsString(tags, c) {
				tags = append(tags, c)
			}
		}

		recommended := false
		if it.Default {
			if defaultTaken {
				res.DuplicateDefaults++
			} else {
				defaultTaken = true
				recommended = true
			}
		}

		res.Entries = append(res.Entries, ModelEntry{
			ID:          id,
			Label:       it.Label,
			Vendor:      localVendor,
			Route:       RouteLocal,
			Recommended: recommended,
			Tags:        tags,
			Description: localDescription,
		})
	}

	return res, nil
}

func isBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// localRegistry holds the immutable slice of local entries.
var localRegistry atomic.Pointer[[]ModelEntry]

// SetLocalModels replaces the registry with a deep copy of entries.
func SetLocalModels(entries []ModelEntry) {
	cp := make([]ModelEntry, len(entries))
	for i, e := range entries {
		e.Tags = append([]string{}, e.Tags...)
		cp[i] = e
	}
	localRegistry.Store(&cp)
}

// ConfigureLocalModels parses the LOCAL_LLM_MODELS value and stores the accepted
// models. On error the registry is emptied (fail closed) and the error is returned
// so the caller can log it; the service still starts.
func ConfigureLocalModels(raw string) (LocalParseResult, error) {
	res, err := ParseLocalModels(raw)
	if err != nil {
		SetLocalModels(nil)
		return res, err
	}

	SetLocalModels(res.Entries)
	return res, nil
}

func localModels() []ModelEntry {
	p := localRegistry.Load()
	if p == nil {
		return nil
	}
	return *p
}
