package bench

import (
	"net/mail"

	z "github.com/Oudwins/zog"
	"github.com/go-playground/validator/v10"
	goz "github.com/iKunalChhabra/go-z/z"
)

// ---------------------------------------------------------------------------
// Struct models (validator / handwritten / zog dest)
// ---------------------------------------------------------------------------

type FlatUser struct {
	Name  string `validate:"required,min=5"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lt=150"`
}

type Address struct {
	City string `validate:"required,min=1"`
	Zip  string `validate:"required,min=3,max=16"`
}

type NestedUser struct {
	Name    string   `validate:"required,min=5"`
	Email   string   `validate:"required,email"`
	Age     int      `validate:"gte=0,lt=150"`
	Address Address  `validate:"required"`
	Tags    []string `validate:"max=10,dive,required,min=1"`
}

type FormatPayload struct {
	Email string `validate:"required,email"`
	Uuid  string `validate:"required,uuid"`
	Url   string `validate:"required,url"`
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func validFlatUser() FlatUser {
	return FlatUser{Name: "alice", Email: "alice@example.com", Age: 30}
}

func validFlatUserMap() map[string]any {
	return map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   float64(30),
	}
}

func invalidFlatUser() FlatUser {
	return FlatUser{Name: "ab", Email: "not-email", Age: 200}
}

func invalidFlatUserMap() map[string]any {
	return map[string]any{
		"name":  "ab",
		"email": "not-email",
		"age":   float64(200),
	}
}

func validNestedUser() NestedUser {
	return NestedUser{
		Name:  "alice",
		Email: "alice@example.com",
		Age:   30,
		Address: Address{
			City: "Seattle",
			Zip:  "98101",
		},
		Tags: []string{"go", "validation", "perf"},
	}
}

func validNestedUserMap() map[string]any {
	return map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   float64(30),
		"address": map[string]any{
			"city": "Seattle",
			"zip":  "98101",
		},
		"tags": []any{"go", "validation", "perf"},
	}
}

func validFormatPayload() FormatPayload {
	return FormatPayload{
		Email: "alice@example.com",
		Uuid:  "550e8400-e29b-41d4-a716-446655440000",
		Url:   "https://example.com/path",
	}
}

func validFormatMap() map[string]any {
	return map[string]any{
		"email": "alice@example.com",
		"uuid":  "550e8400-e29b-41d4-a716-446655440000",
		"url":   "https://example.com/path",
	}
}

func makeFlatUserSlice(n int) []FlatUser {
	out := make([]FlatUser, n)
	u := validFlatUser()
	for i := range out {
		out[i] = u
	}
	return out
}

func makeFlatUserMapSlice(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = validFlatUserMap()
	}
	return out
}

// ---------------------------------------------------------------------------
// go-z schemas
// ---------------------------------------------------------------------------

func gozFlatUser() *goz.ObjectSchema {
	return goz.Object(goz.Shape{
		"name":  goz.String().Min(5),
		"email": goz.String().Email(),
		"age":   goz.Int().Gte(0).Lt(150),
	})
}

func gozNestedUser() *goz.ObjectSchema {
	return goz.Object(goz.Shape{
		"name":  goz.String().Min(5),
		"email": goz.String().Email(),
		"age":   goz.Int().Gte(0).Lt(150),
		"address": goz.Object(goz.Shape{
			"city": goz.String().Min(1),
			"zip":  goz.String().Min(3).Max(16),
		}),
		"tags": goz.Array(goz.String().Min(1)).Max(10),
	})
}

func gozFormats() *goz.ObjectSchema {
	return goz.Object(goz.Shape{
		"email": goz.String().Email(),
		"uuid":  goz.String().UUID(),
		"url":   goz.String().URL(),
	})
}

// ---------------------------------------------------------------------------
// go-playground/validator
// ---------------------------------------------------------------------------

func playgroundValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}

// ---------------------------------------------------------------------------
// Oudwins/zog schemas
// ---------------------------------------------------------------------------

func zogFlatUser() *z.StructSchema {
	return z.Struct(z.Shape{
		"name":  z.String().Min(5).Required(),
		"email": z.String().Email().Required(),
		"age":   z.Int().GTE(0).LT(150).Required(),
	})
}

func zogNestedUser() *z.StructSchema {
	return z.Struct(z.Shape{
		"name":  z.String().Min(5).Required(),
		"email": z.String().Email().Required(),
		"age":   z.Int().GTE(0).LT(150).Required(),
		"address": z.Struct(z.Shape{
			"city": z.String().Min(1).Required(),
			"zip":  z.String().Min(3).Max(16).Required(),
		}).Required(),
		"tags": z.Slice(z.String().Min(1).Required()).Max(10).Required(),
	})
}

func zogFormats() *z.StructSchema {
	return z.Struct(z.Shape{
		"email": z.String().Email().Required(),
		"uuid":  z.String().UUID().Required(),
		"url":   z.String().URL().Required(),
	})
}

// gozFlatUserStruct validates the same shape and decodes into a FlatUser, which
// is what zog's Parse(data, &dest) does — the map-producing benchmark above is
// not comparable to it.
func gozFlatUserStruct() goz.Schema[FlatUser] {
	return goz.ToStruct[FlatUser](gozFlatUser())
}

// ---------------------------------------------------------------------------
// Handwritten baseline (FlatUser lower bound)
// ---------------------------------------------------------------------------

func handwrittenFlatUser(u *FlatUser) error {
	if len(u.Name) < 5 {
		return errTooShortName
	}
	// mail.ParseAddress already rejects an address without "@"; a second check
	// would only make the lower bound look slower than it is.
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return errBadEmail
	}
	if u.Age < 0 || u.Age >= 150 {
		return errBadAge
	}
	return nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

const (
	errTooShortName simpleError = "name too short"
	errBadEmail     simpleError = "bad email"
	errBadAge       simpleError = "bad age"
)
