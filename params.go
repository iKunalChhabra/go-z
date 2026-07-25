package zod

import (
	"fmt"
	"reflect"
)

// Params customizes a schema or check at construction time, porting Zod's
// params union (`string | { error, abort, coerce }`). Fluent methods accept
// variadic `...any` where each element may be:
//
//   - string                    → fixed error message
//   - ErrorMap / func(*Issue) string
//   - Params                    → full options
//   - URLOpts, JWTOpts, MACOpts, ISOTimeOpts, ISODateTimeOpts → format opts
//     (Error/Abort merged; format-specific fields read by the format helper)
//
// A pointer to any of the above works too; nil entries and nil pointers are
// skipped.
//
// # Unsupported types panic
//
// Passing anything else panics. Params are only ever read while a schema is
// being defined — normally during package init or startup — so a typo fails
// immediately and loudly instead of silently dropping your error message.
// Schema definition is effectively compile time for an application: if the
// process starts, every schema in it was built with valid params.
//
// Check-specific extras that are not listed above (for example the `Includes`
// start position `int`) must be filtered by the check before it calls
// normalizeParams.
type Params struct {
	// Error customizes messages for issues produced by this schema/check.
	Error ErrorMap
	// Abort stops subsequent checks when this check fails.
	Abort bool
	// Coerce enables input coercion (primitives only).
	Coerce bool
}

// paramsSource is implemented by the typed option structs (URLOpts, JWTOpts,
// …) so normalizeParams needs one case for all of them. Format-specific
// fields stay on the struct and are read by the format helper itself.
type paramsSource interface {
	params() Params
}

// normalizeParams folds a variadic params list into a single Params, last
// writer wins per field. It is tolerant of nil entries. Unsupported types panic.
func normalizeParams(params []any) Params {
	var out Params
	for _, raw := range params {
		p, ok := derefParam(raw)
		if !ok {
			continue
		}
		switch x := p.(type) {
		case string:
			out.Error = MessageFromString(x)
		case ErrorMap:
			out.Error = x
		case func(iss *Issue) string:
			out.Error = ErrorMap(x)
		case Params:
			mergeParams(&out, &x)
		case paramsSource:
			src := x.params()
			mergeParams(&out, &src)
		default:
			panic(fmt.Sprintf("zod: unsupported params type %T (want string, ErrorMap, Params, format opts, or nil)", p))
		}
	}
	return out
}

// derefParam unwraps a pointer param to its value so the switch above only
// needs value cases. It reports false for nil entries and nil pointers.
func derefParam(p any) (any, bool) {
	if p == nil {
		return nil, false
	}
	rv := reflect.ValueOf(p)
	if rv.Kind() != reflect.Pointer {
		return p, true
	}
	if rv.IsNil() {
		return nil, false
	}
	return rv.Elem().Interface(), true
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
