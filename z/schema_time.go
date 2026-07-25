package z

import (
	"time"
)

// TimeSchema is the Go port of Date / Date (Schema[time.Time]).
type TimeSchema struct {
	schemaBase[time.Time]
	def *Def
}

// Time returns a date/time schema. Accepts time.Time
// and *time.Time; with Coerce also RFC3339 strings.
func Time(params ...any) *TimeSchema {
	p := normalizeParams(params)
	def := &Def{Type: "date", Error: p.Error, Coerce: p.Coerce}
	return newTime(def)
}

func newTime(def *Def) *TimeSchema {
	s := &TimeSchema{def: def}
	s.schemaBase = newBase[time.Time](buildInternals(def, makeTimeParse(def)))
	return s
}

func makeTimeParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			if t, ok := coerceToTime(p.Value); ok {
				p.Value = t
			}
		}
		switch x := p.Value.(type) {
		case time.Time:
			p.Value = x
			return
		case *time.Time:
			if x != nil {
				p.Value = *x
				return
			}
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "date",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

func coerceToTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case *time.Time:
		if x == nil {
			return time.Time{}, false
		}
		return *x, true
	case string:
		t, err := time.Parse(time.RFC3339Nano, x)
		if err != nil {
			t, err = time.Parse(time.RFC3339, x)
		}
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	case float64:
		// Unix milliseconds (common JSON date representation).
		return time.UnixMilli(int64(x)), true
	case int64:
		return time.UnixMilli(x), true
	case int:
		return time.UnixMilli(int64(x)), true
	default:
		return time.Time{}, false
	}
}

// Min attaches a minimum (inclusive) date check — too_small Origin="date".
func (s *TimeSchema) Min(min time.Time, params ...any) *TimeSchema {
	return newTime(s.def.withChecks(GreaterThan(min, true, params...)))
}

// Max attaches a maximum (inclusive) date check — too_big Origin="date".
func (s *TimeSchema) Max(max time.Time, params ...any) *TimeSchema {
	return newTime(s.def.withChecks(LessThan(max, true, params...)))
}

// Check attaches raw checks.
func (s *TimeSchema) Check(checks ...*Check) *TimeSchema {
	return newTime(s.def.withChecks(checks...))
}
