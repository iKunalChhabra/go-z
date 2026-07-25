package z

import (
	"math/big"
	"time"
)

// cloneDefaultValue returns an independent copy of common JSON-model defaults
// (maps/slices) so Default/Prefault/Catch do not share mutable backing storage
// across parses. Scalars and unknown reference types are returned as-is —
// callers who pass custom pointers should use DefaultFunc/CatchFunc.
func cloneDefaultValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneDefaultValue(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = cloneDefaultValue(e)
		}
		return out
	case []string:
		return append([]string(nil), x...)
	case []byte:
		return append([]byte(nil), x...)
	case *big.Int:
		if x == nil {
			return (*big.Int)(nil)
		}
		return new(big.Int).Set(x)
	case time.Time:
		return x
	default:
		return v
	}
}
