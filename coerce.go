package zod

// coerceNS holds the z.coerce.* constructors (Zod's z.coerce namespace).
// Methods (not function fields) so callers cannot replace individual constructors.
type coerceNS struct{}

func withCoerce(params []any) []any {
	args := make([]any, 0, len(params)+1)
	args = append(args, Params{Coerce: true})
	return append(args, params...)
}

// String returns a coercing string schema (z.coerce.string()).
func (coerceNS) String(params ...any) *StringSchema {
	return String(withCoerce(params)...)
}

// Number returns a coercing number schema (z.coerce.number()).
func (coerceNS) Number(params ...any) *NumberSchema {
	return Number(withCoerce(params)...)
}

// Bool returns a coercing bool schema (z.coerce.boolean()).
func (coerceNS) Bool(params ...any) *BoolSchema {
	return Bool(withCoerce(params)...)
}

// Time returns a coercing time schema (z.coerce.date()).
func (coerceNS) Time(params ...any) *TimeSchema {
	return Time(withCoerce(params)...)
}

// BigInt returns a coercing bigint schema (z.coerce.bigint()).
func (coerceNS) BigInt(params ...any) *BigIntSchema {
	return BigInt(withCoerce(params)...)
}

// Coerce mirrors Zod's z.coerce namespace.
//
// Prefer the methods (Coerce.String, …). Reassigning the package variable is
// unsupported and has no effect on already-captured call sites.
var Coerce coerceNS
