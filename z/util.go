package z

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParsedType returns name for a Go value's runtime type, used in
// invalid_type messages ("expected string, received number"). It ports
// util.parsedType with JS→Go equivalences:
//
//	nil → "null", Missing → "undefined", all numerics → "number" (NaN → "NaN"),
//	*big.Int → "bigint", time.Time → "date", map[string]any → "object",
//	slices/arrays → "array", other named types → their Go type name.
func ParsedType(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case missingType:
		return "undefined"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		if math.IsNaN(x) {
			return "NaN"
		}
		return "number"
	case float32:
		if math.IsNaN(float64(x)) {
			return "NaN"
		}
		return "number"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case *big.Int:
		return "bigint"
	case time.Time, *time.Time:
		return "date"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Func:
		return "function"
	case reflect.Ptr:
		if rv.IsNil() {
			return "null"
		}
		return ParsedType(rv.Elem().Interface())
	}
	return rv.Kind().String()
}

// StringifyPrimitive ports util.stringifyPrimitive: strings quoted, bigints
// suffixed with "n", everything else via default formatting.
func StringifyPrimitive(v any) string {
	switch x := v.(type) {
	case string:
		return `"` + x + `"`
	case *big.Int:
		return x.String() + "n"
	case float64:
		return formatFloat(x)
	case float32:
		return formatFloat(float64(x))
	case nil:
		return "null"
	}
	return fmt.Sprintf("%v", v)
}

// JoinValues ports util.joinValues.
func JoinValues(values []any, sep string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = StringifyPrimitive(v)
	}
	return strings.Join(parts, sep)
}

// formatFloat renders floats like JS Number.prototype.toString (no trailing
// ".0", integer floats printed as integers).
func formatFloat(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e21 && !math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// FormatNumeric renders any numeric issue bound (float64, int64, *big.Int,
// time.Time) for messages, matching `.toString()` output.
func FormatNumeric(v any) string {
	switch x := v.(type) {
	case float64:
		return formatFloat(x)
	case float32:
		return formatFloat(float64(x))
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case *big.Int:
		return x.String()
	case time.Time:
		return x.Format(time.RFC3339)
	}
	return fmt.Sprintf("%v", v)
}

// ToFloat converts any Go numeric to float64. ok is false for non-numerics.
func ToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	}
	return 0, false
}

// IsIntegral reports whether v is a numeric with an integer value.
func IsIntegral(v any) bool {
	switch x := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return x == math.Trunc(x) && !math.IsInf(x, 0) && !math.IsNaN(x)
	case float32:
		f := float64(x)
		return f == math.Trunc(f) && !math.IsInf(f, 0) && !math.IsNaN(f)
	}
	return false
}
