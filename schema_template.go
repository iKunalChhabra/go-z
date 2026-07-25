package zod

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateLiteralSchema validates strings that match a concatenated pattern of
// string parts and schema patterns (Zod's z.templateLiteral([...])).
type TemplateLiteralSchema struct {
	schemaBase[string]
	def     *Def
	parts   []any
	pattern *regexp.Regexp
}

// TemplateLiteral builds a template-literal schema from alternating string
// literals and string-producing schemas (String, Enum, Literal, Number, Bool,
// or anything with Internals().Pattern).
//
//	TemplateLiteral([]any{"id-", String().Regex(reDigits)})
func TemplateLiteral(parts []any, params ...any) *TemplateLiteralSchema {
	p := normalizeParams(params)
	def := &Def{Type: "template_literal", Error: p.Error}
	pat, err := compileTemplatePattern(parts)
	if err != nil {
		panic(fmt.Sprintf("zod: TemplateLiteral: %v", err))
	}
	return newTemplateLiteral(def, append([]any(nil), parts...), pat)
}

func newTemplateLiteral(def *Def, parts []any, pat *regexp.Regexp) *TemplateLiteralSchema {
	s := &TemplateLiteralSchema{def: def, parts: parts, pattern: pat}
	in := buildInternals(def, makeTemplateLiteralParse(def, pat))
	in.Pattern = pat
	s.schemaBase = newBase[string](in)
	return s
}

func makeTemplateLiteralParse(def *Def, pat *regexp.Regexp) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		str, ok := p.Value.(string)
		if !ok || !pat.MatchString(str) {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "template_literal",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}
		p.Value = str
	}
}

func compileTemplatePattern(parts []any) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for _, part := range parts {
		frag, err := templatePartFragment(part)
		if err != nil {
			return nil, err
		}
		if frag == "" {
			continue
		}
		b.WriteString("(?:")
		b.WriteString(frag)
		b.WriteString(")")
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func templatePartFragment(part any) (string, error) {
	switch x := part.(type) {
	case string:
		return regexp.QuoteMeta(x), nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case nil:
		return "null", nil
	case int:
		return regexp.QuoteMeta(fmt.Sprintf("%d", x)), nil
	case int64:
		return regexp.QuoteMeta(fmt.Sprintf("%d", x)), nil
	case float64:
		return regexp.QuoteMeta(fmt.Sprintf("%v", x)), nil
	case float32:
		return regexp.QuoteMeta(fmt.Sprintf("%v", x)), nil
	case *regexp.Regexp:
		return x.String(), nil
	case AnySchemaLike:
		return schemaPatternFragment(x)
	default:
		// Allow concrete schema pointers that don't go through the interface
		// in the type switch (already covered by AnySchemaLike).
		if s, ok := part.(AnySchemaLike); ok {
			return schemaPatternFragment(s)
		}
		return "", fmt.Errorf("unsupported part type %T", part)
	}
}

func schemaPatternFragment(schema AnySchemaLike) (string, error) {
	if schema == nil {
		return "", fmt.Errorf("nil schema part")
	}
	// Unwrap wrappers that affect optionality / nullability in the pattern.
	switch s := schema.(type) {
	case *OptionalSchema:
		inner, err := schemaPatternFragment(s.inner)
		if err != nil {
			return "", err
		}
		return "(?:" + inner + ")?", nil
	case *NullableSchema:
		inner, err := schemaPatternFragment(s.inner)
		if err != nil {
			return "", err
		}
		return "(?:" + inner + "|null)", nil
	case *TemplateLiteralSchema:
		if s.pattern != nil {
			// Strip ^...$ anchors from nested template.
			src := s.pattern.String()
			src = strings.TrimPrefix(src, "^")
			src = strings.TrimSuffix(src, "$")
			return src, nil
		}
	case *LazySchema:
		return schemaPatternFragment(s.Inner())
	}

	in := schema.Internals()
	if in.Pattern != nil {
		src := in.Pattern.String()
		src = strings.TrimPrefix(src, "^")
		src = strings.TrimSuffix(src, "$")
		return src, nil
	}
	if in.Values != nil && len(in.Values) > 0 {
		alts := make([]string, 0, len(in.Values))
		for v := range in.Values {
			if IsMissing(v) {
				alts = append(alts, "")
				continue
			}
			switch tv := v.(type) {
			case string:
				alts = append(alts, regexp.QuoteMeta(tv))
			case bool:
				if tv {
					alts = append(alts, "true")
				} else {
					alts = append(alts, "false")
				}
			case nil:
				alts = append(alts, "null")
			case float64:
				alts = append(alts, regexp.QuoteMeta(fmt.Sprintf("%v", tv)))
			case int:
				alts = append(alts, regexp.QuoteMeta(fmt.Sprintf("%d", tv)))
			case int64:
				alts = append(alts, regexp.QuoteMeta(fmt.Sprintf("%d", tv)))
			default:
				alts = append(alts, regexp.QuoteMeta(fmt.Sprintf("%v", tv)))
			}
		}
		return strings.Join(alts, "|"), nil
	}
	if in.Def == nil {
		return "", fmt.Errorf("schema has no def for template pattern")
	}
	switch in.Def.Type {
	case "string":
		return `[\s\S]*`, nil
	case "number", "int64":
		return `-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?`, nil
	case "boolean":
		return `true|false`, nil
	case "null":
		return `null`, nil
	case "undefined", "void":
		return ``, nil
	case "bigint":
		return `-?(?:0|[1-9]\d*)`, nil
	case "enum", "literal":
		return "", fmt.Errorf("enum/literal without Values")
	case "template_literal":
		if in.Pattern != nil {
			src := in.Pattern.String()
			src = strings.TrimPrefix(src, "^")
			src = strings.TrimSuffix(src, "$")
			return src, nil
		}
		return "", fmt.Errorf("template_literal without pattern")
	default:
		return "", fmt.Errorf("schema type %q has no pattern for template literals", in.Def.Type)
	}
}

// Parts returns a copy of the template parts.
func (s *TemplateLiteralSchema) Parts() []any {
	return append([]any(nil), s.parts...)
}

// Check attaches raw checks.
func (s *TemplateLiteralSchema) Check(checks ...*Check) *TemplateLiteralSchema {
	return newTemplateLiteral(s.def.withChecks(checks...), s.parts, s.pattern)
}
