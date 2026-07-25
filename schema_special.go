package zod

import "math"

// NeverSchema always fails with invalid_type expected "never" (z.never()).
type NeverSchema struct {
	schemaBase[any]
	def *Def
}

// Never returns a schema that rejects every input.
func Never(params ...any) *NeverSchema {
	p := normalizeParams(params)
	def := &Def{Type: "never", Error: p.Error}
	s := &NeverSchema{def: def}
	s.schemaBase = newBase[any](buildInternals(def, func(payload *Payload, _ *ParseCtx) {
		payload.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "never",
			Input:    payload.Value,
			errMap:   def.Error,
		})
	}))
	return s
}

// NilSchema accepts only nil (JSON null) — z.null().
type NilSchema struct {
	schemaBase[any]
	def *Def
}

// Nil returns a schema that accepts only nil (JSON null).
func Nil(params ...any) *NilSchema {
	return newNil(params...)
}

// Null is an alias of Nil (z.null()).
func Null(params ...any) *NilSchema {
	return newNil(params...)
}

func newNil(params ...any) *NilSchema {
	p := normalizeParams(params)
	def := &Def{Type: "null", Error: p.Error}
	s := &NilSchema{def: def}
	in := buildInternals(def, func(payload *Payload, _ *ParseCtx) {
		if payload.Value == nil {
			payload.Value = nil
			return
		}
		payload.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "null",
			Input:    payload.Value,
			errMap:   def.Error,
		})
	})
	in.Values = map[any]struct{}{nil: {}}
	s.schemaBase = newBase[any](in)
	return s
}

// NanSchema accepts float64 NaN only (z.nan()).
type NanSchema struct {
	schemaBase[float64]
	def *Def
}

// Nan returns a schema that accepts only NaN.
func Nan(params ...any) *NanSchema {
	p := normalizeParams(params)
	def := &Def{Type: "nan", Error: p.Error}
	s := &NanSchema{def: def}
	s.schemaBase = newBase[float64](buildInternals(def, func(payload *Payload, _ *ParseCtx) {
		f, ok := ToFloat(payload.Value)
		if ok && math.IsNaN(f) {
			payload.Value = math.NaN()
			return
		}
		payload.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "nan",
			Input:    payload.Value,
			errMap:   def.Error,
		})
	}))
	return s
}
