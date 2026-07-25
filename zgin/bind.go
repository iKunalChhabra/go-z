package zgin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-zod"
)

// BindJSON parses c.Request.Body as JSON into any, validates with schema.
// On failure writes 400 + Zod issue JSON and returns (zero, false).
func BindJSON[T any](c *gin.Context, schema zod.Schema[T]) (T, bool) {
	var zero T
	data, ok := readJSONBody(c)
	if !ok {
		return zero, false
	}
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err)
		return zero, false
	}
	return v, true
}

// BindJSONAny parses the JSON body and validates with an untyped schema
// (AnySchemaLike), returning map/any output.
func BindJSONAny(c *gin.Context, schema zod.AnySchemaLike) (any, bool) {
	data, ok := readJSONBody(c)
	if !ok {
		return nil, false
	}
	v, err := parseAny(schema, data)
	if err != nil {
		abortParseError(c, err)
		return nil, false
	}
	return v, true
}

// BindQuery binds URL query parameters (with CoerceQueryValues) into schema.
func BindQuery[T any](c *gin.Context, schema zod.Schema[T]) (T, bool) {
	var zero T
	if c.Request == nil {
		abortParseError(c, &zod.ZodError{Issues: []zod.Issue{{
			Code:    zod.IssueCustom,
			Message: "missing request",
			Path:    []any{},
		}}})
		return zero, false
	}
	raw := c.Request.URL.Query()
	data := CoerceQueryValues(raw)
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err)
		return zero, false
	}
	return v, true
}

// BindURI binds Gin URI parameters into schema (values are strings; use
// zod.Coerce.* field schemas for numeric/bool coercion).
func BindURI[T any](c *gin.Context, schema zod.Schema[T]) (T, bool) {
	var zero T
	data := make(map[string]any, len(c.Params))
	for _, p := range c.Params {
		data[p.Key] = p.Value
	}
	v, err := schema.Parse(data)
	if err != nil {
		abortParseError(c, err)
		return zero, false
	}
	return v, true
}

func readJSONBody(c *gin.Context) (any, bool) {
	if c.Request == nil || c.Request.Body == nil {
		AbortWithError(c, &zod.ZodError{Issues: []zod.Issue{{
			Code:    zod.IssueCustom,
			Message: "missing request body",
			Path:    []any{},
		}}}, Options{Status: http.StatusBadRequest})
		return nil, false
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		AbortWithError(c, &zod.ZodError{Issues: []zod.Issue{{
			Code:    zod.IssueCustom,
			Message: "failed to read request body",
			Path:    []any{},
		}}}, Options{Status: http.StatusBadRequest})
		return nil, false
	}
	if len(body) == 0 {
		AbortWithError(c, &zod.ZodError{Issues: []zod.Issue{{
			Code:    zod.IssueCustom,
			Message: "empty request body",
			Path:    []any{},
		}}}, Options{Status: http.StatusBadRequest})
		return nil, false
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		AbortWithError(c, &zod.ZodError{Issues: []zod.Issue{{
			Code:    zod.IssueCustom,
			Message: "invalid JSON: " + err.Error(),
			Path:    []any{},
		}}}, Options{Status: http.StatusBadRequest})
		return nil, false
	}
	return data, true
}

func abortParseError(c *gin.Context, err error) {
	var zerr *zod.ZodError
	if errors.As(err, &zerr) {
		AbortWithError(c, zerr, Options{})
		return
	}
	AbortWithError(c, &zod.ZodError{Issues: []zod.Issue{{
		Code:    zod.IssueCustom,
		Message: err.Error(),
		Path:    []any{},
	}}}, Options{})
}

// parseAny runs an AnySchemaLike through a typed any identity wrapper so we
// can use the public Parse surface without accessing unexported helpers.
func parseAny(schema zod.AnySchemaLike, data any) (any, error) {
	return zod.TransformTo[any](schema, func(v any) (any, error) {
		return v, nil
	}).Parse(data)
}
