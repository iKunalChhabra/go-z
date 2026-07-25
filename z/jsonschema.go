package z

import (
	"fmt"
	"math"
)

// JSONSchemaTarget selects the JSON Schema dialect for ToJSONSchema.
type JSONSchemaTarget string

const (
	JSONSchemaDraft202012 JSONSchemaTarget = "draft-2020-12"
	JSONSchemaDraft07     JSONSchemaTarget = "draft-07"
	JSONSchemaOpenAPI30   JSONSchemaTarget = "openapi-3.0"
)

// ToJSONSchemaOpts configures ToJSONSchema (subset of toJSONSchema params).
type ToJSONSchemaOpts struct {
	// Target defaults to draft-2020-12. Older dialects are produced by
	// rewriting the 2020-12 document: draft-07 gets the array form of tuple
	// items, and openapi-3.0 gets nullable, single-value enums and boolean
	// exclusive bounds. Constructs the target cannot express (a tuple or a
	// bare null in OpenAPI 3.0) follow the Unrepresentable policy.
	Target JSONSchemaTarget
	// Metadata registry for id/title/description (defaults to GlobalRegistry).
	Metadata *Registry[map[string]any]
	// Unrepresentable: "throw" (default) or "any" (emit {}).
	Unrepresentable string
	// IO: "output" (default) or "input" — which side of pipes/codecs to emit.
	IO string
}

// ToJSONSchema converts a schema into a JSON Schema object
// (map[string]any), porting z.toJSONSchema for the common type surface.
func ToJSONSchema(schema AnySchemaLike, opts ...ToJSONSchemaOpts) (map[string]any, error) {
	var o ToJSONSchemaOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	switch o.Target {
	case "":
		o.Target = JSONSchemaDraft202012
	case JSONSchemaDraft202012, JSONSchemaDraft07, JSONSchemaOpenAPI30:
	default:
		// Silently emitting 2020-12 for a target the caller named would hand them
		// a document their consumer cannot read.
		return nil, fmt.Errorf("go-z: ToJSONSchema: unknown Target %q (want %q, %q or %q)",
			o.Target, JSONSchemaDraft202012, JSONSchemaDraft07, JSONSchemaOpenAPI30)
	}
	if o.Metadata == nil {
		o.Metadata = GlobalRegistry
	}
	if o.Unrepresentable == "" {
		o.Unrepresentable = "throw"
	}
	if o.IO == "" {
		o.IO = "output"
	}
	seen := map[*Internals]bool{}
	out, err := emitJSONSchema(schema, o, seen)
	if err != nil {
		return nil, err
	}
	out, err = applyDialect(out, o)
	if err != nil {
		return nil, err
	}
	if _, has := out["$schema"]; !has {
		switch o.Target {
		case JSONSchemaDraft202012:
			out["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		case JSONSchemaDraft07:
			out["$schema"] = "http://json-schema.org/draft-07/schema#"
		case JSONSchemaOpenAPI30:
			// OpenAPI 3.0 schemas typically omit $schema
		}
	}
	return out, nil
}

func emitJSONSchema(schema AnySchemaLike, o ToJSONSchemaOpts, seen map[*Internals]bool) (map[string]any, error) {
	if schema == nil {
		return nil, fmt.Errorf("go-z: ToJSONSchema(nil)")
	}
	in := schema.Internals()
	if seen[in] {
		return map[string]any{}, nil // break cycles as empty (inline)
	}
	seen[in] = true
	defer delete(seen, in)

	defType := ""
	if in.Def != nil {
		defType = in.Def.Type
	}

	out := map[string]any{}
	if meta, ok := o.Metadata.Get(schema); ok {
		if id, ok := meta["id"].(string); ok && id != "" {
			out["$id"] = id
		}
		if title, ok := meta["title"].(string); ok && title != "" {
			out["title"] = title
		}
		if desc, ok := meta["description"].(string); ok && desc != "" {
			out["description"] = desc
		}
		if dep, ok := meta["deprecated"].(bool); ok && dep {
			out["deprecated"] = true
		}
	}

	switch defType {
	case "string":
		out["type"] = "string"
		applyStringChecks(out, in)
	case "number", "int64":
		out["type"] = "number"
		if defType == "int64" || hasFormatCheck(in, "safeint") || hasFormatCheck(in, "int32") || hasFormatCheck(in, "uint32") {
			out["type"] = "integer"
		}
		applyNumberChecks(out, in)
	case "boolean":
		out["type"] = "boolean"
	case "null":
		out["type"] = "null"
	case "any", "unknown", "json":
		// unrestricted
	case "never":
		out["not"] = map[string]any{}
	case "literal":
		if vals := valueEnum(in); len(vals) == 1 {
			out["const"] = vals[0]
		} else if len(vals) > 1 {
			out["enum"] = vals
		}
	case "enum":
		if vals := valueEnum(in); len(vals) > 0 {
			out["enum"] = vals
			out["type"] = "string"
		}
	case "object":
		if err := emitObjectJSONSchema(schema, out, o, seen); err != nil {
			return nil, err
		}
	case "array", "tuple":
		if err := emitArrayJSONSchema(schema, out, o, seen); err != nil {
			return nil, err
		}
	case "record":
		out["type"] = "object"
		out["additionalProperties"] = map[string]any{}
		if rs, ok := schema.(*RecordSchema); ok && rs.valueSchema != nil {
			vs, err := emitJSONSchema(rs.valueSchema, o, seen)
			if err != nil {
				return nil, err
			}
			out["additionalProperties"] = vs
		}
	case "union", "xor":
		opts := unionOptions(schema)
		anyOf := make([]any, 0, len(opts))
		for _, opt := range opts {
			part, err := emitJSONSchema(opt, o, seen)
			if err != nil {
				return nil, err
			}
			anyOf = append(anyOf, part)
		}
		key := "anyOf"
		if defType == "xor" {
			key = "oneOf"
		}
		out[key] = anyOf
	case "intersection":
		if is, ok := schema.(*IntersectionSchema); ok {
			allOf := []any{}
			for _, side := range []AnySchemaLike{is.Left, is.Right} {
				if side == nil {
					continue
				}
				part, err := emitJSONSchema(side, o, seen)
				if err != nil {
					return nil, err
				}
				allOf = append(allOf, part)
			}
			out["allOf"] = allOf
		}
	case "optional", "nullable", "default", "prefault", "catch", "readonly", "nonoptional", "pipe", "codec", "transform", "check", "lazy":
		inner := unwrapJSONSchemaInner(schema, o.IO)
		if inner == nil {
			return handleUnrepresentable(defType, o)
		}
		part, err := emitJSONSchema(inner, o, seen)
		if err != nil {
			return nil, err
		}
		for k, v := range part {
			if _, exists := out[k]; !exists {
				out[k] = v
			}
		}
		if defType == "nullable" {
			// type: [T, "null"] when possible
			if t, ok := out["type"].(string); ok {
				out["type"] = []any{t, "null"}
			}
		}
	case "undefined", "void", "nan", "bigint", "date", "time", "map", "set":
		return handleUnrepresentable(defType, o)
	case "template_literal":
		out["type"] = "string"
		if in.Pattern != nil {
			out["pattern"] = in.Pattern.String()
		}
	default:
		return handleUnrepresentable(defType, o)
	}

	return out, nil
}

func handleUnrepresentable(kind string, o ToJSONSchemaOpts) (map[string]any, error) {
	if o.Unrepresentable == "any" {
		return map[string]any{}, nil
	}
	return nil, fmt.Errorf("go-z: ToJSONSchema: unrepresentable type %q", kind)
}

func valueEnum(in *Internals) []any {
	if in == nil || in.Values == nil {
		return nil
	}
	out := make([]any, 0, len(in.Values))
	for v := range in.Values {
		if IsMissing(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func hasFormatCheck(in *Internals, format string) bool {
	if in == nil || in.Bag == nil {
		return false
	}
	if f, ok := in.Bag["format"].(string); ok && f == format {
		return true
	}
	if in.Def == nil {
		return false
	}
	for _, ch := range in.Def.Checks {
		if ch == nil {
			continue
		}
		// bag format from OnAttach
	}
	_ = format
	return false
}

func applyStringChecks(out map[string]any, in *Internals) {
	if in == nil || in.Bag == nil {
		return
	}
	if f, ok := in.Bag["format"].(string); ok && f != "" {
		switch f {
		case "email", "uuid", "uri", "url", "date-time", "date", "time", "ipv4", "ipv6":
			out["format"] = normalizeJSONFormat(f)
		default:
			out["format"] = f
		}
	}
	// String length checks record their bounds as minimum/maximum, the same
	// keys the numeric checks use; JSON Schema spells them minLength/maxLength.
	if n, ok := asFloat(in.Bag["minimum"]); ok {
		out["minLength"] = int(n)
	} else if n, ok := asFloat(in.Bag["minLength"]); ok {
		out["minLength"] = int(n)
	}
	if n, ok := asFloat(in.Bag["maximum"]); ok {
		out["maxLength"] = int(n)
	} else if n, ok := asFloat(in.Bag["maxLength"]); ok {
		out["maxLength"] = int(n)
	}
	if pat := in.Pattern; pat != nil {
		out["pattern"] = pat.String()
	}
}

func normalizeJSONFormat(f string) string {
	switch f {
	case "url":
		return "uri"
	case "datetime":
		return "date-time"
	default:
		return f
	}
}

func applyNumberChecks(out map[string]any, in *Internals) {
	if in == nil || in.Bag == nil {
		return
	}
	// Bounds are emitted in the type they were recorded in: rendering an int64
	// bound as a float64 would round it, which is the case Int64 exists for.
	for _, key := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
		if v, ok := numericBagValue(in.Bag[key]); ok {
			out[key] = v
		}
	}
	if n, ok := asFloat(in.Bag["multipleOf"]); ok && n > 0 && !math.IsNaN(n) {
		out["multipleOf"] = n
	}
}

func asFloat(v any) (float64, bool) {
	return ToFloat(v)
}

// numericBagValue returns a bag entry for emission, keeping integers exact and
// widening anything else through float64.
func numericBagValue(v any) (any, bool) {
	switch v.(type) {
	case nil:
		return nil, false
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v, true
	}
	f, ok := asFloat(v)
	return f, ok
}

func emitObjectJSONSchema(schema AnySchemaLike, out map[string]any, o ToJSONSchemaOpts, seen map[*Internals]bool) error {
	out["type"] = "object"
	os, ok := schema.(*ObjectSchema)
	if !ok {
		return nil
	}
	props := map[string]any{}
	required := []string{}
	for _, f := range os.fields {
		if f.schema == nil {
			continue
		}
		part, err := emitJSONSchema(f.schema, o, seen)
		if err != nil {
			return err
		}
		props[f.key] = part
		child := f.child
		if child != nil && !child.OptIn {
			required = append(required, f.key)
		}
	}
	out["properties"] = props
	if len(required) > 0 {
		out["required"] = required
	}
	switch os.mode {
	case unknownStrict:
		out["additionalProperties"] = false
	case unknownCatchall:
		if os.catchall != nil {
			ap, err := emitJSONSchema(os.catchall, o, seen)
			if err != nil {
				return err
			}
			out["additionalProperties"] = ap
		}
	}
	return nil
}

func emitArrayJSONSchema(schema AnySchemaLike, out map[string]any, o ToJSONSchemaOpts, seen map[*Internals]bool) error {
	out["type"] = "array"
	switch s := schema.(type) {
	case *ArraySchema:
		if s.element != nil {
			items, err := emitJSONSchema(s.element, o, seen)
			if err != nil {
				return err
			}
			out["items"] = items
		}
		if s.in != nil && s.in.Bag != nil {
			// Array length checks store bag keys as minimum/maximum.
			if n, ok := asFloat(s.in.Bag["minimum"]); ok {
				out["minItems"] = int(n)
			} else if n, ok := asFloat(s.in.Bag["minLength"]); ok {
				out["minItems"] = int(n)
			}
			if n, ok := asFloat(s.in.Bag["maximum"]); ok {
				out["maxItems"] = int(n)
			} else if n, ok := asFloat(s.in.Bag["maxLength"]); ok {
				out["maxItems"] = int(n)
			}
		}
	case *TupleSchema:
		prefix := make([]any, 0, len(s.items))
		for _, it := range s.items {
			if it == nil {
				continue
			}
			part, err := emitJSONSchema(it, o, seen)
			if err != nil {
				return err
			}
			prefix = append(prefix, part)
		}
		out["prefixItems"] = prefix
		if s.rest != nil {
			rest, err := emitJSONSchema(s.rest, o, seen)
			if err != nil {
				return err
			}
			out["items"] = rest
		} else {
			out["items"] = false
		}
	}
	return nil
}

func unionOptions(schema AnySchemaLike) []AnySchemaLike {
	switch s := schema.(type) {
	case *UnionSchema:
		return s.Options
	case *XorSchema:
		return s.Options
	default:
		return nil
	}
}

// unwrapJSONSchemaInner picks the schema a wrapper should be represented by.
// Pipes and codecs have two sides, so they are selected by io; everything else
// exposes its single inner schema through Unwrapper, which means new wrapper
// types work here without edits.
func unwrapJSONSchemaInner(schema AnySchemaLike, io string) AnySchemaLike {
	switch s := schema.(type) {
	case *PipeSchema:
		if io == "input" {
			return s.inSchema
		}
		return s.outSchema
	case *CodecSchema:
		if io == "input" {
			return s.inSch
		}
		return s.outSch
	case Unwrapper:
		return s.Unwrap()
	default:
		return nil
	}
}
