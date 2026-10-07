package builderhandler

// flow_option.go resolves a SymbolicNode's option map against its action
// type's ref-tagged fields (fmaction.RefFieldsOf), per VOIP-1573 design doc
// §3.2 steps 4-5. It never looks at the action type name to decide what to
// do with a field; it only reads RefKind.

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/actioncatalog"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// resolveOption clears every ref:"resource" field (regardless of what the
// LLM put there), then resolves every ref:"action" field's label(s) against
// labelToID, warning on anything it cannot resolve. Fields with no ref tag
// pass through untouched.
func resolveOption(n flowbuilder.SymbolicNode, labelToID map[string]uuid.UUID, removed map[string]bool) (map[string]any, []string) {
	opt := make(map[string]any, len(n.Option))
	for k, v := range n.Option {
		opt[k] = v
	}

	fields := fmaction.RefFieldsOf(fmaction.Type(n.Type))
	required := actioncatalog.RequiredFields(fmaction.Type(n.Type))

	var warnings []string

	// Step 4 (first half): clear every ref:"resource" field before
	// touching ref:"action" fields, so an LLM-supplied non-UUID string in
	// a resource field can never collide with the label resolution below
	// (design doc §3.2 step 4, round-6 fix).
	for _, f := range fields {
		if f.Kind != fmaction.RefKindResource {
			continue
		}
		delete(opt, f.JSONName) // omitempty -> decodes to the zero uuid.UUID (IDEmpty)
		if required[f.JSONName] {
			warnings = append(warnings, flowbuilder.WarningSelectResource+": "+n.Label+"."+f.JSONName)
		}
	}

	// Step 4 (second half) / step 5: resolve ref:"action" fields.
	for _, f := range fields {
		if f.Kind != fmaction.RefKindAction {
			continue
		}
		if f.IsMap {
			raw, ok := opt[f.JSONName].(map[string]any)
			if !ok {
				continue // absent or not a map: nothing to resolve
			}
			resolved := make(map[string]any, len(raw))
			for key, v := range raw {
				label, ok := v.(string)
				if !ok {
					continue
				}
				id, ok := labelToID[label]
				if !ok {
					if !removed[label] {
						warnings = append(warnings, flowbuilder.WarningInvalidLabelRef+": "+n.Label+"."+f.JSONName+"."+key)
					}
					continue // drop the key; branch falls through to its default
				}
				resolved[key] = id.String()
			}
			opt[f.JSONName] = resolved
			continue
		}

		raw, ok := opt[f.JSONName]
		if !ok || raw == nil {
			continue // absent: stays IDEmpty; §3.3 flags it if required
		}
		label, ok := raw.(string)
		if !ok {
			delete(opt, f.JSONName)
			continue
		}
		id, ok := labelToID[label]
		if !ok {
			delete(opt, f.JSONName)
			if !removed[label] {
				warnings = append(warnings, flowbuilder.WarningInvalidLabelRef+": "+n.Label+"."+f.JSONName)
			}
			continue
		}
		opt[f.JSONName] = id.String()
	}

	// Strict decode (design doc 3.2 step 4): drop keys the option struct
	// does not know or whose value has the wrong shape, and report each as
	// invalid_option. Done last so ref fields are already UUID strings.
	opt, w := dropInvalidOptionKeys(n, opt)
	warnings = append(warnings, w...)

	return opt, warnings
}

// dropInvalidOptionKeys keeps only the keys that decode into the action's
// real OptionXxx struct. It tests each key on its own so one bad key does
// not discard the others.
func dropInvalidOptionKeys(n flowbuilder.SymbolicNode, opt map[string]any) (map[string]any, []string) {
	optAny, ok := fmaction.OptionStructByType[fmaction.Type(n.Type)]
	if !ok {
		return opt, nil
	}
	keys := make([]string, 0, len(opt))
	for k := range opt {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]any, len(opt))
	var warnings []string
	for _, k := range keys {
		b, err := json.Marshal(map[string]any{k: opt[k]})
		if err == nil {
			dec := json.NewDecoder(bytes.NewReader(b))
			dec.DisallowUnknownFields()
			err = dec.Decode(reflect.New(reflect.TypeOf(optAny)).Interface())
		}
		if err != nil {
			warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+n.Label+"."+k)
			continue
		}
		out[k] = opt[k]
	}
	return out, warnings
}
