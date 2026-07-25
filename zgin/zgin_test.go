package zgin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-zod"
	"github.com/iKunalChhabra/go-zod/zgin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func userSchema() *zod.ObjectSchema {
	return zod.Object(zod.Shape{
		"name":  zod.String().Min(2),
		"email": zod.String().Email(),
		"age":   zod.Int().Gte(0),
	})
}

func TestValidateMiddlewareValidJSON(t *testing.T) {
	r := gin.New()
	r.POST("/users", zgin.Validate(userSchema()), func(c *gin.Context) {
		v, ok := zgin.Get(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing"})
			return
		}
		c.JSON(http.StatusOK, v)
	})

	body := `{"name":"Ada","email":"ada@example.com","age":36}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got["name"] != "Ada" || got["email"] != "ada@example.com" {
		t.Fatalf("got %#v", got)
	}
}

func TestValidateMiddlewareInvalidJSONBody(t *testing.T) {
	r := gin.New()
	r.POST("/users", zgin.Validate(userSchema()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := `{"name":"A","email":"bad","age":-1}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Issues []zod.Issue `json:"issues"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v\nbody=%s", err, w.Body.String())
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if len(resp.Error.Issues) == 0 {
		t.Fatal("expected issues")
	}

	codes := map[zod.IssueCode]bool{}
	paths := map[string]bool{}
	for _, iss := range resp.Error.Issues {
		codes[iss.Code] = true
		if len(iss.Path) > 0 {
			if s, ok := iss.Path[0].(string); ok {
				paths[s] = true
			}
		}
	}
	// Expect too_small on name and/or invalid_format on email and/or too_small on age
	if !codes[zod.IssueTooSmall] && !codes[zod.IssueInvalidFormat] && !codes[zod.IssueInvalidType] {
		t.Fatalf("unexpected codes in issues: %+v", resp.Error.Issues)
	}
	if !paths["name"] && !paths["email"] && !paths["age"] {
		t.Fatalf("expected path segments among name/email/age, got %+v", resp.Error.Issues)
	}
}

func TestBindJSONMalformed(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{not-json`))

	_, ok := zgin.BindJSON(c, userSchema())
	if ok {
		t.Fatal("expected failure")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp["success"] != false {
		t.Fatalf("got %#v", resp)
	}
}

func TestBindQueryCoerceNumber(t *testing.T) {
	schema := zod.Object(zod.Shape{
		"limit":  zod.Coerce.Number().Gte(1).Lte(100),
		"offset": zod.Default(zod.Coerce.Number().Gte(0), float64(0)),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/items?limit=25&offset=10", nil)

	got, ok := zgin.BindQuery(c, schema)
	if !ok {
		t.Fatalf("BindQuery failed: %s", w.Body.String())
	}
	if got["limit"] != float64(25) {
		t.Fatalf("limit=%v (%T)", got["limit"], got["limit"])
	}
	if got["offset"] != float64(10) {
		t.Fatalf("offset=%v (%T)", got["offset"], got["offset"])
	}
}

func TestBindQueryCoerceNumberInvalid(t *testing.T) {
	schema := zod.Object(zod.Shape{
		"limit": zod.Coerce.Number().Gte(1),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/items?limit=nope", nil)

	_, ok := zgin.BindQuery(c, schema)
	if ok {
		t.Fatal("expected failure")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Issues []zod.Issue `json:"issues"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Success || len(resp.Error.Issues) == 0 {
		t.Fatalf("got %#v", resp)
	}
	found := false
	for _, iss := range resp.Error.Issues {
		if iss.Code == zod.IssueInvalidType {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected invalid_type, got %+v", resp.Error.Issues)
	}
}

func TestBindURI(t *testing.T) {
	schema := zod.Object(zod.Shape{
		"id": zod.Coerce.Number().Gte(1),
	})

	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		got, ok := zgin.BindURI(c, schema)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, got)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got["id"] != float64(42) {
		t.Fatalf("got %#v", got)
	}
}

func TestCoerceQueryValues(t *testing.T) {
	out := zgin.CoerceQueryValues(map[string][]string{
		"a": {"1"},
		"b": {"x", "y"},
		"c": {},
	})
	if out["a"] != "1" {
		t.Fatalf("a=%v", out["a"])
	}
	bs, ok := out["b"].([]string)
	if !ok || len(bs) != 2 || bs[0] != "x" || bs[1] != "y" {
		t.Fatalf("b=%#v", out["b"])
	}
	if out["c"] != "" {
		t.Fatalf("c=%#v", out["c"])
	}
}

func TestAbortWithErrorFormats(t *testing.T) {
	zerr := &zod.ZodError{Issues: []zod.Issue{{
		Code:    zod.IssueTooSmall,
		Message: "Too small: expected string to have >=2 characters",
		Path:    []any{"name"},
		Minimum: 2,
		Origin:  "string",
	}}}

	t.Run("issues", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		zgin.AbortWithError(c, zerr, zgin.Options{Format: zgin.FormatIssues})
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		errObj, _ := resp["error"].(map[string]any)
		if errObj["issues"] == nil {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("flatten", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		zgin.AbortWithError(c, zerr, zgin.Options{Format: zgin.FormatFlatten})
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		errObj, _ := resp["error"].(map[string]any)
		if errObj["fieldErrors"] == nil {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("pretty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		zgin.AbortWithError(c, zerr, zgin.Options{Format: zgin.FormatPretty})
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if _, ok := resp["error"].(string); !ok {
			t.Fatalf("body=%s", w.Body.String())
		}
	})
}

func TestBindJSONToStruct(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}
	schema := zod.ToStruct[User](userSchema())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(
		`{"name":"Ada","email":"ada@example.com","age":36}`,
	))

	got, ok := zgin.BindJSON(c, schema)
	if !ok {
		t.Fatalf("BindJSON failed: %s", w.Body.String())
	}
	if got.Name != "Ada" || got.Age != 36 {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateToStructGetAs(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	r := gin.New()
	r.POST("/users", zgin.ValidateToStruct[User](userSchema()), func(c *gin.Context) {
		user, ok := zgin.GetAs[User](c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing"})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	body := `{"name":"Ada","email":"ada@example.com","age":36}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got User
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.Name != "Ada" || got.Age != 36 {
		t.Fatalf("got %+v", got)
	}
}
