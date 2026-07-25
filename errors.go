package zod

import "errors"

// ZodError is the error returned when parsing fails. Like Zod's ZodError, its
// message is the pretty-printed JSON of its issues.
type ZodError struct {
	Issues []Issue
}

// Error implements the error interface; matches Zod's ZodError.message
// (JSON.stringify(issues, null, 2)).
func (e *ZodError) Error() string {
	return marshalIssues(e.Issues)
}

// AsZodError extracts a *ZodError from err, unwrapping wrapped errors
// (fmt.Errorf("%w", …), joined errors, custom wrappers). Prefer it over a
// bare type assertion, which fails the moment a caller wraps the error:
//
//	if zerr, ok := z.AsZodError(err); ok {
//	    return c.JSON(400, z.Flatten(zerr))
//	}
func AsZodError(err error) (*ZodError, bool) {
	var zerr *ZodError
	if errors.As(err, &zerr) {
		return zerr, true
	}
	return nil, false
}

// IsZodError reports whether err is (or wraps) a validation error.
func IsZodError(err error) bool {
	_, ok := AsZodError(err)
	return ok
}

// SafeParseResult mirrors Zod's safeParse return value.
type SafeParseResult[T any] struct {
	Success bool
	Data    T
	Error   *ZodError
}

// newZodError finalizes raw issues into a ZodError.
func newZodError(raw []Issue, ctx *ParseCtx) *ZodError {
	cfg := GetConfig()
	issues := make([]Issue, len(raw))
	for i, iss := range raw {
		issues[i] = FinalizeIssue(iss, ctx, cfg)
	}
	return &ZodError{Issues: issues}
}
