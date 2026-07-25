package z

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateLiteralSchema validates strings that match a concatenated pattern of
// string parts and schema patterns (z.templateLiteral([...])).
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
		panic(fmt.Sprintf("go-z: TemplateLiteral: %v", err))
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
		// The same treatment a schema's pattern gets: anchors trimmed, the body
		// grouped, inner anchors reported.
		return patternFragment(x.String())
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

// patternFragment turns a standalone pattern into one that can be concatenated
// with its neighbours.
//
// The outer anchors are dropped because the fragment is anchored by the composed
// pattern, and the rest is wrapped in a non-capturing group so an alternation
// keeps its precedence: without the group, "a|b" spliced after a literal prefix
// would parse as "prefix-a" or "b".
//
// An anchor anywhere other than the outer edges cannot be honoured inside a
// larger pattern — "^cat$|^dog$" would silently reject "dog" — so it is reported
// instead of being mangled.
func patternFragment(src string) (string, error) {
	body := trimAnchors(src)
	if strings.ContainsAny(body, "^$") {
		if idx := indexUnescaped(body, "^$"); idx >= 0 {
			return "", fmt.Errorf("pattern %q anchors in the middle (at offset %d), "+
				"which cannot be embedded in a template literal; write it without inner "+
				"anchors, for example ^(?:cat|dog)$ instead of ^cat$|^dog$", src, idx+1)
		}
	}
	if body == "" {
		return "", nil
	}
	return "(?:" + body + ")", nil
}

// trimAnchors removes the outer ^ and $ of a pattern. A trailing "$" preceded by
// an odd number of backslashes is an escaped dollar sign, not an anchor: trimming
// it would leave a dangling backslash and an uncompilable pattern.
func trimAnchors(src string) string {
	body := strings.TrimPrefix(src, "^")
	if strings.HasSuffix(body, "$") && !isEscapedAt(body, len(body)-1) {
		body = body[:len(body)-1]
	}
	return body
}

// isEscapedAt reports whether the byte at i is escaped by the backslashes before it.
func isEscapedAt(s string, i int) bool {
	backslashes := 0
	for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
		backslashes++
	}
	return backslashes%2 == 1
}

// indexUnescaped returns the offset of the first character from chars that is not
// escaped by a backslash or inside a character class, or -1.
func indexUnescaped(s, chars string) int {
	inClass := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\':
			i++ // skip the escaped character
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case !inClass && strings.IndexByte(chars, c) >= 0:
			return i
		}
	}
	return -1
}

// checkPatternCoverage rejects a part whose validation cannot be expressed in
// the composed pattern. A template literal validates by matching one regexp, so
// a check with no regexp equivalent — Min, Refine, a transform — would be
// silently dropped, and the template would accept values the part rejects.
// Failing at construction is the honest alternative.
func checkPatternCoverage(in *Internals) error {
	if in == nil || in.Def == nil {
		return nil
	}
	total := len(in.Def.Checks)
	if total == 0 {
		return nil
	}
	if patternConflict(in) {
		return fmt.Errorf("template literal part has two independent patterns, " +
			"which cannot be combined into one; validate it outside the template")
	}
	if covered := patternChecks(in); covered < total {
		return fmt.Errorf("template literal part has %d check(s) with no pattern equivalent (%s); "+
			"a template validates by matching one pattern, so these would be ignored — "+
			"validate the field with its own schema instead",
			total-covered, strings.Join(uncoveredCheckNames(in), ", "))
	}
	return nil
}

// uncoveredCheckNames lists the checks that contributed no pattern, for the
// error message. Pattern-bearing checks are string and number formats.
func uncoveredCheckNames(in *Internals) []string {
	names := make([]string, 0, len(in.Def.Checks))
	for _, ch := range in.Def.Checks {
		if ch == nil {
			continue
		}
		switch ch.Name {
		case "string_format", "number_format":
			continue
		}
		name := ch.Name
		if name == "" {
			name = "custom"
		}
		names = append(names, name)
	}
	return names
}

func schemaPatternFragment(schema AnySchemaLike) (string, error) {
	if schema == nil {
		return "", fmt.Errorf("nil schema part")
	}
	in := schema.Internals()

	// Wrappers that widen the pattern: unwrap through the generic Unwrapper
	// interface so every optional/nullable instantiation is handled.
	if w, ok := schema.(Unwrapper); ok && in.Def != nil {
		switch in.Def.Type {
		case "optional":
			inner, err := schemaPatternFragment(w.Unwrap())
			if err != nil {
				return "", err
			}
			return "(?:" + inner + ")?", nil
		case "nullable":
			inner, err := schemaPatternFragment(w.Unwrap())
			if err != nil {
				return "", err
			}
			return "(?:" + inner + "|null)", nil
		case "lazy":
			return schemaPatternFragment(w.Unwrap())
		}
	}

	if err := checkPatternCoverage(in); err != nil {
		return "", err
	}

	if in.Pattern != nil {
		return patternFragment(in.Pattern.String())
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
			return patternFragment(in.Pattern.String())
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
