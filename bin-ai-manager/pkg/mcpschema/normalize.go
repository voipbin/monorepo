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
	// lookExpansions counts the R7a look-ahead's $ref expansions for the
	// whole tool. It has its own maxRefExpansions budget, so the look-ahead
	// neither eats into the build's budget nor fans out without bound.
	lookExpansions int

	// maxOut is the per-tool output cap; nodes counts emitted subschemas.
	// aborted is set once either limit is exceeded: the build stops and the
	// whole tool is dropped (R12).
	maxOut   int
	nodes    int
	workDone int
	aborted  bool
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

// work counts k units of transient work (map keys copied or scanned while
// resolving $ref/allOf and counting dropped keys) and aborts past maxWork.
// It bounds CPU and garbage on inputs the output charge does not see, such
// as long allOf chains whose merged keys are all dropped (R12).
func (n *normalizer) work(k int) {
	n.workDone += k
	if n.workDone > maxWork {
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

	// R7a: next to a type, an anyOf/oneOf whose members only add
	// constraints to that type (for example {"required": [...]} or
	// {"format": ...} variants) says nothing the kept subset can express.
	// It is removed, not evaluated, so it cannot make a usable subschema
	// unusable.
	if hasMembers && len(typ) > 0 && n.refines(members, path, depth, stack) {
		n.rep.DroppedKeys++
		members, hasMembers = nil, false
	}

	// R4: a list type becomes an anyOf of one member per type. An explicit
	// anyOf next to it is evaluated as a usability gate only (see below).
	if isList {
		n.rep.Rewrites++
		if hasMembers {
			if _, f := n.anyOf(members, path, depth, stack); f != nil {
				return nil, f
			}
			n.rep.DroppedKeys++
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
	// evaluated first as a usability gate (R7, R10). With no usable member
	// the subschema is unusable. No output ever carries type and anyOf
	// together (R10a): when the type is a scalar, the parent has no
	// constraint of its own, and every kept member has that same type (the
	// documented-enum shape, oneOf of {const, description}), the members
	// are emitted as the anyOf and the parent type is dropped; otherwise
	// the anyOf is discarded and only the type is emitted. A leaf member of
	// a scalar parent that names no type of its own takes the parent's
	// type, as JSON Schema applies it, so {const: 1} under an integer is an
	// integer member rather than an unusable typeless one. A member that
	// adds nothing but a description (a non-string const is dropped) keeps
	// the bare type, so values the kept subset cannot carry are not
	// advertised as empty alternatives.
	if hasMembers {
		if scalarTypes[ts] {
			members = n.inheritType(members, ts)
		}
		am, f := n.anyOf(members, path, depth, stack)
		if f != nil {
			return nil, f
		}
		if scalarTypes[ts] && !hasScalarConstraint(v) && allMembersOfType(am, ts) && allMembersConstrained(am) {
			delete(out, "type")
			out["anyOf"] = am
			return out, nil
		}
		n.rep.DroppedKeys++
	}

	switch ts {
	case typeString:
		if e, ok := n.stringEnum(n.enumOf(v, ts)); ok {
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
		n.work(len(l))
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
		if _, ok := n.stringEnum(v["enum"]); ok {
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
		n.work(len(t))
		res := make([]string, 0, len(t))
		seen := map[string]bool{}
		for _, x := range t {
			s, ok := x.(string)
			if !ok || !validTypes[s] {
				return nil, false, &failure{reason: ReasonBadType, path: path}
			}
			// R4: one anyOf member per distinct listed type.
			if seen[s] {
				continue
			}
			seen[s] = true
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
		if n.aborted {
			return nil, nil, errTooLarge
		}
		if ref, present := v["$ref"]; present {
			target, key, f := n.refTarget(ref, path, stack)
			if f != nil {
				return nil, nil, f
			}
			n.expansions++
			n.rep.Rewrites++
			stack = append(stack[:len(stack):len(stack)], key)
			v = mergeUnder(target, v, "$ref")
			n.work(len(v))
			continue
		}

		raw, present := v["allOf"]
		if !present {
			return v, stack, nil
		}
		l, ok := raw.([]any)
		if !ok || len(l) == 0 {
			v = without(v, "allOf")
			n.work(len(v))
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
		n.work(len(v))
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
		n.work(len(rl))
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
	n.work(len(v))
	for k := range v {
		if !keepKeys[k] && !consumedKeys[k] {
			n.rep.DroppedKeys++
		}
	}
}

// refines reports whether any anyOf/oneOf member only adds constraints to
// its parent (R7a). It is a look-ahead: it leaves the report and the build's
// $ref expansion count as they were, counts its own $ref expansions against
// a separate per-tool budget of maxRefExpansions, and is charged as work
// (R12). Once that budget is spent, a member behind a further $ref is not a
// refinement and is left to the normal anyOf evaluation.
func (n *normalizer) refines(members []member, path string, depth int, stack []string) bool {
	rewrites, dropped, expansions := n.rep.Rewrites, n.rep.DroppedKeys, n.expansions
	n.expansions = n.lookExpansions
	res := n.hasRefinementMember(members, path, depth, stack)
	n.lookExpansions = n.expansions
	n.rep.Rewrites, n.rep.DroppedKeys, n.expansions = rewrites, dropped, expansions
	return res
}

// hasRefinementMember reports whether any member is a refinement. A member
// is judged after its own $ref and single-member allOf are resolved, so a
// constraint-only member behind a $ref or an allOf wrapper counts too; a
// member that is itself only a combinator is a refinement when any of its
// own members is. A member that cannot be resolved is not a refinement: it
// is left to the normal anyOf evaluation, which removes it. Call it through
// refines.
func (n *normalizer) hasRefinementMember(members []member, path string, depth int, stack []string) bool {
	for _, m := range members {
		if n.isRefinement(m.raw, path+m.path, depth+1, stack) {
			return true
		}
	}
	return false
}

// isRefinement reports whether raw, once resolved, has no shape of its own.
func (n *normalizer) isRefinement(raw any, path string, depth int, stack []string) bool {
	if n.aborted || depth > maxDepth {
		return false
	}
	n.work(1)
	mm, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	v, stack, f := n.resolve(mm, path, stack)
	if f != nil {
		return false
	}
	for _, k := range shapeKeys {
		if _, ok := v[k]; ok {
			return false
		}
	}
	nested, has := n.anyOfMembers(v)
	if !has {
		return true
	}
	return n.hasRefinementMember(nested, path, depth, stack)
}

// shapeKeys are the keys that give a resolved anyOf member a shape of its
// own. $ref and allOf are resolved before the check; anyOf/oneOf are
// followed into their members.
var shapeKeys = []string{"type", "properties", "items", "enum", "const"}

// scalarTypes are the types whose same-typed members may replace the
// parent type (R10a).
var scalarTypes = map[string]bool{typeString: true, typeNumber: true, typeInteger: true, typeBoolean: true}

// hasScalarConstraint reports whether a scalar subschema carries its own
// constraint that its anyOf members would not repeat.
func hasScalarConstraint(v map[string]any) bool {
	for _, k := range []string{"enum", "const", "format", "minimum", "maximum"} {
		if _, ok := v[k]; ok {
			return true
		}
	}
	return false
}

// inheritKeys are the keys that stop a member from taking its scalar
// parent's type: it has a type or shape of its own, or its type comes from
// a $ref, an allOf or its own members.
var inheritKeys = []string{"type", "properties", "items", "$ref", "allOf", "anyOf", "oneOf"}

// inheritType returns members with each leaf member that has none of
// inheritKeys, and whose const (if any) is a value of type t, copied with
// type t added. Each copy is a rewrite and charged as work (R12). A member
// whose const is not of type t cannot match the parent and is left as it
// is, so it stays unusable.
func (n *normalizer) inheritType(members []member, t string) []member {
	res := make([]member, 0, len(members))
	for _, m := range members {
		mm, ok := m.raw.(map[string]any)
		if ok && !hasAnyKey(mm, inheritKeys) && constOfType(mm, t) {
			c := make(map[string]any, len(mm)+1)
			for k, x := range mm {
				c[k] = x
			}
			c["type"] = t
			n.work(len(c))
			n.rep.Rewrites++
			m = member{raw: c, path: m.path}
		}
		res = append(res, m)
	}
	return res
}

// hasAnyKey reports whether m has any of keys.
func hasAnyKey(m map[string]any, keys []string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// constOfType reports whether m has no const, or a const that is a JSON
// value of scalar type t.
func constOfType(m map[string]any, t string) bool {
	c, present := m["const"]
	if !present {
		return true
	}
	switch x := c.(type) {
	case string:
		return t == typeString
	case bool:
		return t == typeBoolean
	case float64:
		return t == typeNumber || (t == typeInteger && x == math.Trunc(x))
	}
	return false
}

// allMembersConstrained reports whether every normalized member carries a
// value constraint of its own (enum, format, minimum or maximum), so that
// emitting the members says more than the bare type.
func allMembersConstrained(members []any) bool {
	for _, m := range members {
		mm, ok := m.(map[string]any)
		if !ok || !hasAnyKey(mm, []string{"enum", "format", "minimum", "maximum"}) {
			return false
		}
	}
	return true
}

// allMembersOfType reports whether every normalized member has type t.
func allMembersOfType(members []any, t string) bool {
	for _, m := range members {
		mm, ok := m.(map[string]any)
		if !ok || mm["type"] != t {
			return false
		}
	}
	return true
}

// stringEnum returns a copy of e when it is a non-empty list of strings. The
// scan is charged as work (R12) since it runs before the enum is accepted.
func (n *normalizer) stringEnum(e any) ([]any, bool) {
	l, ok := e.([]any)
	if !ok || len(l) == 0 {
		return nil, false
	}
	n.work(len(l))
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
