package zgin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
)

// DefaultMaxBodyBytes bounds how much of a request body the JSON binders read.
// A handler that needs more should pass BindOptions{MaxBodyBytes: n}; a negative
// value removes the limit.
const DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

// BindOptions configures how a request body is read before validation.
type BindOptions struct {
	// MaxBodyBytes caps the bytes read from the body. Zero means
	// DefaultMaxBodyBytes; a negative value means unlimited.
	MaxBodyBytes int64
	// AllowAnyContentType skips the JSON media-type check. By default a request
	// whose Content-Type is not a JSON type is rejected with 415, so a form post
	// or an HTML upload does not reach the schema as "invalid JSON".
	AllowAnyContentType bool
	// AllowTrailingData accepts extra bytes after the JSON value instead of
	// rejecting the request. Off by default: two concatenated JSON documents are
	// a sign of a confused client or a smuggling attempt, not a valid body.
	AllowTrailingData bool
	// ContextKey overrides where Validate and ValidateToStruct store the parsed
	// value. Empty means ContextKey. Use it when two middlewares validate
	// different schemas on the same request and both results are needed.
	ContextKey string
}

func (o BindOptions) contextKey() string {
	if o.ContextKey == "" {
		return ContextKey
	}
	return o.ContextKey
}

func (o BindOptions) limit() int64 {
	if o.MaxBodyBytes == 0 {
		return DefaultMaxBodyBytes
	}
	return o.MaxBodyBytes
}

func firstBindOptions(opts []BindOptions) BindOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return BindOptions{}
}

// bodyCacheKey holds the raw request body on the context so a second bind on
// the same request — a chain of two Validate middlewares, or a handler that
// binds again — sees the same bytes. A request body is a one-shot reader, so
// without this the second read finds it empty. The raw bytes are cached rather
// than the decoded value: a decoded value tied to the first call's options
// would let a later, stricter bind (smaller MaxBodyBytes, no
// AllowAnyContentType, no AllowTrailingData) silently reuse a result its own
// options would reject.
const bodyCacheKey = "go-z:raw-body"

// readJSONBody reads, size-limits and decodes the request body. It writes the
// response and returns false when the request cannot produce a JSON value.
func readJSONBody(c *gin.Context, opts BindOptions) (any, bool) {
	if c.Request == nil || c.Request.Body == nil {
		bindFailure(c, http.StatusBadRequest, "missing request body")
		return nil, false
	}
	if !opts.AllowAnyContentType && !isJSONContentType(c.Request.Header.Get("Content-Type")) {
		bindFailure(c, http.StatusUnsupportedMediaType,
			"unsupported Content-Type: expected application/json")
		return nil, false
	}

	raw, ok := cachedBody(c)
	if !ok {
		body := c.Request.Body
		if limit := opts.limit(); limit >= 0 {
			// MaxBytesReader stops the read rather than buffering an arbitrarily
			// large body in memory, and closes the underlying reader.
			body = http.MaxBytesReader(c.Writer, body, limit)
		}
		b, err := io.ReadAll(body)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				bindFailure(c, http.StatusRequestEntityTooLarge, "request body too large")
			} else {
				bindFailure(c, http.StatusBadRequest, "could not read request body: "+err.Error())
			}
			return nil, false
		}
		raw = b
		c.Set(bodyCacheKey, raw)
	} else if limit := opts.limit(); limit >= 0 && int64(len(raw)) > limit {
		// The bytes were read under a looser limit; this bind's cap still applies.
		bindFailure(c, http.StatusRequestEntityTooLarge, "request body too large")
		return nil, false
	}

	// The read above drained the one-shot body; restore it so downstream
	// handlers and logging middleware can still read the request.
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))

	dec := json.NewDecoder(bytes.NewReader(raw))
	// Decode integers exactly: encoding/json's default float64 silently rounds
	// anything above 2^53, which is where database identifiers live.
	dec.UseNumber()

	var data any
	if err := dec.Decode(&data); err != nil {
		if errors.Is(err, io.EOF) {
			bindFailure(c, http.StatusBadRequest, "empty request body")
		} else {
			bindFailure(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		}
		return nil, false
	}
	if !opts.AllowTrailingData && dec.More() {
		bindFailure(c, http.StatusBadRequest, "unexpected data after the JSON value")
		return nil, false
	}

	return normalizeJSONNumbers(data), true
}

func cachedBody(c *gin.Context) ([]byte, bool) {
	cached, ok := c.Get(bodyCacheKey)
	if !ok {
		return nil, false
	}
	raw, ok := cached.([]byte)
	return raw, ok
}

// isJSONContentType accepts application/json and the +json structured suffix,
// with or without parameters. An absent Content-Type on a body-carrying request
// is treated as JSON, since many clients omit it.
func isJSONContentType(header string) bool {
	if strings.TrimSpace(header) == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return false
	}
	return mediaType == "application/json" ||
		mediaType == "text/json" ||
		strings.HasSuffix(mediaType, "+json")
}

// normalizeJSONNumbers replaces json.Number with a plain Go number, so schemas
// and user closures never see the decoder's intermediate type. A number that
// float64 cannot hold exactly becomes an int64, which is the whole point of
// decoding with UseNumber.
func normalizeJSONNumbers(v any) any {
	switch n := v.(type) {
	case json.Number:
		return normalizeJSONNumber(n)
	case map[string]any:
		for k, member := range n {
			n[k] = normalizeJSONNumbers(member)
		}
		return n
	case []any:
		for i, member := range n {
			n[i] = normalizeJSONNumbers(member)
		}
		return n
	default:
		return v
	}
}

// maxExactFloatInt is 2^53: the largest magnitude at which every integer is
// exactly representable as a float64.
const maxExactFloatInt = int64(1) << 53

func normalizeJSONNumber(n json.Number) any {
	if i, err := n.Int64(); err == nil {
		if i > maxExactFloatInt || i < -maxExactFloatInt {
			// float64 cannot hold every integer of this magnitude, and comparing
			// float64(i) with the parsed float would not reveal that — the
			// conversion rounds to the same value. Keep the integer.
			return i
		}
		return float64(i)
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	// Outside both int64 and float64 range; hand the schema the text so it can
	// report an invalid_type rather than a silent infinity.
	return n.String()
}

func bindFailure(c *gin.Context, status int, message string) {
	AbortWithError(c, &z.Error{Issues: []z.Issue{{
		Code:    z.IssueCustom,
		Message: message,
		Path:    []any{},
	}}}, Options{Status: status})
}
