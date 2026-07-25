package zgin

import (
	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
)

// ContextKey is the gin.Context key used by Validate / Get for the parsed value.
const ContextKey = "go-z:value"

// Validate returns middleware that parses the JSON body with schema, stores the
// result under ContextKey, and calls the next handler — or aborts with an issue
// response. Pass BindOptions to change the body limit or Content-Type policy.
func Validate(schema z.AnySchemaLike, opts ...BindOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := BindJSONAny(c, schema, opts...)
		if !ok {
			return
		}
		c.Set(firstBindOptions(opts).contextKey(), v)
		c.Next()
	}
}

// Get returns the value stored by Validate middleware as any.
func Get(c *gin.Context) (any, bool) {
	return c.Get(ContextKey)
}

// GetFrom returns the value a Validate middleware stored under a custom
// BindOptions.ContextKey.
func GetFrom(c *gin.Context, key string) (any, bool) {
	if key == "" {
		key = ContextKey
	}
	return c.Get(key)
}

// GetAsFrom is GetAs for a custom BindOptions.ContextKey.
func GetAsFrom[T any](c *gin.Context, key string) (T, bool) {
	var zero T
	v, ok := GetFrom(c, key)
	if !ok {
		return zero, false
	}
	out, ok := v.(T)
	if !ok {
		return zero, false
	}
	return out, true
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
		c.Set(firstBindOptions(opts).contextKey(), v)
		c.Next()
	}
}
