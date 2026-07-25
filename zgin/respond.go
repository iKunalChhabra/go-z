package zgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-zod"
)

// ErrorFormat selects how ZodError is rendered in the HTTP response body.
type ErrorFormat int

const (
	// FormatIssues is the default Zod HTTP error shape:
	// {"success":false,"error":{"issues":[...]}}
	FormatIssues ErrorFormat = iota
	// FormatFlatten uses zod.Flatten (formErrors + fieldErrors).
	FormatFlatten
	// FormatTree uses zod.Treeify (nested errors/properties/items).
	FormatTree
	// FormatPretty uses ZodError.Error() (indented issues JSON) or Prettify.
	FormatPretty
)

// Options configures AbortWithError.
type Options struct {
	// Status is the HTTP status code (default 400).
	Status int
	// Format selects the error body shape (default FormatIssues).
	Format ErrorFormat
}

// AbortWithError writes a Zod-shaped error response and aborts the Gin context.
func AbortWithError(c *gin.Context, err *zod.ZodError, opts Options) {
	if err == nil {
		err = &zod.ZodError{Issues: nil}
	}
	status := opts.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	c.AbortWithStatusJSON(status, renderError(err, opts.Format))
}

func renderError(err *zod.ZodError, format ErrorFormat) gin.H {
	switch format {
	case FormatFlatten:
		flat := zod.Flatten(err)
		return gin.H{
			"success": false,
			"error": gin.H{
				"formErrors":  flat.FormErrors,
				"fieldErrors": flat.FieldErrors,
			},
		}
	case FormatTree:
		return gin.H{
			"success": false,
			"error":   zod.Treeify(err),
		}
	case FormatPretty:
		return gin.H{
			"success": false,
			"error":   zod.Prettify(err),
		}
	default: // FormatIssues
		return gin.H{
			"success": false,
			"error": gin.H{
				"issues": err.Issues,
			},
		}
	}
}
