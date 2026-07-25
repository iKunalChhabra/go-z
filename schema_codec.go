package zod

import (
	"encoding/json"
	"fmt"
)

// ParseDirection selects forward (decode) or backward (encode) parsing.
type ParseDirection int

const (
	// DirectionDecode is forward parsing (input → output). Default for Parse.
	DirectionDecode ParseDirection = iota
	// DirectionEncode is backward parsing (output → input).
	DirectionEncode
)

// IsEncode reports whether ctx requests encode direction.
func (ctx *ParseCtx) IsEncode() bool {
	return ctx != nil && ctx.Direction == DirectionEncode
}

// CodecTx holds the bidirectional transforms for Codec.
type CodecTx struct {
	Decode func(any, *RefinementCtx) (any, error)
	Encode func(any, *RefinementCtx) (any, error)
}

// CodecSchema is a bidirectional pipe between an input schema and an output
// schema (Zod's z.codec / $ZodCodec). Decode: in → decodeTx → out.
// Encode: out → encodeTx → in.
type CodecSchema struct {
	schemaBase[any]
	def    *Def
	inSch  AnySchemaLike
	outSch AnySchemaLike
	tx     CodecTx
}

// Codec returns a bidirectional codec schema.
func Codec(in, out AnySchemaLike, tx CodecTx) *CodecSchema {
	if in == nil || out == nil {
		panic("zod: Codec requires non-nil input and output schemas")
	}
	if tx.Decode == nil {
		tx.Decode = func(v any, _ *RefinementCtx) (any, error) { return v, nil }
	}
	if tx.Encode == nil {
		tx.Encode = func(v any, _ *RefinementCtx) (any, error) { return v, nil }
	}
	def := &Def{Type: "codec"}
	return newCodec(def, in, out, tx)
}

func newCodec(def *Def, in, out AnySchemaLike, tx CodecTx) *CodecSchema {
	s := &CodecSchema{def: def, inSch: in, outSch: out, tx: tx}
	inIn := in.Internals()
	outIn := out.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if ctx.IsEncode() {
			// encode: validate with out, encodeTx, validate with in
			RunSelf(outIn, p, ctx)
			if len(p.Issues) > 0 {
				p.Aborted = true
				return
			}
			rctx := &RefinementCtx{payload: p}
			next, err := tx.Encode(p.Value, rctx)
			if err != nil {
				p.AddIssue(Issue{Code: IssueCustom, Message: err.Error(), Input: p.Value})
				return
			}
			if len(p.Issues) > 0 {
				return
			}
			p.Value = next
			RunSelf(inIn, p, ctx)
			return
		}
		// decode: validate with in, decodeTx, validate with out
		RunSelf(inIn, p, ctx)
		if len(p.Issues) > 0 {
			p.Aborted = true
			return
		}
		rctx := &RefinementCtx{payload: p}
		next, err := tx.Decode(p.Value, rctx)
		if err != nil {
			p.AddIssue(Issue{Code: IssueCustom, Message: err.Error(), Input: p.Value})
			return
		}
		if len(p.Issues) > 0 {
			return
		}
		p.Value = next
		RunSelf(outIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = inIn.OptIn
	s.in.OptOut = outIn.OptOut
	propagateWrapperMeta(s.in, inIn)
	return s
}

// In returns the input-side schema.
func (s *CodecSchema) In() AnySchemaLike { return s.inSch }

// Out returns the output-side schema.
func (s *CodecSchema) Out() AnySchemaLike { return s.outSch }

// Check attaches raw checks after codec parse (Zod .check / .refine on codecs).
func (s *CodecSchema) Check(checks ...*Check) *CheckedSchema {
	return CheckSchema(s, checks...)
}

// InvertCodec swaps input/output schemas and decode/encode transforms.
func InvertCodec(c *CodecSchema) *CodecSchema {
	if c == nil {
		panic("zod: InvertCodec(nil)")
	}
	return Codec(c.outSch, c.inSch, CodecTx{
		Decode: c.tx.Encode,
		Encode: c.tx.Decode,
	})
}

// Decode parses data in the forward direction (same as Parse for most schemas).
func Decode(schema AnySchemaLike, data any) (any, error) {
	return runDirectional(schema, data, DirectionDecode)
}

// Encode parses data in the backward direction (output → input).
func Encode(schema AnySchemaLike, data any) (any, error) {
	return runDirectional(schema, data, DirectionEncode)
}

// SafeDecode is Decode with a SafeParseResult.
func SafeDecode(schema AnySchemaLike, data any) SafeParseResult[any] {
	v, err := Decode(schema, data)
	if err != nil {
		zerr, _ := err.(*ZodError)
		return SafeParseResult[any]{Success: false, Error: zerr}
	}
	return SafeParseResult[any]{Success: true, Data: v}
}

// SafeEncode is Encode with a SafeParseResult.
func SafeEncode(schema AnySchemaLike, data any) SafeParseResult[any] {
	v, err := Encode(schema, data)
	if err != nil {
		zerr, _ := err.(*ZodError)
		return SafeParseResult[any]{Success: false, Error: zerr}
	}
	return SafeParseResult[any]{Success: true, Data: v}
}

func runDirectional(schema AnySchemaLike, data any, dir ParseDirection) (any, error) {
	if schema == nil {
		return nil, fmt.Errorf("zod: nil schema")
	}
	ctx := &ParseCtx{Direction: dir}
	p := AcquirePayload(data)
	p.parseCtx = ctx
	schema.Internals().Run(p, ctx)
	if len(p.Issues) > 0 {
		err := newZodError(p.Issues, ctx)
		ReleasePayload(p)
		return nil, err
	}
	out := p.Value
	ReleasePayload(p)
	return out, nil
}

// JSONStringCodec returns a codec that JSON-parses strings into schema and
// stringifies on encode (Zod's jsonCodec helper pattern).
func JSONStringCodec(schema AnySchemaLike) *CodecSchema {
	return Codec(String(), schema, CodecTx{
		Decode: func(v any, ctx *RefinementCtx) (any, error) {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("expected string")
			}
			var out any
			if err := json.Unmarshal([]byte(s), &out); err != nil {
				ctx.AddIssue(Issue{
					Code:    IssueInvalidFormat,
					Format:  "json",
					Message: err.Error(),
					Input:   v,
				})
				return Missing, nil
			}
			return out, nil
		},
		Encode: func(v any, _ *RefinementCtx) (any, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		},
	})
}
