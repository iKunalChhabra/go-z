package z

import (
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Number format range constants (JS Number semantics).
const (
	MaxSafeInteger  = float64(1<<53 - 1) // 9007199254740991
	MinSafeInteger  = -MaxSafeInteger    // -9007199254740991
	MaxFloat32Exact = 3.4028234663852886e38
	MinFloat32Exact = -MaxFloat32Exact
	MaxInt32        = float64(2147483647)
	MinInt32        = float64(-2147483648)
	MaxUint32       = float64(4294967295)
)

// floatSafeRemainder ports util.floatSafeRemainder for multipleOf checks.
func floatSafeRemainder(val, step float64) float64 {
	ratio := val / step
	rounded := math.Round(ratio)
	// Number.EPSILON * Math.max(Math.abs(ratio), 1)
	eps := math.Nextafter(1.0, 2.0) - 1.0
	tolerance := eps * math.Max(math.Abs(ratio), 1)
	if math.Abs(ratio-rounded) < tolerance {
		return 0
	}
	return ratio - rounded
}

// numericOrigin returns the too_small/too_big Origin for a bound value.
func numericOrigin(value any) string {
	switch value.(type) {
	case *big.Int:
		return "bigint"
	case time.Time, *time.Time:
		return "date"
	default:
		return "number"
	}
}

// issueBound normalizes a comparison bound for issue Minimum/Maximum fields.
// Dates become Unix milliseconds (Date.getTime()). Integers report as float64
// like every other number, except beyond the safe-integer range where float64
// would round them — there the exact integer is reported instead.
func issueBound(value any) any {
	switch x := value.(type) {
	case time.Time:
		return float64(x.UnixMilli())
	case *time.Time:
		if x == nil {
			return value
		}
		return float64(x.UnixMilli())
	default:
		if n, ok := exactInt(value); ok {
			if n >= int64(MinSafeInteger) && n <= int64(MaxSafeInteger) {
				return float64(n)
			}
			return n
		}
		return value
	}
}

// compareNumeric compares two numeric-like values. Returns -1, 0, 1 and ok.
func compareNumeric(a, b any) (int, bool) {
	// big.Int
	if ai, ok := asBigInt(a); ok {
		if bi, ok := asBigInt(b); ok {
			return ai.Cmp(bi), true
		}
	}
	// time
	if at, ok := asTime(a); ok {
		if bt, ok := asTime(b); ok {
			if at.Equal(bt) {
				return 0, true
			}
			if at.Before(bt) {
				return -1, true
			}
			return 1, true
		}
	}
	// Two integers compare exactly. float64 would round anything above 2^53,
	// which is precisely the range Int64 exists to serve.
	if ai, ok := exactInt(a); ok {
		if bi, ok := exactInt(b); ok {
			switch {
			case ai < bi:
				return -1, true
			case ai > bi:
				return 1, true
			default:
				return 0, true
			}
		}
	}
	// float64 / ints
	fa, oka := ToFloat(a)
	fb, okb := ToFloat(b)
	if oka && okb {
		switch {
		case fa < fb:
			return -1, true
		case fa > fb:
			return 1, true
		default:
			return 0, true
		}
	}
	return 0, false
}

// exactInt reports v as an int64 when it is a Go integer that fits, so
// comparisons can avoid float64 rounding. Floats are excluded on purpose: a
// float64 has already lost whatever precision it was going to lose.
func exactInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint:
		if uint64(x) > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	case uint64:
		if x > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	default:
		return namedInt(v)
	}
}

// namedInt is the slow path of exactInt: a named integer type (type Port uint16)
// does not match a type switch on its underlying kind.
func namedInt(v any) (int64, bool) {
	if v == nil {
		return 0, false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > math.MaxInt64 {
			return 0, false
		}
		return int64(u), true
	default:
		return 0, false
	}
}

func asBigInt(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case *big.Int:
		if x == nil {
			return nil, false
		}
		return x, true
	case int64:
		return big.NewInt(x), true
	case int:
		return big.NewInt(int64(x)), true
	}
	return nil, false
}

func asTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case *time.Time:
		if x == nil {
			return time.Time{}, false
		}
		return *x, true
	}
	return time.Time{}, false
}

// GreaterThan implements the equivalent check (inclusive ⇒ gte, else gt).
func GreaterThan(value any, inclusive bool, params ...any) *Check {
	p := normalizeParams(params)
	origin := numericOrigin(value)
	ch := &Check{
		Name:  "greater_than",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				key := "minimum"
				if !inclusive {
					key = "exclusiveMinimum"
				}
				in.Bag[key] = issueBound(value)
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		cmp, ok := compareNumeric(payload.Value, value)
		if !ok {
			return
		}
		pass := cmp > 0 || (inclusive && cmp == 0)
		if pass {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooSmall,
			Origin:    origin,
			Minimum:   issueBound(value),
			Inclusive: inclusive,
			Input:     payload.Value,
		}))
	}
	return ch
}

// LessThan implements the equivalent check (inclusive ⇒ lte, else lt).
func LessThan(value any, inclusive bool, params ...any) *Check {
	p := normalizeParams(params)
	origin := numericOrigin(value)
	ch := &Check{
		Name:  "less_than",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				key := "maximum"
				if !inclusive {
					key = "exclusiveMaximum"
				}
				in.Bag[key] = issueBound(value)
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		cmp, ok := compareNumeric(payload.Value, value)
		if !ok {
			return
		}
		pass := cmp < 0 || (inclusive && cmp == 0)
		if pass {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooBig,
			Origin:    origin,
			Maximum:   issueBound(value),
			Inclusive: inclusive,
			Input:     payload.Value,
		}))
	}
	return ch
}

// Gt is GreaterThan(value, false).
func Gt(value float64, params ...any) *Check { return GreaterThan(value, false, params...) }

// Gte is GreaterThan(value, true).
func Gte(value float64, params ...any) *Check { return GreaterThan(value, true, params...) }

// Lt is LessThan(value, false).
func Lt(value float64, params ...any) *Check { return LessThan(value, false, params...) }

// Lte is LessThan(value, true).
func Lte(value float64, params ...any) *Check { return LessThan(value, true, params...) }

// MultipleOf implements the equivalent check for float64 divisors.
func MultipleOf(divisor float64, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "multiple_of",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				if _, ok := in.Bag["multipleOf"]; !ok {
					in.Bag["multipleOf"] = divisor
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		f, ok := ToFloat(payload.Value)
		if !ok {
			return
		}
		if floatSafeRemainder(f, divisor) == 0 {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:    IssueNotMultipleOf,
			Origin:  "number",
			Divisor: divisor,
			Input:   payload.Value,
		}))
	}
	return ch
}

// MultipleOfInt64 is MultipleOf for integer divisors. It uses integer
// remainder, so it stays exact above 2^53 where the float64 path rounds.
func MultipleOfInt64(divisor int64, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "multiple_of",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				if _, ok := in.Bag["multipleOf"]; !ok {
					in.Bag["multipleOf"] = issueBound(divisor)
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		if divisor == 0 {
			return
		}
		n, ok := exactInt(payload.Value)
		if !ok {
			// Not an integer value: fall back to the float comparison.
			f, isNum := ToFloat(payload.Value)
			if !isNum || floatSafeRemainder(f, float64(divisor)) == 0 {
				return
			}
		} else if n%divisor == 0 {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:    IssueNotMultipleOf,
			Origin:  "number",
			Divisor: float64(divisor),
			Input:   payload.Value,
		}))
	}
	return ch
}

// MultipleOfBigInt implements the equivalent check for *big.Int divisors.
func MultipleOfBigInt(divisor *big.Int, params ...any) *Check {
	p := normalizeParams(params)
	if divisor == nil {
		divisor = big.NewInt(0)
	}
	divCopy := new(big.Int).Set(divisor)
	ch := &Check{
		Name:  "multiple_of",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				if _, ok := in.Bag["multipleOf"]; !ok {
					in.Bag["multipleOf"] = new(big.Int).Set(divCopy)
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		v, ok := asBigInt(payload.Value)
		if !ok || divCopy.Sign() == 0 {
			return
		}
		mod := new(big.Int).Mod(v, divCopy)
		if mod.Sign() == 0 {
			return
		}
		// Issue.Divisor is float64; use float when exact, else 0 with message via Error.
		divF, _ := new(big.Float).SetInt(divCopy).Float64()
		payload.AddIssue(ch.Issue(Issue{
			Code:    IssueNotMultipleOf,
			Origin:  "bigint",
			Divisor: divF,
			Input:   payload.Value,
		}))
	}
	return ch
}

// numberFormatPattern returns the regexp shape of a number format, used when a
// numeric schema appears inside a template literal. It describes the digits, not
// the range: the range is enforced by the check itself.
func numberFormatPattern(format string) *regexp.Regexp {
	switch format {
	case "safeint", "int32", "int64":
		return reIntegerText
	case "uint32", "uint64":
		return reUnsignedIntegerText
	case "float32", "float64":
		return reNumberText
	default:
		return nil
	}
}

// NumberFormat implements the equivalent check ("int32"|"uint32"|"float32"|"float64"|"safeint").
func NumberFormat(format string, params ...any) *Check {
	p := normalizeParams(params)
	if format == "" {
		format = "float64"
	}
	isInt := containsInt(format)
	origin := "number"
	if isInt {
		origin = "int"
	}
	minimum, maximum := numberFormatRange(format)

	ch := &Check{
		Name:  "number_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = format
				in.Bag["minimum"] = minimum
				in.Bag["maximum"] = maximum
				attachPattern(in, numberFormatPattern(format))
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		f, ok := ToFloat(payload.Value)
		if !ok {
			return
		}
		if isInt {
			if !IsIntegral(f) {
				// Abort subsequent checks (continue: false).
				iss := ch.Issue(Issue{
					Code:     IssueInvalidType,
					Expected: origin,
					Format:   format,
					Input:    payload.Value,
				})
				iss.cont = continueNo
				payload.AddIssue(iss)
				return
			}
			// Safe-integer gate for int formats (Number.isSafeInteger).
			if f > MaxSafeInteger {
				payload.AddIssue(ch.Issue(Issue{
					Code:      IssueTooBig,
					Origin:    origin,
					Maximum:   MaxSafeInteger,
					Inclusive: true,
					Input:     payload.Value,
				}))
				return
			}
			if f < MinSafeInteger {
				payload.AddIssue(ch.Issue(Issue{
					Code:      IssueTooSmall,
					Origin:    origin,
					Minimum:   MinSafeInteger,
					Inclusive: true,
					Input:     payload.Value,
				}))
				return
			}
		}
		if f < minimum {
			payload.AddIssue(ch.Issue(Issue{
				Code:      IssueTooSmall,
				Origin:    "number",
				Minimum:   minimum,
				Inclusive: true,
				Input:     payload.Value,
			}))
		}
		if f > maximum {
			payload.AddIssue(ch.Issue(Issue{
				Code:      IssueTooBig,
				Origin:    "number",
				Maximum:   maximum,
				Inclusive: true,
				Input:     payload.Value,
			}))
		}
	}
	return ch
}

func containsInt(format string) bool {
	return strings.Contains(format, "int")
}

func numberFormatRange(format string) (minimum, maximum float64) {
	switch format {
	case "safeint":
		return MinSafeInteger, MaxSafeInteger
	case "int32":
		return MinInt32, MaxInt32
	case "uint32":
		return 0, MaxUint32
	case "float32":
		return MinFloat32Exact, MaxFloat32Exact
	case "float64":
		return -math.MaxFloat64, math.MaxFloat64
	default:
		return -math.MaxFloat64, math.MaxFloat64
	}
}
