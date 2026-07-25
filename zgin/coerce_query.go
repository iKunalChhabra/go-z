package zgin

import "github.com/iKunalChhabra/go-z/z"

// CoerceQueryValues converts Gin query/form values (map[string][]string) into
// map[string]any suitable for schemas that use z.Coerce.* fields.
//
// Single values are unwrapped from []string to string; multi-values stay as
// []string (also assignable as []any element sources via BindQuery decode).
func CoerceQueryValues(values map[string][]string) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return coerceQueryValues(values, nil)
}

// CoerceQueryValuesFor is CoerceQueryValues with the target schema in hand: a key
// the schema declares as an array or set always becomes a slice, even when the
// query carries a single value.
//
// Without it, ?tag=a produces a string and ?tag=a&tag=b produces a slice, so a
// schema expecting an array rejects the single-value request — a trap that only
// shows up in production, on the request with one tag.
func CoerceQueryValuesFor(schema z.AnySchemaLike, values map[string][]string) map[string]any {
	shape, ok := z.ObjectShapeOf(schema)
	if !ok {
		return coerceQueryValues(values, nil)
	}
	sliceKeys := make(map[string]bool, len(shape))
	for key, field := range shape {
		if isSliceSchema(field) {
			sliceKeys[key] = true
		}
	}
	return coerceQueryValues(values, sliceKeys)
}

// isSliceSchema reports whether a field schema consumes a list, looking through
// wrappers such as Optional and Default.
func isSliceSchema(field z.AnySchemaLike) bool {
	for range 32 {
		if field == nil {
			return false
		}
		in := field.Internals()
		if in == nil || in.Def == nil {
			return false
		}
		switch in.Def.Type {
		case "array", "set", "tuple":
			return true
		}
		unwrapper, ok := field.(z.Unwrapper)
		if !ok {
			return false
		}
		field = unwrapper.Unwrap()
	}
	return false
}

func coerceQueryValues(values map[string][]string, sliceKeys map[string]bool) map[string]any {
	out := make(map[string]any, len(values))
	for k, vs := range values {
		if sliceKeys[k] {
			cp := make([]any, len(vs))
			for i := range vs {
				cp[i] = vs[i]
			}
			out[k] = cp
			continue
		}
		switch len(vs) {
		case 0:
			out[k] = ""
		case 1:
			out[k] = vs[0]
		default:
			// Keep multi-values as []string; schemas/array coercion see strings.
			cp := make([]string, len(vs))
			copy(cp, vs)
			out[k] = cp
		}
	}
	return out
}
