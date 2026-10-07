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
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/actioncatalog"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// resolveOption clears every ref:"resource" field (regardless of what the
// LLM put there), then resolves every ref:"action" field's label(s) against
// labelToID, warning on anything it cannot resolve. Fields with no ref tag
// pass through untouched.
func resolveOption(n flowbuilder.SymbolicNode, labelToID map[string]uuid.UUID, removed map[string]bool) (map[string]any, []string) {
	// encoding/json matches keys case-insensitively, and flow-manager decodes
	// an action's option with it, so "Queue_ID" would fill queue_id there.
	// Every later step matches keys exactly, so keep only exact keys first:
	// a variant could otherwise carry a resource id or an unresolved action
	// reference past the clearing and resolution below.
	opt, droppedKeys := exactOptionKeys(fmaction.Type(n.Type), n.Option)

	fields := fmaction.RefFieldsOf(fmaction.Type(n.Type))
	required := actioncatalog.RequiredFields(fmaction.Type(n.Type))

	var warnings []string
	for _, k := range droppedKeys {
		warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+n.Label+"."+shortKey(k))
	}

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
			keys := make([]string, 0, len(raw))
			for key := range raw {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				v := raw[key]
				// A key is the value to match (a digit, a word). Only an exact
				// uuid-shaped key is dropped: this keeps an id out of the usual
				// place, it does not promise that no key text can hold one.
				if _, err := uuid.FromString(key); err == nil {
					warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+n.Label+"."+f.JSONName)
					continue
				}
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

	// ref:"address" fields: an address whose type is a platform resource
	// carries that resource's id in target, so the target is cleared and the
	// user picks it in the editor; a phone number, SIP URI or email address
	// is the user's own value and stays.
	for _, f := range fields {
		if f.Kind != fmaction.RefKindAddress {
			continue
		}
		var w []string
		opt, w = sanitizeAddressField(opt, f, n.Label, true)
		warnings = append(warnings, w...)
	}

	// Strict decode (design doc 3.2 step 4): drop keys the option struct
	// does not know or whose value has the wrong shape, and report each as
	// invalid_option. Done last so ref fields are already UUID strings.
	opt, w := dropInvalidOptionKeys(n, opt)
	warnings = append(warnings, w...)

	// The draft must fit the limit the next request is validated against.
	// Substituting a label with a 36 character id can grow an option past it.
	opt, w = fitOptionSize(n.Label, opt, fields)
	warnings = append(warnings, w...)

	return opt, warnings
}

// addressKeys are the exact json names of commonaddress.Address.
var addressKeys = map[string]bool{"type": true, "target": true, "target_name": true, "name": true, "detail": true}

// sanitizeAddressField rewrites one address (or list of addresses) field:
// keys that are not an exact Address json name are dropped (encoding/json
// would otherwise match "Target" case-insensitively), and the target of an
// address that is not an external endpoint is cleared. report is false when
// the caller only wants the sanitising (the model view), true to also get
// invalid_option and select_resource warnings.
func sanitizeAddressField(opt map[string]any, f fmaction.RefField, label string, report bool) (map[string]any, []string) {
	raw, ok := opt[f.JSONName]
	if !ok || raw == nil {
		return opt, nil
	}

	var warnings []string
	clean := func(item any, where string) (any, bool) {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false // wrong shape: dropped, the strict decode would reject it anyway
		}
		out := make(map[string]any, len(obj))
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !addressKeys[k] {
				if report {
					warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+where+"."+shortKey(k))
				}
				continue
			}
			out[k] = obj[k]
		}
		typ, _ := out["type"].(string)
		if !commonaddress.IsExternalEndpoint(commonaddress.Type(typ)) {
			if _, has := out["target"]; has {
				delete(out, "target")
				if report {
					warnings = append(warnings, flowbuilder.WarningSelectResource+": "+where)
				}
			}
		}
		return out, true
	}

	if f.IsList {
		list, ok := raw.([]any)
		if !ok {
			delete(opt, f.JSONName)
			return opt, nil
		}
		out := make([]any, 0, len(list))
		for i, item := range list {
			if c, ok := clean(item, label+"."+f.JSONName+"["+strconv.Itoa(i)+"]"); ok {
				out = append(out, c)
			}
		}
		opt[f.JSONName] = out
		return opt, warnings
	}

	if c, ok := clean(raw, label+"."+f.JSONName); ok {
		opt[f.JSONName] = c
	} else {
		delete(opt, f.JSONName)
	}
	return opt, warnings
}

// fitOptionSize keeps an option under flowbuilder.MaxOptionBytes once
// serialized. It first drops entries of map ref fields (the part label
// substitution grows), in key order, then, if that is not enough, the whole
// option. Each cut is reported as invalid_option.
func fitOptionSize(label string, opt map[string]any, fields []fmaction.RefField) (map[string]any, []string) {
	size := func() int {
		b, err := json.Marshal(opt)
		if err != nil {
			return flowbuilder.MaxOptionBytes + 1
		}
		return len(b)
	}
	if size() <= flowbuilder.MaxOptionBytes {
		return opt, nil
	}

	var warnings []string
	for _, f := range fields {
		if !f.IsMap || f.Kind != fmaction.RefKindAction {
			continue
		}
		m, ok := opt[f.JSONName].(map[string]any)
		if !ok {
			continue
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		for _, k := range keys {
			if size() <= flowbuilder.MaxOptionBytes {
				return opt, warnings
			}
			delete(m, k)
			warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+label+"."+f.JSONName)
		}
	}
	if size() > flowbuilder.MaxOptionBytes {
		warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+label)
		return map[string]any{}, warnings
	}
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
			warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+n.Label+"."+shortKey(k))
			continue
		}
		out[k] = opt[k]
	}
	return out, warnings
}

// optionFieldNames returns the exact top-level json names of the action's
// option struct, and false when the type has no known option struct.
func optionFieldNames(t fmaction.Type) (map[string]bool, bool) {
	optAny, ok := fmaction.OptionStructByType[t]
	if !ok {
		return nil, false
	}
	rt := reflect.TypeOf(optAny)
	if rt.Kind() != reflect.Struct {
		return map[string]bool{}, true
	}
	names := make(map[string]bool, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		name := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			names[name] = true
		}
	}
	return names, true
}

// exactOptionKeys returns a copy of opt with only the keys that equal a json
// field name of the action's option struct exactly (case included), plus the
// sorted list of keys it dropped. A type with no known option struct is
// returned unchanged: the type filter removes such a node anyway.
func exactOptionKeys(t fmaction.Type, opt map[string]any) (map[string]any, []string) {
	names, known := optionFieldNames(t)
	out := make(map[string]any, len(opt))
	var dropped []string
	for k, v := range opt {
		if known && !names[k] {
			dropped = append(dropped, k)
			continue
		}
		out[k] = v
	}
	sort.Strings(dropped)
	return out, dropped
}

// shortKey bounds a model-supplied key echoed in a warning, so one long key
// cannot grow the response.
func shortKey(k string) string {
	r := []rune(k)
	if len(r) > 40 {
		return string(r[:40]) + "..."
	}
	return k
}
