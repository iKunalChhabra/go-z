package z

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// FlattenedError mirrors flattenError output (string messages).
type FlattenedError struct {
	FormErrors  []string            `json:"formErrors"`
	FieldErrors map[string][]string `json:"fieldErrors"`
}

// FlattenedErrorU is flattenError with a custom mapped issue type.
type FlattenedErrorU[U any] struct {
	FormErrors  []U            `json:"formErrors"`
	FieldErrors map[string][]U `json:"fieldErrors"`
}

// ErrorTree mirrors treeifyError output (string messages).
type ErrorTree struct {
	Errors     []string              `json:"errors"`
	Properties map[string]*ErrorTree `json:"properties,omitempty"`
	Items      []*ErrorTree          `json:"items,omitempty"`
}

// ErrorTreeU is treeifyError with a custom mapped issue type.
type ErrorTreeU[U any] struct {
	Errors     []U                       `json:"errors"`
	Properties map[string]*ErrorTreeU[U] `json:"properties,omitempty"`
	Items      []*ErrorTreeU[U]          `json:"items,omitempty"`
}

// Flatten ports flattenError: form-level issues (empty path) vs first-segment field errors.
func Flatten(err *Error) FlattenedError {
	mapped := FlattenMap(err, func(iss Issue) string { return iss.Message })
	return FlattenedError{FormErrors: mapped.FormErrors, FieldErrors: mapped.FieldErrors}
}

// FlattenMap ports flattenError with a custom mapper.
func FlattenMap[U any](err *Error, mapper func(Issue) U) FlattenedErrorU[U] {
	out := FlattenedErrorU[U]{
		FormErrors:  []U{},
		FieldErrors: map[string][]U{},
	}
	if err == nil {
		return out
	}
	for _, sub := range err.Issues {
		if len(sub.Path) > 0 {
			key := pathSegString(sub.Path[0])
			out.FieldErrors[key] = append(out.FieldErrors[key], mapper(sub))
		} else {
			out.FormErrors = append(out.FormErrors, mapper(sub))
		}
	}
	return out
}

// Format ports formatError into a nested map with "_errors" arrays at each node.
func Format(err *Error) map[string]any {
	return FormatMap(err, func(iss Issue) string { return iss.Message })
}

// FormatMap ports formatError with a custom mapper. Nested invalid_union /
// invalid_key / invalid_element issues are traversed too.
func FormatMap[U any](err *Error, mapper func(Issue) U) map[string]any {
	fieldErrors := map[string]any{"_errors": []U{}}
	if err == nil {
		return fieldErrors
	}
	var process func(issues []Issue, path []any)
	process = func(issues []Issue, path []any) {
		for _, issue := range issues {
			switch {
			case issue.Code == IssueInvalidUnion && len(issue.Errors) > 0:
				base := appendPath(path, issue.Path)
				for _, nested := range issue.Errors {
					process(nested, base)
				}
			case issue.Code == IssueInvalidKey:
				process(issue.Issues, appendPath(path, issue.Path))
			case issue.Code == IssueInvalidElement:
				process(issue.Issues, appendPath(path, issue.Path))
			default:
				fullpath := appendPath(path, issue.Path)
				if len(fullpath) == 0 {
					fieldErrors["_errors"] = append(asSliceU[U](fieldErrors["_errors"]), mapper(issue))
					continue
				}
				curr := fieldErrors
				for i, el := range fullpath {
					key := pathSegString(el)
					child, ok := curr[key].(map[string]any)
					if !ok {
						child = map[string]any{"_errors": []U{}}
						curr[key] = child
					}
					if i == len(fullpath)-1 {
						child["_errors"] = append(asSliceU[U](child["_errors"]), mapper(issue))
					}
					curr = child
				}
			}
		}
	}
	process(err.Issues, nil)
	return fieldErrors
}

// Treeify ports treeifyError into a typed error tree (string messages).
func Treeify(err *Error) *ErrorTree {
	result := &ErrorTree{Errors: []string{}}
	if err == nil {
		return result
	}
	processTree(err.Issues, nil, result, func(iss Issue) string { return iss.Message })
	return result
}

// TreeifyMap ports treeifyError with a custom mapper.
func TreeifyMap[U any](err *Error, mapper func(Issue) U) *ErrorTreeU[U] {
	result := &ErrorTreeU[U]{Errors: []U{}}
	if err == nil {
		return result
	}
	processTreeU(err.Issues, nil, result, mapper)
	return result
}

func processTree(issues []Issue, path []any, result *ErrorTree, mapper func(Issue) string) {
	for _, issue := range issues {
		switch {
		case issue.Code == IssueInvalidUnion && len(issue.Errors) > 0:
			base := appendPath(path, issue.Path)
			for _, nested := range issue.Errors {
				processTree(nested, base, result, mapper)
			}
		case issue.Code == IssueInvalidKey:
			processTree(issue.Issues, appendPath(path, issue.Path), result, mapper)
		case issue.Code == IssueInvalidElement:
			processTree(issue.Issues, appendPath(path, issue.Path), result, mapper)
		default:
			fullpath := appendPath(path, issue.Path)
			if len(fullpath) == 0 {
				result.Errors = append(result.Errors, mapper(issue))
				continue
			}
			curr := result
			for i, el := range fullpath {
				terminal := i == len(fullpath)-1
				if idx := pathSegIndex(el); idx >= 0 {
					if curr.Items == nil {
						curr.Items = []*ErrorTree{}
					}
					for len(curr.Items) <= idx {
						curr.Items = append(curr.Items, nil)
					}
					if curr.Items[idx] == nil {
						curr.Items[idx] = &ErrorTree{Errors: []string{}}
					}
					curr = curr.Items[idx]
				} else {
					if curr.Properties == nil {
						curr.Properties = map[string]*ErrorTree{}
					}
					key := pathSegString(el)
					if curr.Properties[key] == nil {
						curr.Properties[key] = &ErrorTree{Errors: []string{}}
					}
					curr = curr.Properties[key]
				}
				if terminal {
					curr.Errors = append(curr.Errors, mapper(issue))
				}
			}
		}
	}
}

func processTreeU[U any](issues []Issue, path []any, result *ErrorTreeU[U], mapper func(Issue) U) {
	for _, issue := range issues {
		switch {
		case issue.Code == IssueInvalidUnion && len(issue.Errors) > 0:
			base := appendPath(path, issue.Path)
			for _, nested := range issue.Errors {
				processTreeU(nested, base, result, mapper)
			}
		case issue.Code == IssueInvalidKey:
			processTreeU(issue.Issues, appendPath(path, issue.Path), result, mapper)
		case issue.Code == IssueInvalidElement:
			processTreeU(issue.Issues, appendPath(path, issue.Path), result, mapper)
		default:
			fullpath := appendPath(path, issue.Path)
			if len(fullpath) == 0 {
				result.Errors = append(result.Errors, mapper(issue))
				continue
			}
			curr := result
			for i, el := range fullpath {
				terminal := i == len(fullpath)-1
				if idx := pathSegIndex(el); idx >= 0 {
					if curr.Items == nil {
						curr.Items = []*ErrorTreeU[U]{}
					}
					for len(curr.Items) <= idx {
						curr.Items = append(curr.Items, nil)
					}
					if curr.Items[idx] == nil {
						curr.Items[idx] = &ErrorTreeU[U]{Errors: []U{}}
					}
					curr = curr.Items[idx]
				} else {
					if curr.Properties == nil {
						curr.Properties = map[string]*ErrorTreeU[U]{}
					}
					key := pathSegString(el)
					if curr.Properties[key] == nil {
						curr.Properties[key] = &ErrorTreeU[U]{Errors: []U{}}
					}
					curr = curr.Properties[key]
				}
				if terminal {
					curr.Errors = append(curr.Errors, mapper(issue))
				}
			}
		}
	}
}

// Prettify ports prettifyError: sort by path length, then
// "✖ message\n  → at path".
func Prettify(err *Error) string {
	if err == nil || len(err.Issues) == 0 {
		return ""
	}
	issues := make([]Issue, len(err.Issues))
	copy(issues, err.Issues)
	sort.SliceStable(issues, func(i, j int) bool {
		return len(issues[i].Path) < len(issues[j].Path)
	})
	var lines []string
	for _, issue := range issues {
		lines = append(lines, "✖ "+issue.Message)
		if len(issue.Path) > 0 {
			lines = append(lines, "  → at "+ToDotPath(issue.Path))
		}
	}
	return strings.Join(lines, "\n")
}

var nonIdentSeg = regexp.MustCompile(`[^\w$]`)

// ToDotPath ports toDotPath: a.b[0].c / ["weird.key"] / ["Symbol(x)"].
func ToDotPath(path []any) string {
	normalized := make([]any, 0, len(path))
	for _, seg := range path {
		normalized = append(normalized, unwrapPathSeg(seg))
	}
	var segs []string
	for _, seg := range normalized {
		switch v := seg.(type) {
		case int:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case int8:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case int16:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case int32:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case int64:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case uint:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case uint8:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case uint16:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case uint32:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case uint64:
			segs = append(segs, fmt.Sprintf("[%d]", v))
		case float64:
			if v == float64(int64(v)) {
				segs = append(segs, fmt.Sprintf("[%d]", int64(v)))
			} else {
				segs = append(segs, bracketJSON(fmt.Sprint(v)))
			}
		case string:
			if nonIdentSeg.MatchString(v) {
				segs = append(segs, bracketJSON(v))
			} else {
				if len(segs) > 0 {
					segs = append(segs, ".")
				}
				segs = append(segs, v)
			}
		default:
			segs = append(segs, bracketJSON(fmt.Sprint(v)))
		}
	}
	return strings.Join(segs, "")
}

func bracketJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `["` + s + `"]`
	}
	return "[" + string(b) + "]"
}

func unwrapPathSeg(seg any) any {
	switch v := seg.(type) {
	case map[string]any:
		if key, ok := v["key"]; ok {
			return key
		}
	}
	return seg
}

func pathSegString(seg any) string {
	seg = unwrapPathSeg(seg)
	switch v := seg.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return formatFloat(v)
	default:
		return fmt.Sprint(v)
	}
}

// pathSegIndex returns a non-negative array index when seg is numeric, else -1.
func pathSegIndex(seg any) int {
	seg = unwrapPathSeg(seg)
	switch v := seg.(type) {
	case int:
		if v >= 0 {
			return v
		}
	case int8:
		if v >= 0 {
			return int(v)
		}
	case int16:
		if v >= 0 {
			return int(v)
		}
	case int32:
		if v >= 0 {
			return int(v)
		}
	case int64:
		if v >= 0 {
			return int(v)
		}
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		if v >= 0 && v == float64(int(v)) {
			return int(v)
		}
	}
	return -1
}

func appendPath(base, extra []any) []any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make([]any, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

func asSliceU[U any](v any) []U {
	if v == nil {
		return []U{}
	}
	if s, ok := v.([]U); ok {
		return s
	}
	return []U{}
}
