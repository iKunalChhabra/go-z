package zgin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
)

// BindJSON parses c.Request.Body as JSON, validates it with schema, and writes
// an issue response on failure. The body is size-limited (DefaultMaxBodyBytes),
// its Content-Type is checked, and integers are decoded exactly; see BindOptions.
func BindJSON[T any](c *gin.Context, schema z.Schema[T], opts ...BindOptions) (T, bool) {
	return BindJSONWithOptions(c, schema, Options{}, opts...)
}

// BindJSONWithOptions is BindJSON with control over the error response:
// errOpts sets the HTTP status and body shape written when validation fails.
// Body-read failures (bad JSON, wrong Content-Type, oversize) keep their own
// transport statuses.
func BindJSONWithOptions[T any](c *gin.Context, schema z.Schema[T], errOpts Options, opts ...BindOptions) (T, bool) {
	var zero T
	data, ok := readJSONBody(c, firstBindOptions(opts))
	if !ok {
		return zero, false
	}
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err, errOpts)
		return zero, false
	}
	return v, true
}

// BindJSONAny parses the JSON body and validates with an untyped schema
// (AnySchemaLike), returning map/any output.
func BindJSONAny(c *gin.Context, schema z.AnySchemaLike, opts ...BindOptions) (any, bool) {
	return BindJSONAnyWithOptions(c, schema, Options{}, opts...)
}

// BindJSONAnyWithOptions is BindJSONAny with control over the error response;
// see BindJSONWithOptions.
func BindJSONAnyWithOptions(c *gin.Context, schema z.AnySchemaLike, errOpts Options, opts ...BindOptions) (any, bool) {
	data, ok := readJSONBody(c, firstBindOptions(opts))
	if !ok {
		return nil, false
	}
	v, err := z.ParseAny(schema, data)
	if err != nil {
		abortParseError(c, err, errOpts)
		return nil, false
	}
	return v, true
}

// BindQuery binds URL query parameters (with CoerceQueryValues) into schema.
func BindQuery[T any](c *gin.Context, schema z.Schema[T]) (T, bool) {
	return BindQueryWithOptions(c, schema, Options{})
}

// BindQueryWithOptions is BindQuery with control over the error response:
// errOpts sets the HTTP status and body shape written when validation fails.
func BindQueryWithOptions[T any](c *gin.Context, schema z.Schema[T], errOpts Options) (T, bool) {
	var zero T
	if c.Request == nil {
		abortParseError(c, &z.Error{Issues: []z.Issue{{
			Code:    z.IssueCustom,
			Message: "missing request",
			Path:    []any{},
		}}}, errOpts)
		return zero, false
	}
	raw := c.Request.URL.Query()
	// Array-typed fields get a slice even for a single value, so ?tag=a and
	// ?tag=a&tag=b both satisfy an array schema.
	data := CoerceQueryValuesFor(schema, raw)
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err, errOpts)
		return zero, false
	}
	return v, true
}

// BindURI binds Gin URI parameters into schema (values are strings; use
// z.Coerce.* field schemas for numeric/bool coercion).
func BindURI[T any](c *gin.Context, schema z.Schema[T]) (T, bool) {
	return BindURIWithOptions(c, schema, Options{})
}

// BindURIWithOptions is BindURI with control over the error response:
// errOpts sets the HTTP status and body shape written when validation fails.
func BindURIWithOptions[T any](c *gin.Context, schema z.Schema[T], errOpts Options) (T, bool) {
	var zero T
	data := make(map[string]any, len(c.Params))
	for _, p := range c.Params {
		data[p.Key] = p.Value
	}
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err, errOpts)
		return zero, false
	}
	return v, true
}

func abortParseError(c *gin.Context, err error, opts Options) {
	var zerr *z.Error
	if errors.As(err, &zerr) {
		AbortWithError(c, zerr, opts)
		return
	}
	AbortWithError(c, &z.Error{Issues: []z.Issue{{
		Code:    z.IssueCustom,
		Message: err.Error(),
		Path:    []any{},
	}}}, opts)
}
