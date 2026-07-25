package zod

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
)

// StringSchema is the Go port of ZodString / $ZodString.
type StringSchema struct {
	schemaBase[string]
	def *Def
}

// String returns a string schema (z.string()). Optional params may be a
// string message, ErrorMap, or Params (including Coerce).
func String(params ...any) *StringSchema {
	p := normalizeParams(params)
	def := &Def{Type: "string", Error: p.Error, Coerce: p.Coerce}
	return newString(def)
}

func newString(def *Def) *StringSchema {
	s := &StringSchema{def: def}
	parse := makeStringParse(def)
	s.schemaBase = newBase[string](buildInternals(def, parse))
	return s
}

func makeStringParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			p.Value = coerceToString(p.Value)
		}
		if _, ok := p.Value.(string); ok {
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "string",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

// coerceToString mirrors Zod's String(value) for common primitives.
func coerceToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(x)
	case float64:
		if math.IsNaN(x) {
			return "NaN"
		}
		if math.IsInf(x, 1) {
			return "Infinity"
		}
		if math.IsInf(x, -1) {
			return "-Infinity"
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return coerceToString(float64(x))
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case *big.Int:
		if x == nil {
			return "null"
		}
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Length / content checks
//////////////////////////////////////////////////////////////////////////////

// Min attaches a minimum-length check.
func (s *StringSchema) Min(n int, params ...any) *StringSchema {
	return newString(s.def.withChecks(MinLength(n, params...)))
}

// Max attaches a maximum-length check.
func (s *StringSchema) Max(n int, params ...any) *StringSchema {
	return newString(s.def.withChecks(MaxLength(n, params...)))
}

// Length attaches an exact-length check.
func (s *StringSchema) Length(n int, params ...any) *StringSchema {
	return newString(s.def.withChecks(LengthEquals(n, params...)))
}

// NonEmpty is Min(1).
func (s *StringSchema) NonEmpty(params ...any) *StringSchema {
	return s.Min(1, params...)
}

// Regex attaches a regex format check.
func (s *StringSchema) Regex(pattern *regexp.Regexp, params ...any) *StringSchema {
	return newString(s.def.withChecks(Regex(pattern, params...)))
}

// Includes requires the string to contain value.
func (s *StringSchema) Includes(value string, params ...any) *StringSchema {
	return newString(s.def.withChecks(Includes(value, params...)))
}

// StartsWith requires a prefix.
func (s *StringSchema) StartsWith(value string, params ...any) *StringSchema {
	return newString(s.def.withChecks(StartsWith(value, params...)))
}

// EndsWith requires a suffix.
func (s *StringSchema) EndsWith(value string, params ...any) *StringSchema {
	return newString(s.def.withChecks(EndsWith(value, params...)))
}

// Uppercase requires all characters to be uppercase (check, not transform).
func (s *StringSchema) Uppercase(params ...any) *StringSchema {
	return newString(s.def.withChecks(UpperCase(params...)))
}

// Lowercase requires all characters to be lowercase (check, not transform).
func (s *StringSchema) Lowercase(params ...any) *StringSchema {
	return newString(s.def.withChecks(LowerCase(params...)))
}

//////////////////////////////////////////////////////////////////////////////
// Overwrite transforms
//////////////////////////////////////////////////////////////////////////////

// Trim trims leading/trailing whitespace (overwrite).
func (s *StringSchema) Trim() *StringSchema {
	return newString(s.def.withChecks(Trim()))
}

// ToLowerCase lowercases the value (overwrite).
func (s *StringSchema) ToLowerCase() *StringSchema {
	return newString(s.def.withChecks(ToLowerCase()))
}

// ToUpperCase uppercases the value (overwrite).
func (s *StringSchema) ToUpperCase() *StringSchema {
	return newString(s.def.withChecks(ToUpperCase()))
}

// Normalize applies Unicode NFC normalization (overwrite).
func (s *StringSchema) Normalize() *StringSchema {
	return newString(s.def.withChecks(NormalizeNFC()))
}

//////////////////////////////////////////////////////////////////////////////
// Format checks
//////////////////////////////////////////////////////////////////////////////

func (s *StringSchema) Email(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatEmail(params...)))
}
func (s *StringSchema) URL(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatURL(params...)))
}
func (s *StringSchema) UUID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatUUID(params...)))
}
func (s *StringSchema) UUIDv4(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatUUIDv4(params...)))
}
func (s *StringSchema) UUIDv6(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatUUIDv6(params...)))
}
func (s *StringSchema) UUIDv7(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatUUIDv7(params...)))
}
func (s *StringSchema) GUID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatGUID(params...)))
}
func (s *StringSchema) NanoID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatNanoID(params...)))
}
func (s *StringSchema) CUID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatCUID(params...)))
}
func (s *StringSchema) CUID2(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatCUID2(params...)))
}
func (s *StringSchema) ULID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatULID(params...)))
}
func (s *StringSchema) KSUID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatKSUID(params...)))
}
func (s *StringSchema) XID(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatXID(params...)))
}
func (s *StringSchema) Base64(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatBase64(params...)))
}
func (s *StringSchema) Base64URL(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatBase64URL(params...)))
}
func (s *StringSchema) Hex(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatHex(params...)))
}
func (s *StringSchema) JWT(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatJWT(params...)))
}
func (s *StringSchema) E164(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatE164(params...)))
}
func (s *StringSchema) Emoji(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatEmoji(params...)))
}
func (s *StringSchema) IPv4(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatIPv4(params...)))
}
func (s *StringSchema) IPv6(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatIPv6(params...)))
}
func (s *StringSchema) CIDRv4(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatCIDRv4(params...)))
}
func (s *StringSchema) CIDRv6(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatCIDRv6(params...)))
}
func (s *StringSchema) MAC(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatMAC(params...)))
}
func (s *StringSchema) ISODate(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatISODate(params...)))
}
func (s *StringSchema) ISOTime(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatISOTime(params...)))
}
func (s *StringSchema) ISODateTime(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatISODateTime(params...)))
}
func (s *StringSchema) ISODuration(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatISODuration(params...)))
}

// Check attaches raw checks (composition primitive).
func (s *StringSchema) Check(checks ...*Check) *StringSchema {
	return newString(s.def.withChecks(checks...))
}
