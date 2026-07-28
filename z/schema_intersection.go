package z

import (
	"reflect"
	"slices"
	"time"
)

// IntersectionSchema is the schema.
type IntersectionSchema struct {
	schemaBase[any]
	def   *Def
	Left  AnySchemaLike
	Right AnySchemaLike
}

// Intersection returns a schema that parses left and right, then deep-merges
// their outputs (z.intersection(a, b)).
func Intersection(left, right AnySchemaLike, params ...any) *IntersectionSchema {
	p := normalizeParams(params)
	def := &Def{Type: "intersection", Error: p.Error}
	return newIntersection(def, left, right)
}

func newIntersection(def *Def, left, right AnySchemaLike) *IntersectionSchema {
	s := &IntersectionSchema{def: def, Left: left, Right: right}
	parse := makeIntersectionParse(def, left, right)
	in := buildInternals(def, parse)
	// OptIn/OptOut: either side optional ⇒ intersection optional.
	lin, rin := left.Internals(), right.Internals()
	in.OptIn = lin.OptIn || rin.OptIn
	in.OptOut = lin.OptOut || rin.OptOut
	s.schemaBase = newBase[any](in)
	return s
}

func makeIntersectionParse(def *Def, left, right AnySchemaLike) ParseFn {
	// When both sides are plain objects the intersection's recognized shape is
	// the union of theirs, resolved once here.
	recognized := intersectionShapeUnion(left, right)
	return func(p *Payload, ctx *ParseCtx) {
		lp := AcquirePayload(p.Value)
		rp := AcquirePayload(p.Value)
		left.Internals().Run(lp, ctx)
		right.Internals().Run(rp, ctx)
		handleIntersectionResults(p, lp, rp, def, recognized)
		ReleasePayload(lp)
		ReleasePayload(rp)
	}
}

// intersectionShapeUnion returns the union of both sides' keys when both sides
// are object schemas, and nil otherwise. A key inside the union is recognized by
// the intersection even though one side alone does not know it, which is what
// makes intersecting two strict objects usable.
func intersectionShapeUnion(left, right AnySchemaLike) map[string]struct{} {
	lo, lok := left.(*ObjectSchema)
	ro, rok := right.(*ObjectSchema)
	if !lok || !rok {
		return nil
	}
	union := make(map[string]struct{}, len(lo.shape)+len(ro.shape))
	for k := range lo.shape {
		union[k] = struct{}{}
	}
	for k := range ro.shape {
		union[k] = struct{}{}
	}
	return union
}

// handleIntersectionResults merges both sides' issues and values.
//
// Unrecognized keys need care. Each side only knows its own shape, so a strict
// object inside an intersection reports the other side's keys as unrecognized
// even though the intersection accepts them. When both sides are objects, the
// intersection's recognized set is the union of their shapes and only keys
// outside that union are reported — so `Intersection(Strict{a}, Loose{b})` still
// rejects an unknown `c`. When a side is not a plain object its shape is unknown,
// and the fallback is to report only keys both sides flagged.
func handleIntersectionResults(result, left, right *Payload, def *Def, recognized map[string]struct{}) {
	type flags struct{ l, r bool }
	unrecKeys := map[string]*flags{}
	var unrecIssue *Issue

	for i := range left.Issues {
		iss := left.Issues[i]
		if iss.Code == IssueUnrecognizedKeys {
			cp := iss
			unrecIssue = &cp
			for _, k := range iss.Keys {
				f := unrecKeys[k]
				if f == nil {
					f = &flags{}
					unrecKeys[k] = f
				}
				f.l = true
			}
			continue
		}
		result.Issues = append(result.Issues, iss)
	}
	for i := range right.Issues {
		iss := right.Issues[i]
		if iss.Code == IssueUnrecognizedKeys {
			if unrecIssue == nil {
				cp := iss
				unrecIssue = &cp
			}
			for _, k := range iss.Keys {
				f := unrecKeys[k]
				if f == nil {
					f = &flags{}
					unrecKeys[k] = f
				}
				f.r = true
			}
			continue
		}
		result.Issues = append(result.Issues, iss)
	}

	report := make([]string, 0, len(unrecKeys))
	for k, f := range unrecKeys {
		if recognized != nil {
			if _, ok := recognized[k]; !ok {
				report = append(report, k)
			}
			continue
		}
		if f.l && f.r {
			report = append(report, k)
		}
	}
	if len(report) > 0 && unrecIssue != nil {
		slices.Sort(report)
		iss := *unrecIssue
		iss.Keys = report
		result.Issues = append(result.Issues, iss)
	}

	if result.aborted(0) {
		return
	}

	merged, mergePath, ok := mergeValues(left.Value, right.Value)
	if !ok {
		// throws Error; Go prefers a custom issue so SafeParse stays safe.
		path := make([]any, len(mergePath))
		for i, seg := range mergePath {
			path[i] = seg
		}
		result.AddIssue(Issue{
			Code:    IssueCustom,
			Message: "Unmergeable intersection results",
			Path:    path,
			Input:   result.Value,
			errMap:  def.Error,
		})
		return
	}
	result.Value = merged
}

// mergeValues ports mergeValues: objects merge keys, equal-length arrays
// merge elementwise, identical primitives (incl. equal times) pass through.
// The []any path is the merge-error path (mergeErrorPath).
func mergeValues(a, b any) (any, []any, bool) {
	//: if (a === b) — reference/primitive equality. Avoid Go == on
	// maps/slices (they panic when compared with ==).
	if sameIntersectionRef(a, b) {
		return a, nil, true
	}
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok && ta.Equal(tb) {
			return a, nil, true
		}
		return nil, nil, false
	}

	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		newObj := make(map[string]any, len(aMap)+len(bMap))
		for k, v := range aMap {
			newObj[k] = v
		}
		for k, v := range bMap {
			newObj[k] = v
		}
		for k := range aMap {
			bv, ok := bMap[k]
			if !ok {
				continue
			}
			shared, subPath, valid := mergeValues(aMap[k], bv)
			if !valid {
				return nil, append([]any{k}, subPath...), false
			}
			newObj[k] = shared
		}
		return newObj, nil, true
	}

	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return nil, nil, false
		}
		out := make([]any, len(aSlice))
		for i := range aSlice {
			shared, subPath, valid := mergeValues(aSlice[i], bSlice[i])
			if !valid {
				return nil, append([]any{i}, subPath...), false
			}
			out[i] = shared
		}
		return out, nil, true
	}

	return nil, nil, false
}

// sameIntersectionRef reports === for values safe to compare in Go.
func sameIntersectionRef(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return reflect.ValueOf(av).Pointer() == reflect.ValueOf(bv).Pointer()
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		// Two nil slices compare equal; otherwise compare data pointers.
		if av == nil && bv == nil {
			return true
		}
		return reflect.ValueOf(av).Pointer() == reflect.ValueOf(bv).Pointer()
	default:
		return a == b
	}
}

// Check attaches raw checks (immutable clone).
func (s *IntersectionSchema) Check(checks ...*Check) *IntersectionSchema {
	return newIntersection(s.def.withChecks(checks...), s.Left, s.Right)
}
