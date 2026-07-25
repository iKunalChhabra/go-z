package zod

import "fmt"

// Params customizes a schema or check at construction time, porting Zod's
// params union (`string | { error, abort, coerce }`). Fluent methods accept
// variadic `...any` where each element may be:
//
//   - string                    → fixed error message
//   - ErrorMap / func(*Issue) string
//   - Params / *Params          → full options
//   - URLOpts, JWTOpts, MACOpts, ISOTimeOpts, ISODateTimeOpts → format opts
//     (Error/Abort merged; format-specific fields read by the format helper)
//
// Unknown types panic — silent ignore was a DX trap. Check-specific extras
// that are not listed above (e.g. Includes start position `int`) must be
// filtered by the check before calling normalizeParams.
type Params struct {
	// Error customizes messages for issues produced by this schema/check.
	Error ErrorMap
	// Abort stops subsequent checks when this check fails.
	Abort bool
	// Coerce enables input coercion (primitives only).
	Coerce bool
}

// normalizeParams folds a variadic params list into a single Params, last
// writer wins per field. It is tolerant of nil entries. Unsupported types panic.
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
		case URLOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
			}
		case *URLOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
				}
			}
		case JWTOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
			}
		case *JWTOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
				}
			}
		case MACOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
			}
		case *MACOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
				}
			}
		case ISOTimeOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
			}
		case *ISOTimeOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
				}
			}
		case ISODateTimeOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
			}
		case *ISODateTimeOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
				}
			}
		case map[string]string:
			// JWT alg shorthand: FormatJWT reads this; ignore for Params merge.
		default:
			panic(fmt.Sprintf("zod: unsupported params type %T (want string, ErrorMap, Params, format opts, or nil)", p))
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
