package z

import (
	"encoding/json"
)

// IssueCode enumerates issue taxonomy. Values match v4 exactly.
type IssueCode string

const (
	IssueInvalidType      IssueCode = "invalid_type"
	IssueTooBig           IssueCode = "too_big"
	IssueTooSmall         IssueCode = "too_small"
	IssueInvalidFormat    IssueCode = "invalid_format"
	IssueNotMultipleOf    IssueCode = "not_multiple_of"
	IssueUnrecognizedKeys IssueCode = "unrecognized_keys"
	IssueInvalidUnion     IssueCode = "invalid_union"
	IssueInvalidKey       IssueCode = "invalid_key"
	IssueInvalidElement   IssueCode = "invalid_element"
	IssueInvalidValue     IssueCode = "invalid_value"
	IssueCustom           IssueCode = "custom"
)

// AllIssueCodes is every issue code, in taxonomy order. Useful for exhaustive
// switches over the taxonomy and for locale coverage checks.
var AllIssueCodes = []IssueCode{
	IssueInvalidType,
	IssueTooBig,
	IssueTooSmall,
	IssueInvalidFormat,
	IssueNotMultipleOf,
	IssueUnrecognizedKeys,
	IssueInvalidUnion,
	IssueInvalidKey,
	IssueInvalidElement,
	IssueInvalidValue,
	IssueCustom,
}

// continueMode mirrors tri-state `continue` flag on raw issues:
// unset ⇒ issue aborts subsequent checks, true ⇒ checks continue,
// false ⇒ explicit abort (respected even by `when`-gated checks).
type continueMode int8

const (
	continueUnset continueMode = iota // like `continue: undefined` (aborts)
	continueYes                       // like `continue: true`
	continueNo                        // like `continue: false` (explicit abort)
)

// Issue is a single validation problem. It doubles as "raw issue"
// (produced during parsing, message possibly empty) and the finalized issue
// exposed on Error; FinalizeIssue converts the former into the latter.
//
// Only fields relevant to the Code are populated. JSON serialization matches
// issue objects.
type Issue struct {
	Code    IssueCode `json:"code"`
	Path    []any     `json:"path"`
	Message string    `json:"message"`
	// Input is the offending input value. During parsing it is always set
	// (error maps need it); FinalizeIssue clears it unless ParseCtx.ReportInput
	// is true, mirroring.
	Input any `json:"input,omitempty"`

	// invalid_type
	Expected string `json:"expected,omitempty"`
	// too_big / too_small / invalid_key / invalid_element origin:
	// "string" | "number" | "int" | "array" | "set" | "map" | "record" | "date" | ...
	Origin    string `json:"origin,omitempty"`
	Maximum   any    `json:"maximum,omitempty"`
	Minimum   any    `json:"minimum,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Exact     bool   `json:"exact,omitempty"`
	// invalid_format
	Format    string `json:"format,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Includes  string `json:"includes,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	// not_multiple_of
	Divisor float64 `json:"divisor,omitempty"`
	// unrecognized_keys
	Keys []string `json:"keys,omitempty"`
	// invalid_union
	Errors        [][]Issue `json:"errors,omitempty"`
	Discriminator string    `json:"discriminator,omitempty"`
	// invalid_key / invalid_element
	Issues []Issue `json:"issues,omitempty"`
	Key    any     `json:"key,omitempty"`
	// invalid_value
	Values []any `json:"values,omitempty"`
	// custom
	Params map[string]any `json:"params,omitempty"`

	// runtime-only fields (never serialized)
	cont   continueMode
	errMap ErrorMap // error map of the schema/check that produced this issue
}

// WithContinue marks the issue so subsequent checks still run
// (`continue: true`, set by non-aborting checks).
func (i Issue) WithContinue() Issue { i.cont = continueYes; return i }

// WithAbort marks the issue as an explicit abort (`continue: false`).
func (i Issue) WithAbort() Issue { i.cont = continueNo; return i }

// FinalizeIssue resolves the issue message through precedence chain:
// explicit message → producing schema/check error map → per-parse error map →
// global custom error map → locale error map → "Invalid input".
func FinalizeIssue(iss Issue, ctx *ParseCtx, cfg *Config) Issue {
	if iss.Message == "" {
		msg := ""
		if iss.errMap != nil {
			msg = iss.errMap(&iss)
		}
		if msg == "" && ctx != nil && ctx.Error != nil {
			msg = ctx.Error(&iss)
		}
		if msg == "" && cfg != nil && cfg.CustomError != nil {
			msg = cfg.CustomError(&iss)
		}
		if msg == "" {
			locale := EnLocale
			if cfg != nil && cfg.LocaleError != nil {
				locale = cfg.LocaleError
			}
			msg = locale(&iss)
		}
		if msg == "" {
			msg = "Invalid input"
		}
		iss.Message = msg
	}
	if iss.Path == nil {
		iss.Path = []any{}
	}
	if ctx == nil || !ctx.ReportInput {
		iss.Input = nil
	}
	iss.errMap = nil
	return iss
}

// marshalIssues renders issues the way Error.message does:
// JSON.stringify(issues, null, 2).
func marshalIssues(issues []Issue) string {
	b, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}
