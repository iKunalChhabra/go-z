package zod

// Params customizes a schema or check at construction time, porting Zod's
// params union (`string | { error, abort, coerce }`). Fluent methods accept
// variadic `...any` where each element may be:
//
//   - string        → fixed error message (z.string().min(5, "too short"))
//   - ErrorMap      → custom error map
//   - Params/*Params → full options
type Params struct {
	// Error customizes messages for issues produced by this schema/check.
	Error ErrorMap
	// Abort stops subsequent checks when this check fails.
	Abort bool
	// Coerce enables input coercion (primitives only).
	Coerce bool
}

// normalizeParams folds a variadic params list into a single Params, last
// writer wins per field. It is tolerant of nil entries.
func normalizeParams(params []any) Params {
	var out Params
	for _, p := range params {
		switch x := p.(type) {
		case nil:
		case string:
			out.Error = MessageFromString(x)
		case ErrorMap:
			out.Error = x
		case func(iss *Issue) string:
			out.Error = ErrorMap(x)
		case Params:
			mergeParams(&out, &x)
		case *Params:
			if x != nil {
				mergeParams(&out, x)
			}
		}
	}
	return out
}

func mergeParams(dst, src *Params) {
	if src.Error != nil {
		dst.Error = src.Error
	}
	if src.Abort {
		dst.Abort = true
	}
	if src.Coerce {
		dst.Coerce = true
	}
}
