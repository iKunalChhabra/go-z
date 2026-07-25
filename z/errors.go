package z

import "errors"

// Error is the error returned when parsing fails. Like Error, its
// message is the pretty-printed JSON of its issues.
type Error struct {
	Issues []Issue
}

// Error implements the error interface; matches Error.message
// (JSON.stringify(issues, null, 2)).
func (e *Error) Error() string {
	return marshalIssues(e.Issues)
}

// AsError extracts a *Error from err, unwrapping wrapped errors
// (fmt.Errorf("%w", …), joined errors, custom wrappers). Prefer it over a
// bare type assertion, which fails the moment a caller wraps the error:
//
//	if zerr, ok := z.AsError(err); ok {
//	    return c.JSON(400, z.Flatten(zerr))
//	}
func AsError(err error) (*Error, bool) {
	var zerr *Error
	if errors.As(err, &zerr) {
		return zerr, true
	}
	return nil, false
}

// IsError reports whether err is (or wraps) a validation error.
func IsError(err error) bool {
	_, ok := AsError(err)
	return ok
}

// SafeParseResult mirrors safeParse return value.
type SafeParseResult[T any] struct {
	Success bool
	Data    T
	Error   *Error
}

// newError finalizes raw issues into a Error.
func newError(raw []Issue, ctx *ParseCtx) *Error {
	cfg := GetConfig()
	issues := make([]Issue, len(raw))
	for i, iss := range raw {
		issues[i] = FinalizeIssue(iss, ctx, cfg)
	}
	return &Error{Issues: issues}
}
