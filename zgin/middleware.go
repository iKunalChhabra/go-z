package zgin

import (
	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
)

// ContextKey is the gin.Context key used by Validate / Get for the parsed value.
const ContextKey = "zod:value"

// Validate returns middleware that parses the JSON body with schema, stores the
// result under ContextKey, and calls the next handler — or aborts with an issue
// response. Pass BindOptions to change the body limit or Content-Type policy.
func Validate(schema z.AnySchemaLike, opts ...BindOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := BindJSONAny(c, schema, opts...)
		if !ok {
			return
		}
		c.Set(ContextKey, v)
		c.Next()
	}
}

// Get returns the value stored by Validate middleware as any.
func Get(c *gin.Context) (any, bool) {
	return c.Get(ContextKey)
}

// GetAs returns the value stored by Validate, type-asserted to T.
// Use with ToStruct pipelines, e.g. GetAs[User](c).
func GetAs[T any](c *gin.Context) (T, bool) {
	v, ok := c.Get(ContextKey)
	if !ok {
		var zero T
		return zero, false
	}
	out, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return out, true
}

// ValidateToStruct is like Validate but wraps schema with z.ToStruct[T] and
// stores the typed struct under ContextKey for GetAs[T].
func ValidateToStruct[T any](schema z.AnySchemaLike, opts ...BindOptions) gin.HandlerFunc {
	typed := z.ToStruct[T](schema)
	return func(c *gin.Context) {
		v, ok := BindJSON(c, typed, opts...)
		if !ok {
			return
		}
		c.Set(ContextKey, v)
		c.Next()
	}
}
