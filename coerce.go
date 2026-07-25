package zod

// coerceNS holds the z.coerce.* constructors (Zod's z.coerce namespace).
type coerceNS struct {
	String func(params ...any) *StringSchema
	Number func(params ...any) *NumberSchema
	Bool   func(params ...any) *BoolSchema
	Time   func(params ...any) *TimeSchema
}

func withCoerce(params []any) []any {
	args := make([]any, 0, len(params)+1)
	args = append(args, Params{Coerce: true})
	return append(args, params...)
}

// Coerce mirrors Zod's z.coerce namespace. Each constructor returns the
// corresponding primitive with Def.Coerce=true.
var Coerce = coerceNS{
	String: func(params ...any) *StringSchema { return String(withCoerce(params)...) },
	Number: func(params ...any) *NumberSchema { return Number(withCoerce(params)...) },
	Bool:   func(params ...any) *BoolSchema { return Bool(withCoerce(params)...) },
	Time:   func(params ...any) *TimeSchema { return Time(withCoerce(params)...) },
}
