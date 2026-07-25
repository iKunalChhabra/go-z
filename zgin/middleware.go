package zgin

import (
	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-zod"
)

// ContextKey is the gin.Context key used by Validate / Get for the parsed value.
const ContextKey = "zod:value"

// Validate returns middleware that parses the JSON body with schema, stores the
// result under ContextKey, and calls the next handler — or aborts with 400.
func Validate(schema zod.AnySchemaLike) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := BindJSONAny(c, schema)
		if !ok {
			return
		}
		c.Set(ContextKey, v)
		c.Next()
	}
}

// Get returns the value stored by Validate middleware.
func Get(c *gin.Context) (any, bool) {
	return c.Get(ContextKey)
}
