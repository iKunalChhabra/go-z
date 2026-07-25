package zgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
)

// ErrorFormat selects how Error is rendered in the HTTP response body.
type ErrorFormat int

const (
	// FormatIssues is the default HTTP error shape:
	// {"success":false,"error":{"issues":[...]}}
	FormatIssues ErrorFormat = iota
	// FormatFlatten uses z.Flatten (formErrors + fieldErrors).
	FormatFlatten
	// FormatTree uses z.Treeify (nested errors/properties/items).
	FormatTree
	// FormatPretty uses Error.Error() (indented issues JSON) or Prettify.
	FormatPretty
)

// Options configures AbortWithError.
type Options struct {
	// Status is the HTTP status code (default 400).
	Status int
	// Format selects the error body shape (default FormatIssues).
	Format ErrorFormat
}

// AbortWithError writes an issue-shaped error response and aborts the Gin context.
func AbortWithError(c *gin.Context, err *z.Error, opts Options) {
	if err == nil {
		err = &z.Error{Issues: nil}
	}
	status := opts.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	c.AbortWithStatusJSON(status, renderError(err, opts.Format))
}

func renderError(err *z.Error, format ErrorFormat) gin.H {
	switch format {
	case FormatFlatten:
		flat := z.Flatten(err)
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
			"error":   z.Treeify(err),
		}
	case FormatPretty:
		return gin.H{
			"success": false,
			"error":   z.Prettify(err),
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
