package zod

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// hasLength is the When gate for length checks (Zod: value not nullish and
// has a .length property). Strings and slices/arrays qualify.
func hasLength(p *Payload) bool {
	switch v := p.Value.(type) {
	case string:
		return true
	case []any:
		_ = v
		return true
	default:
		return false
	}
}

// stringLen returns the Zod/JS-compatible length (UTF-16 code units).
func stringLen(s string) int {
	return utf16Len(s)
}

// utf16Len returns the number of UTF-16 code units (JS string.length).
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func lengthOf(v any) (int, bool) {
	switch x := v.(type) {
	case string:
		return stringLen(x), true
	case []any:
		return len(x), true
	default:
		return 0, false
	}
}

// MinLength ports $ZodCheckMinLength with Origin="string".
func MinLength(minimum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "min_length",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				curr, _ := in.Bag["minimum"].(int)
				if minimum > curr {
					in.Bag["minimum"] = minimum
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n >= minimum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooSmall,
			Origin:    "string",
			Minimum:   minimum,
			Inclusive: true,
			Input:     payload.Value,
		}))
	}
	return ch
}

// MaxLength ports $ZodCheckMaxLength with Origin="string".
func MaxLength(maximum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "max_length",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				if curr, ok := in.Bag["maximum"].(int); ok {
					if maximum < curr {
						in.Bag["maximum"] = maximum
					}
				} else {
					in.Bag["maximum"] = maximum
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n <= maximum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooBig,
			Origin:    "string",
			Maximum:   maximum,
			Inclusive: true,
			Input:     payload.Value,
		}))
	}
	return ch
}

// LengthEquals ports $ZodCheckLengthEquals with Origin="string".
func LengthEquals(length int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "length_equals",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["minimum"] = length
				in.Bag["maximum"] = length
				in.Bag["length"] = length
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n == length {
			return
		}
		iss := Issue{
			Origin:    "string",
			Inclusive: true,
			Exact:     true,
			Input:     payload.Value,
		}
		if n > length {
			iss.Code = IssueTooBig
			iss.Maximum = length
		} else {
			iss.Code = IssueTooSmall
			iss.Minimum = length
		}
		payload.AddIssue(ch.Issue(iss))
	}
	return ch
}

// Regex ports $ZodCheckRegex.
func Regex(pattern *regexp.Regexp, params ...any) *Check {
	p := normalizeParams(params)
	patStr := jsPattern(pattern)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "regex"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		if pattern.MatchString(s) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:    IssueInvalidFormat,
			Origin:  "string",
			Format:  "regex",
			Pattern: patStr,
			Input:   payload.Value,
		}))
	}
	return ch
}

// Includes ports $ZodCheckIncludes. An int among params sets the start position.
func Includes(includes string, params ...any) *Check {
	position := -1
	filtered := make([]any, 0, len(params))
	for _, x := range params {
		if n, ok := x.(int); ok {
			position = n
			continue
		}
		filtered = append(filtered, x)
	}
	p := normalizeParams(filtered)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "includes"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		if position < 0 {
			if strings.Contains(s, includes) {
				return
			}
		} else {
			if position <= len(s) && strings.Contains(s[position:], includes) {
				return
			}
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:     IssueInvalidFormat,
			Origin:   "string",
			Format:   "includes",
			Includes: includes,
			Input:    payload.Value,
		}))
	}
	return ch
}

// StartsWith ports $ZodCheckStartsWith.
func StartsWith(prefix string, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "starts_with"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || strings.HasPrefix(s, prefix) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Origin: "string",
			Format: "starts_with",
			Prefix: prefix,
			Input:  payload.Value,
		}))
	}
	return ch
}

// EndsWith ports $ZodCheckEndsWith.
func EndsWith(suffix string, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "ends_with"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || strings.HasSuffix(s, suffix) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Origin: "string",
			Format: "ends_with",
			Suffix: suffix,
			Input:  payload.Value,
		}))
	}
	return ch
}

// LowerCase ports $ZodCheckLowerCase (format="lowercase").
func LowerCase(params ...any) *Check {
	return stringFormatPattern("lowercase", reLowercase, jsPattern(reLowercase), params...)
}

// UpperCase ports $ZodCheckUpperCase (format="uppercase").
func UpperCase(params ...any) *Check {
	return stringFormatPattern("uppercase", reUppercase, jsPattern(reUppercase), params...)
}

// Overwrite ports $ZodCheckOverwrite: mutates p.Value in place.
func Overwrite(tx func(string) string) *Check {
	ch := &Check{Name: "overwrite"}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		payload.Value = tx(s)
	}
	return ch
}

// Trim is an overwrite check (Zod checks.trim).
func Trim() *Check { return Overwrite(strings.TrimSpace) }

// ToLowerCase is an overwrite check.
func ToLowerCase() *Check {
	return Overwrite(func(s string) string { return strings.ToLower(s) })
}

// ToUpperCase is an overwrite check.
func ToUpperCase() *Check {
	return Overwrite(func(s string) string { return strings.ToUpper(s) })
}

// NormalizeNFC is an overwrite check applying Unicode NFC normalization.
func NormalizeNFC() *Check {
	return Overwrite(func(s string) string { return norm.NFC.String(s) })
}

// stringFormatPattern builds a pattern-based invalid_format check.
func stringFormatPattern(format string, re *regexp.Regexp, patternLiteral string, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = format
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		if re != nil && re.MatchString(s) {
			return
		}
		iss := Issue{
			Code:   IssueInvalidFormat,
			Origin: "string",
			Format: format,
			Input:  payload.Value,
		}
		if patternLiteral != "" {
			iss.Pattern = patternLiteral
		}
		payload.AddIssue(ch.Issue(iss))
	}
	return ch
}

// stringFormatFn builds a predicate-based invalid_format check.
func stringFormatFn(format, patternLiteral string, match func(string) bool, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = format
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		if match(s) {
			return
		}
		iss := Issue{
			Code:   IssueInvalidFormat,
			Origin: "string",
			Format: format,
			Input:  payload.Value,
		}
		if patternLiteral != "" {
			iss.Pattern = patternLiteral
		}
		payload.AddIssue(ch.Issue(iss))
	}
	return ch
}
