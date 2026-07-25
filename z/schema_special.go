package z

import (
	"math"
	"strings"
)

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

// UndefinedSchema accepts only Missing (undefined) — z.undefined().
type UndefinedSchema struct {
	schemaBase[any]
	def *Def
}

// Undefined returns a schema that accepts only the Missing sentinel.
func Undefined(params ...any) *UndefinedSchema {
	p := normalizeParams(params)
	def := &Def{Type: "undefined", Error: p.Error}
	s := &UndefinedSchema{def: def}
	in := buildInternals(def, func(payload *Payload, _ *ParseCtx) {
		if IsMissing(payload.Value) {
			payload.Value = missingSentinel
			return
		}
		payload.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "undefined",
			Input:    payload.Value,
			errMap:   def.Error,
		})
	})
	in.Values = map[any]struct{}{missingSentinel: {}}
	s.schemaBase = newBase[any](in)
	return s
}

// VoidSchema accepts only Missing — in v4, z.void() is equivalent to
// z.undefined() for parsing (expected type name is "void").
type VoidSchema struct {
	schemaBase[any]
	def *Def
}

// Void returns a schema that accepts only the Missing sentinel (z.void()).
func Void(params ...any) *VoidSchema {
	p := normalizeParams(params)
	def := &Def{Type: "void", Error: p.Error}
	s := &VoidSchema{def: def}
	in := buildInternals(def, func(payload *Payload, _ *ParseCtx) {
		if IsMissing(payload.Value) {
			payload.Value = missingSentinel
			return
		}
		payload.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "void",
			Input:    payload.Value,
			errMap:   def.Error,
		})
	})
	in.Values = map[any]struct{}{missingSentinel: {}}
	s.schemaBase = newBase[any](in)
	return s
}

// JSONSchema validates recursive JSON values (string|number|bool|null|array|object).
type JSONSchema struct {
	schemaBase[any]
	def *Def
}

// JSON returns a schema for JSON-compatible values (z.json()), built via Lazy.
func JSON(params ...any) *JSONSchema {
	p := normalizeParams(params)
	def := &Def{Type: "json", Error: p.Error}
	var lazy *LazySchema
	lazy = Lazy(func() AnySchemaLike {
		return Union([]AnySchemaLike{
			String(params...),
			Number(),
			Bool(),
			Nil(),
			Array(lazy),
			Record(String(), lazy),
		})
	})
	s := &JSONSchema{def: def}
	s.schemaBase = newBase[any](buildInternals(def, func(payload *Payload, ctx *ParseCtx) {
		RunSelf(lazy.Internals(), payload, ctx)
	}))
	return s
}

// StringBool returns a boolean schema that coerces common string truthy/falsy
// tokens (case-insensitive): true/false, yes/no, 1/0, on/off (and extras
// y/n/enabled/disabled). Non-string inputs fail.
func StringBool(params ...any) *BoolSchema {
	p := normalizeParams(params)
	def := &Def{Type: "boolean", Error: p.Error}
	s := &BoolSchema{def: def}
	s.schemaBase = newBase[bool](buildInternals(def, makeStringBoolParse(def)))
	return s
}

func makeStringBoolParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		str, ok := p.Value.(string)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "boolean",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}
		switch strings.ToLower(strings.TrimSpace(str)) {
		case "true", "yes", "1", "on", "y", "enabled":
			p.Value = true
			return
		case "false", "no", "0", "off", "n", "disabled":
			p.Value = false
			return
		default:
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "boolean",
				Input:    p.Value,
				errMap:   def.Error,
			})
		}
	}
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
