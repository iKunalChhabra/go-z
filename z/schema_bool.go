package z

import (
	"strings"
)

// BoolSchema is the schema.
type BoolSchema struct {
	schemaBase[bool]
	def *Def
}

// Bool returns a boolean schema (z.boolean()).
func Bool(params ...any) *BoolSchema {
	p := normalizeParams(params)
	def := &Def{Type: "boolean", Error: p.Error, Coerce: p.Coerce}
	return newBool(def)
}

func newBool(def *Def) *BoolSchema {
	s := &BoolSchema{def: def}
	s.schemaBase = newBase[bool](buildInternals(def, makeBoolParse(def)))
	return s
}

func makeBoolParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			if b, ok := coerceToBool(p.Value); ok {
				p.Value = b
			}
		}
		if b, ok := p.Value.(bool); ok {
			p.Value = b
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "boolean",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

// coerceToBool accepts true/false, 1/0, "true"/"false", "1"/"0" (case-insensitive).
func coerceToBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		default:
			return false, false
		}
	case float64:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
		return false, false
	case float32:
		return coerceToBool(float64(x))
	case int:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
		return false, false
	case int64:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
		return false, false
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		f, ok := ToFloat(x)
		if !ok {
			return false, false
		}
		return coerceToBool(f)
	default:
		return false, false
	}
}

// Check attaches raw checks.
func (s *BoolSchema) Check(checks ...*Check) *BoolSchema {
	return newBool(s.def.withChecks(checks...))
}
