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
	// FormatFlatten renders a minimal formErrors/fieldErrors object.
	// Full Flatten lands in WP-F; this is a local approximation.
	FormatFlatten
	// FormatTree renders a minimal path-keyed tree of messages.
	// Full Treeify lands in WP-F; this is a local approximation.
	FormatTree
	// FormatPretty uses ZodError.Error() (indented issues JSON).
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
		return gin.H{
			"success": false,
			"error":   flattenIssues(err.Issues),
		}
	case FormatTree:
		return gin.H{
			"success": false,
			"error":   treeifyIssues(err.Issues),
		}
	case FormatPretty:
		return gin.H{
			"success": false,
			"error":   err.Error(),
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

// flattenIssues is a minimal local flatten (WP-F will provide zod.Flatten).
// formErrors: messages with empty path; fieldErrors: keyed by first path segment.
func flattenIssues(issues []zod.Issue) gin.H {
	formErrors := make([]string, 0)
	fieldErrors := map[string][]string{}
	for _, iss := range issues {
		if len(iss.Path) == 0 {
			formErrors = append(formErrors, iss.Message)
			continue
		}
		key, _ := iss.Path[0].(string)
		if key == "" {
			key = pathSegString(iss.Path[0])
		}
		fieldErrors[key] = append(fieldErrors[key], iss.Message)
	}
	return gin.H{
		"formErrors":  formErrors,
		"fieldErrors": fieldErrors,
	}
}

// treeifyIssues is a minimal local tree (WP-F will provide zod.Treeify).
func treeifyIssues(issues []zod.Issue) gin.H {
	root := gin.H{"errors": []string{}}
	for _, iss := range issues {
		if len(iss.Path) == 0 {
			root["errors"] = append(root["errors"].([]string), iss.Message)
			continue
		}
		node := root
		for i, seg := range iss.Path {
			key := pathSegString(seg)
			if i == len(iss.Path)-1 {
				props, _ := node["properties"].(gin.H)
				if props == nil {
					props = gin.H{}
					node["properties"] = props
				}
				leaf, _ := props[key].(gin.H)
				if leaf == nil {
					leaf = gin.H{"errors": []string{}}
					props[key] = leaf
				}
				leaf["errors"] = append(leaf["errors"].([]string), iss.Message)
				continue
			}
			props, _ := node["properties"].(gin.H)
			if props == nil {
				props = gin.H{}
				node["properties"] = props
			}
			child, _ := props[key].(gin.H)
			if child == nil {
				child = gin.H{"errors": []string{}}
				props[key] = child
			}
			node = child
		}
	}
	return root
}

func pathSegString(seg any) string {
	switch v := seg.(type) {
	case string:
		return v
	case int:
		return itoa(v)
	case float64:
		return itoa(int(v))
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
