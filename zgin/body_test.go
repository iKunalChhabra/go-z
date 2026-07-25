package zgin_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
	"github.com/iKunalChhabra/go-z/zgin"
)

// post sends body to a handler that binds it with schema, returning the recorder.
func post(t *testing.T, schema z.AnySchemaLike, contentType, body string, opts ...zgin.BindOptions) (*httptest.ResponseRecorder, any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var bound any
	r.POST("/", func(c *gin.Context) {
		v, ok := zgin.BindJSONAny(c, schema, opts...)
		if !ok {
			return
		}
		bound = v
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, bound
}

// An unbounded io.ReadAll of the request body let any client allocate as much
// server memory as it cared to send.
func TestBodySizeIsLimited(t *testing.T) {
	schema := z.Object(z.Shape{"note": z.String()})
	big := `{"note":"` + strings.Repeat("a", 4096) + `"}`

	w, _ := post(t, schema, "application/json", big, zgin.BindOptions{MaxBodyBytes: 1024})
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413. body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "too large") {
		t.Errorf("body = %s", w.Body.String())
	}

	// Within the limit it binds normally.
	w, bound := post(t, schema, "application/json", `{"note":"short"}`, zgin.BindOptions{MaxBodyBytes: 1024})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if bound.(map[string]any)["note"] != "short" {
		t.Errorf("bound = %#v", bound)
	}

	// A negative limit means unlimited.
	w, _ = post(t, schema, "application/json", big, zgin.BindOptions{MaxBodyBytes: -1})
	if w.Code != http.StatusOK {
		t.Fatalf("unlimited: status = %d: %s", w.Code, w.Body.String())
	}
}

func TestContentTypeIsChecked(t *testing.T) {
	schema := z.Object(z.Shape{"note": z.String()})
	body := `{"note":"x"}`

	for _, ct := range []string{"application/json", "application/json; charset=utf-8", "application/vnd.api+json", ""} {
		w, _ := post(t, schema, ct, body)
		if w.Code != http.StatusOK {
			t.Errorf("Content-Type %q should be accepted, got %d: %s", ct, w.Code, w.Body.String())
		}
	}

	for _, ct := range []string{"application/x-www-form-urlencoded", "text/html", "multipart/form-data; boundary=x"} {
		w, _ := post(t, schema, ct, body)
		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q should be rejected with 415, got %d", ct, w.Code)
		}
	}

	// Opting out reaches the schema regardless of the header.
	w, _ := post(t, schema, "text/html", body, zgin.BindOptions{AllowAnyContentType: true})
	if w.Code != http.StatusOK {
		t.Errorf("AllowAnyContentType: status = %d: %s", w.Code, w.Body.String())
	}
}

// encoding/json's default float64 rounds anything above 2^53, so an int64 id
// arrived at the schema already wrong. The body decoder keeps it exact.
func TestLargeIntegersSurviveBinding(t *testing.T) {
	const id int64 = 9007199254740993 // 2^53 + 1
	schema := z.Object(z.Shape{"id": z.Int64()})

	w, bound := post(t, schema, "application/json", fmt.Sprintf(`{"id":%d}`, id))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	got := bound.(map[string]any)["id"]
	if got != id {
		t.Fatalf("id = %#v, want int64(%d)", got, id)
	}

	// Ordinary numbers still arrive as float64, the JSON model.
	_, bound = post(t, z.Object(z.Shape{"n": z.Any()}), "application/json", `{"n":1.5}`)
	if v := bound.(map[string]any)["n"]; v != 1.5 {
		t.Fatalf("n = %#v (%T), want float64(1.5)", v, v)
	}
	_, bound = post(t, z.Object(z.Shape{"n": z.Any()}), "application/json", `{"n":42}`)
	if v := bound.(map[string]any)["n"]; v != float64(42) {
		t.Fatalf("n = %#v (%T), want float64(42)", v, v)
	}
}

func TestBodyEdgeCases(t *testing.T) {
	schema := z.Object(z.Shape{"note": z.String()})

	cases := []struct {
		name   string
		body   string
		status int
		want   string
	}{
		{"empty", "", http.StatusBadRequest, "empty request body"},
		{"invalid json", `{"note":`, http.StatusBadRequest, "invalid JSON"},
		{"trailing data", `{"note":"a"}{"note":"b"}`, http.StatusBadRequest, "unexpected data"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, _ := post(t, schema, "application/json", c.body)
			if w.Code != c.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, c.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), c.want) {
				t.Errorf("body = %s, want it to mention %q", w.Body.String(), c.want)
			}
		})
	}

	// Trailing data can be allowed for clients that stream concatenated values.
	w, bound := post(t, schema, "application/json", `{"note":"a"}{"note":"b"}`,
		zgin.BindOptions{AllowTrailingData: true})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if bound.(map[string]any)["note"] != "a" {
		t.Errorf("the first value should win: %#v", bound)
	}
}

// The failure response is the same issue shape as a validation failure, so a
// client can parse one error format.
func TestBindFailureShape(t *testing.T) {
	w, _ := post(t, z.Object(z.Shape{"note": z.String()}), "text/html", `{}`)
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Issues []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"issues"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not the issue shape: %s", w.Body.String())
	}
	if body.Success || len(body.Error.Issues) != 1 || body.Error.Issues[0].Code != "custom" {
		t.Fatalf("body = %s", w.Body.String())
	}
}
