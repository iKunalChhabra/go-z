package z

import "fmt"

// The emitter produces draft 2020-12. Older dialects are reached by rewriting
// that document, so there is one emitter to keep correct rather than three.
//
// Rewriting only touches positions where a subschema can appear, never arbitrary
// map keys: a property named "const" or "prefixItems" is a field name, not a
// keyword, and must be left alone.
var (
	subschemaKeys      = []string{"items", "additionalItems", "additionalProperties", "not", "contains", "propertyNames", "if", "then", "else"}
	subschemaListKeys  = []string{"anyOf", "oneOf", "allOf", "prefixItems"}
	subschemaMapKeys   = []string{"properties", "patternProperties", "$defs", "definitions"}
	tupleUnsupportedIn = map[JSONSchemaTarget]bool{JSONSchemaOpenAPI30: true}
)

// applyDialect rewrites a 2020-12 document for the requested target.
func applyDialect(doc map[string]any, o ToJSONSchemaOpts) (map[string]any, error) {
	if o.Target == JSONSchemaDraft202012 {
		return doc, nil
	}
	if err := walkSchemaNode(doc, o); err != nil {
		return nil, err
	}
	return doc, nil
}

// walkSchemaNode rewrites node in place, depth first.
func walkSchemaNode(node map[string]any, o ToJSONSchemaOpts) error {
	for _, key := range subschemaKeys {
		if child, ok := node[key].(map[string]any); ok {
			if err := walkSchemaNode(child, o); err != nil {
				return err
			}
		}
	}
	for _, key := range subschemaListKeys {
		list, ok := node[key].([]any)
		if !ok {
			continue
		}
		for _, item := range list {
			if child, ok := item.(map[string]any); ok {
				if err := walkSchemaNode(child, o); err != nil {
					return err
				}
			}
		}
	}
	for _, key := range subschemaMapKeys {
		members, ok := node[key].(map[string]any)
		if !ok {
			continue
		}
		for _, member := range members {
			if child, ok := member.(map[string]any); ok {
				if err := walkSchemaNode(child, o); err != nil {
					return err
				}
			}
		}
	}

	switch o.Target {
	case JSONSchemaDraft07:
		return downgradeToDraft07(node, o)
	case JSONSchemaOpenAPI30:
		return downgradeToOpenAPI30(node, o)
	default:
		return nil
	}
}

// downgradeToDraft07 converts the 2020-12 tuple keywords to the draft-07 array
// form. Everything else the emitter produces — const, numeric exclusive bounds,
// type arrays, anyOf/oneOf/allOf — is already valid draft-07.
func downgradeToDraft07(node map[string]any, _ ToJSONSchemaOpts) error {
	prefix, ok := node["prefixItems"].([]any)
	if !ok {
		return nil
	}
	delete(node, "prefixItems")
	// In draft-07 an array of schemas under "items" positions them, and
	// "additionalItems" governs the rest.
	rest, hadRest := node["items"]
	node["items"] = prefix
	if hadRest {
		node["additionalItems"] = rest
	} else {
		node["additionalItems"] = false
	}
	return nil
}

// downgradeToOpenAPI30 converts to the OpenAPI 3.0 schema subset: no tuples, no
// null type, no const, and boolean exclusive bounds.
func downgradeToOpenAPI30(node map[string]any, o ToJSONSchemaOpts) error {
	if _, ok := node["prefixItems"]; ok && tupleUnsupportedIn[o.Target] {
		if o.Unrepresentable != "any" {
			return fmt.Errorf("go-z: ToJSONSchema: a tuple cannot be represented in %s "+
				"(set Unrepresentable: \"any\" to emit an unconstrained array instead)", o.Target)
		}
		delete(node, "prefixItems")
		delete(node, "items")
	}

	// const → single-value enum.
	if v, ok := node["const"]; ok {
		delete(node, "const")
		node["enum"] = []any{v}
	}

	// type: [T, "null"] → type: T + nullable: true; a bare null type has no
	// OpenAPI 3.0 equivalent at all.
	switch t := node["type"].(type) {
	case []any:
		kept := make([]any, 0, len(t))
		nullable := false
		for _, entry := range t {
			if s, ok := entry.(string); ok && s == "null" {
				nullable = true
				continue
			}
			kept = append(kept, entry)
		}
		switch len(kept) {
		case 0:
			if err := unrepresentableInTarget(node, o, "null"); err != nil {
				return err
			}
		case 1:
			node["type"] = kept[0]
		default:
			// OpenAPI 3.0 has no type unions either; anyOf is the closest.
			variants := make([]any, 0, len(kept))
			for _, entry := range kept {
				variants = append(variants, map[string]any{"type": entry})
			}
			delete(node, "type")
			node["anyOf"] = variants
		}
		if nullable {
			node["nullable"] = true
		}
	case string:
		if t == "null" {
			if err := unrepresentableInTarget(node, o, "null"); err != nil {
				return err
			}
		}
	}

	// Numeric exclusive bounds became booleans alongside minimum/maximum. A
	// schema can carry both forms (Gte(0).Gt(5)), and OpenAPI 3.0 has one slot
	// for each side, so keep whichever bound is tighter rather than overwriting.
	if v, ok := node["exclusiveMinimum"]; ok {
		if _, isBool := v.(bool); !isBool {
			if inclusive, has := node["minimum"]; !has || boundIsLooser(inclusive, v, true) {
				node["minimum"] = v
				node["exclusiveMinimum"] = true
			} else {
				// The inclusive bound is the tighter one; drop the exclusive form.
				delete(node, "exclusiveMinimum")
			}
		}
	}
	if v, ok := node["exclusiveMaximum"]; ok {
		if _, isBool := v.(bool); !isBool {
			if inclusive, has := node["maximum"]; !has || boundIsLooser(inclusive, v, false) {
				node["maximum"] = v
				node["exclusiveMaximum"] = true
			} else {
				delete(node, "exclusiveMaximum")
			}
		}
	}
	return nil
}

// boundIsLooser reports whether the existing inclusive bound constrains less than
// the exclusive one, in which case the exclusive value should replace it. For a
// minimum, larger is tighter; for a maximum, smaller is tighter.
func boundIsLooser(inclusive, exclusive any, isMinimum bool) bool {
	cmp, ok := compareNumeric(inclusive, exclusive)
	if !ok {
		return true
	}
	if isMinimum {
		return cmp <= 0
	}
	return cmp >= 0
}

// unrepresentableInTarget applies the Unrepresentable policy to a node the target
// dialect cannot express.
func unrepresentableInTarget(node map[string]any, o ToJSONSchemaOpts, what string) error {
	if o.Unrepresentable != "any" {
		return fmt.Errorf("go-z: ToJSONSchema: %s cannot be represented in %s "+
			"(set Unrepresentable: \"any\" to emit {} instead)", what, o.Target)
	}
	delete(node, "type")
	node["nullable"] = true
	return nil
}
