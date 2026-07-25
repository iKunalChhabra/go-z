package zgin

// CoerceQueryValues converts Gin query/form values (map[string][]string) into
// map[string]any suitable for schemas that use zod.Coerce.* fields.
//
// Single values are unwrapped from []string to string; multi-values stay as
// []string (also assignable as []any element sources via BindQuery decode).
func CoerceQueryValues(values map[string][]string) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for k, vs := range values {
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
