package mcpschema

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// failure says why a subschema is unusable and where.
type failure struct {
	reason string
	path   string
}

// normalizer holds the state of one Normalize call.
type normalizer struct {
	rep *Report

	// defs holds the root's $defs and definitions maps, by keyword.
	defs map[string]map[string]any
	// expansions counts $ref expansions for the whole tool.
	expansions int

	// maxOut is the per-tool output cap; nodes counts emitted subschemas.
	// aborted is set once either limit is exceeded: the build stops and the
	// whole tool is dropped (R12).
	maxOut  int
	nodes   int
	aborted bool
}

// errTooLarge is the failure every call returns once the build is aborted.
var errTooLarge = &failure{reason: ReasonTooLarge}

// charge adds k bytes to the output charge and aborts past maxOut.
func (n *normalizer) charge(k int) {
	n.rep.OutBytes += k
	if n.rep.OutBytes > n.maxOut {
		n.aborted = true
	}
}

// chargeStr charges one emitted string: its UTF-8 length plus 3.
func (n *normalizer) chargeStr(s string) {
	n.charge(len(s) + 3)
}

// emitted counts one emitted subschema and aborts past maxNodes.
func (n *normalizer) emitted() {
	n.nodes++
	if n.nodes > maxNodes {
		n.aborted = true
	}
}

// Normalize returns a provider-neutral copy of schema (the decoded top-level
// inputSchema object). It never mutates its input. A nil schema returns (nil,
// Report{}). maxOutBytes is the per-tool output cap; the caller passes the
// same cap the raw input has.
func Normalize(schema map[string]any, maxOutBytes int) (map[string]any, Report) {
	var rep Report
	if schema == nil {
		return nil, rep
	}
	n := &normalizer{rep: &rep, defs: map[string]map[string]any{}, maxOut: maxOutBytes}
	for _, k := range []string{"$defs", "definitions"} {
		if m, ok := schema[k].(map[string]any); ok {
			n.defs[k] = m
		}
	}

	out, f := n.root(schema)
	if n.aborted {
		f = errTooLarge
	}
	if f != nil {
		rep.ToolDropped = true
		rep.DropReason = f.reason
		rep.DropPath = f.path
		return nil, rep
	}
	return out, rep
}

// root normalizes the top-level parameters object (R0). Its anyOf/oneOf are
// dropped, an absent type is object, and properties is always emitted.
func (n *normalizer) root(s map[string]any) (map[string]any, *failure) {
	n.charge(visitBytes)
	v, stack, f := n.resolve(s, "", nil)
	if f != nil {
		return nil, f
	}

	t, hasType := v["type"]
	if !hasType {
		t = typeObject
	}
	// R4: a single-element list is the plain type.
	if l, ok := t.([]any); ok && len(l) == 1 {
		t = l[0]
	}
	if ts, ok := t.(string); !ok || ts != typeObject {
		return nil, &failure{reason: ReasonRootNotObject, path: ""}
	}

	n.countDropped(v)
	for _, k := range []string{"anyOf", "oneOf", "const"} {
		if _, ok := v[k]; ok {
			n.rep.DroppedKeys++
		}
	}

	out := map[string]any{"type": typeObject}
	n.chargeStr(typeObject)
	n.setDescription(out, v)

	props, req, f := n.objectBody(v, "", 0, stack)
	if f != nil {
		return nil, f
	}
	out["properties"] = props
	if len(req) > 0 {
		out["required"] = req
	}
	n.emitted()
	return out, nil
}

// node normalizes one non-root subschema at path and depth. A failure means
// the subschema is unusable; the caller applies the cascade (R7).
func (n *normalizer) node(raw any, path string, depth int, stack []string) (map[string]any, *failure) {
	if n.aborted {
		return nil, errTooLarge
	}
	n.charge(visitBytes)
	out, f := n.build(raw, path, depth, stack)
	if f != nil {
		return nil, f
	}
	n.emitted()
	return out, nil
}

// build does node's work for one visited subschema.
func (n *normalizer) build(raw any, path string, depth int, stack []string) (map[string]any, *failure) {
	if depth > maxDepth {
		return nil, &failure{reason: ReasonTooDeep, path: path}
	}
	s, ok := raw.(map[string]any)
	if !ok {
		return nil, &failure{reason: ReasonBadShape, path: path}
	}

	v, stack, f := n.resolve(s, path, stack)
	if f != nil {
		return nil, f
	}
	n.countDropped(v)

	members, hasMembers := n.anyOfMembers(v)
	typ, isList, f := n.typeOf(v, path, hasMembers)
	if f != nil {
		return nil, f
	}

	// R4: a list type becomes an anyOf of one member per type. An explicit
	// anyOf next to it is evaluated as a usability gate only (see below).
	if isList {
		n.rep.Rewrites++
		if hasMembers {
			if _, f := n.anyOf(members, path, depth, stack); f != nil {
				return nil, f
			}
		}
		out := map[string]any{}
		n.setDescription(out, v)
		listMembers := make([]member, 0, len(typ))
		for i, t := range typ {
			m := map[string]any{"type": t}
			if t != typeNull {
				for _, k := range siblingKeys {
					if x, ok := v[k]; ok {
						m[k] = x
					}
				}
			}
			listMembers = append(listMembers, member{raw: m, path: "/anyOf/" + strconv.Itoa(i)})
		}
		am, f := n.anyOf(listMembers, path, depth, stack)
		if f != nil {
			return nil, f
		}
		out["anyOf"] = am
		return out, nil
	}

	out := map[string]any{}
	n.setDescription(out, v)

	// No type: the subschema is its anyOf (R6).
	if len(typ) == 0 {
		am, f := n.anyOf(members, path, depth, stack)
		if f != nil {
			return nil, f
		}
		out["anyOf"] = am
		return out, nil
	}

	ts := typ[0]
	out["type"] = ts
	n.chargeStr(ts)

	// A typed subschema that also carries anyOf/oneOf: the anyOf is
	// evaluated first (R7, R10). With no usable member the subschema is
	// unusable; otherwise it is kept next to the type (R1), except on an
	// object left without properties, which is free-form (R10, below).
	if hasMembers {
		am, f := n.anyOf(members, path, depth, stack)
		if f != nil {
			return nil, f
		}
		out["anyOf"] = am
	}

	switch ts {
	case typeString:
		if e, ok := stringEnum(n.enumOf(v, ts)); ok {
			out["enum"] = e
			for _, x := range e {
				n.chargeStr(x.(string))
			}
		}
	case typeNumber, typeInteger:
		for _, k := range []string{"minimum", "maximum"} {
			if x, ok := v[k].(float64); ok {
				out[k] = x
				n.charge(numberBytes)
			}
		}
	case typeArray:
		items, present := v["items"]
		if !present {
			return nil, &failure{reason: ReasonArrayItems, path: path}
		}
		if _, isObj := items.(map[string]any); !isObj {
			return nil, &failure{reason: ReasonArrayItems, path: path}
		}
		it, f := n.node(items, path+"/items", depth+1, stack)
		if f != nil {
			return nil, f
		}
		out["items"] = it
		for _, k := range []string{"minItems", "maxItems"} {
			if x, ok := v[k].(float64); ok && x >= 0 && x == math.Trunc(x) {
				out[k] = x
				n.charge(numberBytes)
			}
		}
	case typeObject:
		props, req, f := n.objectBody(v, path, depth, stack)
		if f != nil {
			return nil, f
		}
		// R10: a nested object with no usable properties is free-form,
		// exactly {type: object, description?}.
		if len(props) == 0 {
			delete(out, "anyOf")
			return out, nil
		}
		out["properties"] = props
		if len(req) > 0 {
			out["required"] = req
		}
	}

	if fm, ok := v["format"].(string); ok && formatAllowed(ts, fm, out["enum"] != nil) {
		out["format"] = fm
		n.chargeStr(fm)
	}
	return out, nil
}

// siblingKeys are the constraints a list-type member inherits (R4); each
// member then keeps only what is valid for its own type.
var siblingKeys = []string{"const", "enum", "format", "items", "properties", "required", "minimum", "maximum", "minItems", "maxItems"}

// member is one anyOf candidate with the path it came from.
type member struct {
	raw  any
	path string
}

// anyOfMembers collects anyOf then oneOf members (R5). hasMembers is true
// when either key holds a list, even an empty one.
func (n *normalizer) anyOfMembers(v map[string]any) ([]member, bool) {
	var res []member
	has := false
	for _, k := range []string{"anyOf", "oneOf"} {
		raw, present := v[k]
		if !present {
			continue
		}
		l, ok := raw.([]any)
		if !ok {
			n.rep.DroppedKeys++
			continue
		}
		has = true
		if k == "oneOf" {
			n.rep.Rewrites++
		}
		for i, m := range l {
			res = append(res, member{raw: m, path: "/" + k + "/" + strconv.Itoa(i)})
		}
	}
	return res, has
}

// anyOf normalizes members one level deeper, removing unusable ones (R7).
// It fails when no member is left, or only {type: null} members are.
func (n *normalizer) anyOf(members []member, path string, depth int, stack []string) ([]any, *failure) {
	out := make([]any, 0, len(members))
	nonNull := 0
	for _, m := range members {
		nm, f := n.node(m.raw, path+m.path, depth+1, stack)
		if n.aborted {
			return nil, errTooLarge
		}
		if f != nil {
			continue
		}
		if nm["type"] != typeNull {
			nonNull++
		}
		out = append(out, nm)
	}
	if nonNull == 0 {
		return nil, &failure{reason: ReasonAnyOf, path: path}
	}
	return out, nil
}

// typeOf returns the subschema's type list after R5 const and R6 inference.
// An empty result with no failure means "no type, the anyOf describes it".
// isList is true for a list type of two or more entries (R4).
func (n *normalizer) typeOf(v map[string]any, path string, hasMembers bool) ([]string, bool, *failure) {
	raw, hasType := v["type"]
	if !hasType {
		if isStringConst(v) {
			return []string{typeString}, false, nil
		}
		if _, ok := v["properties"].(map[string]any); ok {
			n.rep.Rewrites++
			return []string{typeObject}, false, nil
		}
		if _, ok := v["items"]; ok {
			n.rep.Rewrites++
			return []string{typeArray}, false, nil
		}
		if _, ok := stringEnum(v["enum"]); ok {
			n.rep.Rewrites++
			return []string{typeString}, false, nil
		}
		if hasMembers {
			return nil, false, nil
		}
		return nil, false, &failure{reason: ReasonTypeless, path: path}
	}

	switch t := raw.(type) {
	case string:
		if !validTypes[t] {
			return nil, false, &failure{reason: ReasonBadType, path: path}
		}
		return []string{t}, false, nil
	case []any:
		if len(t) == 0 {
			return nil, false, &failure{reason: ReasonBadType, path: path}
		}
		res := make([]string, 0, len(t))
		for _, x := range t {
			s, ok := x.(string)
			if !ok || !validTypes[s] {
				return nil, false, &failure{reason: ReasonBadType, path: path}
			}
			res = append(res, s)
		}
		return res, len(res) > 1, nil
	}
	return nil, false, &failure{reason: ReasonBadType, path: path}
}

// isStringConst reports whether v carries a string const.
func isStringConst(v map[string]any) bool {
	_, ok := v["const"].(string)
	return ok
}

// enumOf returns the enum for a string subschema: a string const wins over
// enum (R5), anything else is the raw enum.
func (n *normalizer) enumOf(v map[string]any, t string) any {
	if c, ok := v["const"].(string); ok && t == typeString {
		n.rep.Rewrites++
		return []any{c}
	}
	return v["enum"]
}

// resolve applies R5's $ref inlining and single-member allOf merge until
// neither is left. The result may share maps with the input but is never
// written to. stack is the chain of ref targets expanded on the path to
// here; the returned stack includes the ones expanded now.
func (n *normalizer) resolve(s map[string]any, path string, stack []string) (map[string]any, []string, *failure) {
	v := s
	for {
		if ref, present := v["$ref"]; present {
			target, key, f := n.refTarget(ref, path, stack)
			if f != nil {
				return nil, nil, f
			}
			n.expansions++
			n.rep.Rewrites++
			stack = append(stack[:len(stack):len(stack)], key)
			v = mergeUnder(target, v, "$ref")
			continue
		}

		raw, present := v["allOf"]
		if !present {
			return v, stack, nil
		}
		l, ok := raw.([]any)
		if !ok || len(l) == 0 {
			v = without(v, "allOf")
			continue
		}
		if len(l) > 1 {
			return nil, nil, &failure{reason: ReasonAllOf, path: path}
		}
		m, ok := l[0].(map[string]any)
		if !ok {
			return nil, nil, &failure{reason: ReasonBadShape, path: path}
		}
		n.rep.Rewrites++
		v = mergeUnder(m, v, "allOf")
	}
}

// refTarget looks up a local $ref (#/$defs/<name> or #/definitions/<name>)
// and enforces the cycle, nesting and per-tool expansion limits. key
// identifies the target on the expansion stack.
func (n *normalizer) refTarget(ref any, path string, stack []string) (map[string]any, string, *failure) {
	fail := &failure{reason: ReasonRef, path: path}
	rs, ok := ref.(string)
	if !ok {
		return nil, "", fail
	}
	var kw, name string
	for _, k := range []string{"$defs", "definitions"} {
		if rest, found := strings.CutPrefix(rs, "#/"+k+"/"); found {
			kw, name = k, rest
			break
		}
	}
	if kw == "" || name == "" || strings.Contains(name, "/") {
		return nil, "", fail
	}
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")

	target, ok := n.defs[kw][name].(map[string]any)
	if !ok {
		return nil, "", fail
	}
	key := kw + "/" + name
	for _, k := range stack {
		if k == key {
			return nil, "", fail
		}
	}
	if len(stack) >= maxRefDepth || n.expansions >= maxRefExpansions {
		return nil, "", fail
	}
	return target, key, nil
}

// mergeUnder returns base overlaid with over's keys except skip; over wins
// on conflict. Neither input is modified.
func mergeUnder(base, over map[string]any, skip string) map[string]any {
	res := make(map[string]any, len(base)+len(over))
	for k, x := range base {
		res[k] = x
	}
	delete(res, skip)
	for k, x := range over {
		if k != skip {
			res[k] = x
		}
	}
	// A key named skip in base (a nested $ref or allOf) is resolved next.
	if x, ok := base[skip]; ok {
		res[skip] = x
	}
	return res
}

// without returns a copy of v without key k.
func without(v map[string]any, k string) map[string]any {
	res := make(map[string]any, len(v))
	for kk, x := range v {
		if kk != k {
			res[kk] = x
		}
	}
	return res
}

// objectBody normalizes an object's properties and required list, applying
// the unusable-property cascade (R7) and required pruning (R8).
func (n *normalizer) objectBody(v map[string]any, path string, depth int, stack []string) (map[string]any, []any, *failure) {
	required := map[string]bool{}
	reqOrder := []string{}
	if rl, ok := v["required"].([]any); ok {
		for _, r := range rl {
			rs, ok := r.(string)
			if !ok || required[rs] {
				continue
			}
			required[rs] = true
			reqOrder = append(reqOrder, rs)
		}
	}

	props := map[string]any{}
	if pm, ok := v["properties"].(map[string]any); ok {
		for _, name := range sortedKeys(pm) {
			childPath := path + "/properties/" + escapePointer(name)
			child, f := n.node(pm[name], childPath, depth+1, stack)
			if n.aborted {
				return nil, nil, errTooLarge
			}
			if f != nil {
				if required[name] {
					return nil, nil, f
				}
				n.dropProp(childPath)
				continue
			}
			props[name] = child
			n.chargeStr(name)
		}
	}

	req := []any{}
	for _, r := range reqOrder {
		if _, ok := props[r]; ok {
			req = append(req, r)
			n.chargeStr(r)
		}
	}
	return props, req, nil
}

// setDescription copies a string description from v into out.
func (n *normalizer) setDescription(out, v map[string]any) {
	if d, ok := v["description"].(string); ok {
		out["description"] = d
		n.chargeStr(d)
	}
}

// dropProp records an optional property removed by the cascade.
func (n *normalizer) dropProp(path string) {
	n.rep.DroppedPropsN++
	if len(n.rep.DroppedProps) < maxReportedPaths {
		n.rep.DroppedProps = append(n.rep.DroppedProps, path)
	}
}

// countDropped counts the keys of v that are neither emitted nor consumed.
func (n *normalizer) countDropped(v map[string]any) {
	for k := range v {
		if !keepKeys[k] && !consumedKeys[k] {
			n.rep.DroppedKeys++
		}
	}
}

// stringEnum returns a copy of e when it is a non-empty list of strings.
func stringEnum(e any) ([]any, bool) {
	l, ok := e.([]any)
	if !ok || len(l) == 0 {
		return nil, false
	}
	out := make([]any, 0, len(l))
	for _, x := range l {
		s, ok := x.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// formatAllowed reports whether format fm may be emitted for type t (R3).
func formatAllowed(t, fm string, hasEnum bool) bool {
	switch t {
	case typeString:
		return fm == "date-time" || (fm == "enum" && hasEnum)
	case typeInteger:
		return fm == "int32" || fm == "int64"
	case typeNumber:
		return fm == "float" || fm == "double"
	}
	return false
}

// escapePointer escapes a property name as a JSON pointer segment.
func escapePointer(s string) string {
	if !strings.ContainsAny(s, "~/") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

// sortedKeys returns m's keys in ascending order (R13).
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
