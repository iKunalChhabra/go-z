package zod

// coerceNS holds the z.coerce.* constructors. Only primitives available on
// this branch are wired; Number/Bool/Time are added when those schemas land.
type coerceNS struct {
	String func(params ...any) *StringSchema
}

// Coerce mirrors Zod's z.coerce namespace. Each constructor returns the
// corresponding primitive with Def.Coerce=true.
var Coerce = coerceNS{
	String: func(params ...any) *StringSchema {
		args := make([]any, 0, len(params)+1)
		args = append(args, Params{Coerce: true})
		args = append(args, params...)
		return String(args...)
	},
}

// When NumberSchema / BoolSchema / TimeSchema land (WP-B), wire:
//
//	Coerce.Number = func(params ...any) *NumberSchema {
//	    return Number(append([]any{Params{Coerce: true}}, params...)...)
//	}
//	Coerce.Bool = ...
//	Coerce.Time = ...
