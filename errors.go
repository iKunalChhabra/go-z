package zod

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
