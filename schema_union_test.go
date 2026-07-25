package zod

import (
	"testing"
)

// Ports cases from v4/classic/tests/union.test.ts.

// Port: "return valid over invalid"
func TestUnionReturnsValidOverInvalid(t *testing.T) {
	schema := Union([]AnySchemaLike{
		Object(Shape{"email": String().Email()}),
		String(),
	})
	if got, err := schema.Parse("asdf"); err != nil || got != "asdf" {
		t.Fatalf("Parse(asdf) = %v, %v", got, err)
	}
	got, err := schema.Parse(map[string]any{"email": "asdlkjf@lkajsdf.com"})
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["email"] != "asdlkjf@lkajsdf.com" {
		t.Fatalf("got %#v", got)
	}
}

// Port: "return errors from both union arms"
func TestUnionErrorsFromBothArms(t *testing.T) {
	schema := Union([]AnySchemaLike{Number(), Bool()})
	res := schema.SafeParse("a")
	if res.Success {
		t.Fatal("expected failure")
	}
	if len(res.Error.Issues) != 1 || res.Error.Issues[0].Code != IssueInvalidUnion {
		t.Fatalf("want single invalid_union, got %+v", res.Error.Issues)
	}
	iss := res.Error.Issues[0]
	if iss.Message != "Invalid input" {
		t.Fatalf("message = %q", iss.Message)
	}
	if len(iss.Errors) != 2 {
		t.Fatalf("want 2 error groups, got %d", len(iss.Errors))
	}
	if len(iss.Errors[0]) != 1 || iss.Errors[0][0].Expected != "number" {
		t.Fatalf("arm0 = %+v", iss.Errors[0])
	}
	if len(iss.Errors[1]) != 1 || iss.Errors[1][0].Expected != "boolean" {
		t.Fatalf("arm1 = %+v", iss.Errors[1])
	}
}

// Port: "options getter"
func TestUnionOptionsGetter(t *testing.T) {
	u := Union([]AnySchemaLike{String(), Number()})
	if len(u.Options) != 2 {
		t.Fatalf("options len = %d", len(u.Options))
	}
	if _, err := u.Options[0].(*StringSchema).Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Options[1].(*NumberSchema).Parse(1234.0); err != nil {
		t.Fatal(err)
	}
}

// Port: "union values"
func TestUnionValues(t *testing.T) {
	schema := Union([]AnySchemaLike{
		Literal("a"),
		Literal("b"),
		Literal("c"),
	})
	vals := schema.Internals().Values
	if len(vals) != 3 {
		t.Fatalf("want 3 values, got %v", vals)
	}
	for _, v := range []any{"a", "b", "c"} {
		if _, ok := vals[v]; !ok {
			t.Fatalf("missing %v", v)
		}
	}
}

// Port: "surface continuable errors only if they exist"
func TestUnionSurfaceFormatErrors(t *testing.T) {
	schema := Union([]AnySchemaLike{Bool(), String().UUID(), String().JWT()})
	res := schema.SafeParse("asdf")
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || len(iss.Errors) != 3 {
		t.Fatalf("got %+v", iss)
	}
	if iss.Errors[0][0].Code != IssueInvalidType {
		t.Fatalf("arm0 code = %s", iss.Errors[0][0].Code)
	}
	if iss.Errors[1][0].Code != IssueInvalidFormat || iss.Errors[1][0].Format != "uuid" {
		t.Fatalf("arm1 = %+v", iss.Errors[1][0])
	}
	if iss.Errors[2][0].Code != IssueInvalidFormat || iss.Errors[2][0].Format != "jwt" {
		t.Fatalf("arm2 = %+v", iss.Errors[2][0])
	}
}

// Port: "z.union([]) constructs and rejects all input"
func TestUnionEmpty(t *testing.T) {
	schema := Union(nil)
	res := schema.SafeParse("anything")
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || len(iss.Errors) != 0 || iss.Message != "Invalid input" {
		t.Fatalf("got %+v", iss)
	}
}

func TestUnionOfVariadic(t *testing.T) {
	schema := UnionOf(String(), Number())
	if _, err := schema.Parse("hi"); err != nil {
		t.Fatal(err)
	}
	if _, err := schema.Parse(1.5); err != nil {
		t.Fatal(err)
	}
}

func TestUnionSingleOptionFastPath(t *testing.T) {
	schema := Union([]AnySchemaLike{String().Min(3)})
	if _, err := schema.Parse("abcd"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse("ab")
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error.Issues[0].Code == IssueInvalidUnion {
		t.Fatalf("single-option should not wrap: %+v", res.Error.Issues)
	}
}

func TestUnionFirstSuccessShortCircuits(t *testing.T) {
	schema := Union([]AnySchemaLike{String(), Any()})
	got, err := schema.Parse("x")
	if err != nil || got != "x" {
		t.Fatalf("got %v, %v", got, err)
	}
}

// Port: "non-aborted errors"
func TestUnionNonAbortedErrors(t *testing.T) {
	arm0 := Object(Shape{
		"date":      Number(),
		"startDate": Optional(Nil()),
		"endDate":   Optional(Nil()),
	})
	arm1 := Object(Shape{
		"date":      Optional(Nil()),
		"startDate": Number(),
		"endDate":   Number(),
	})
	failContinue := &Check{Name: "refine", Abort: false}
	failContinue.Fn = func(p *Payload) {
		m, _ := p.Value.(map[string]any)
		if m["startDate"] == m["endDate"] {
			p.AddIssue(failContinue.Issue(Issue{
				Code:    IssueCustom,
				Message: "startDate and endDate must be different",
				Path:    []any{"endDate"},
				Input:   p.Value,
			}))
		}
	}
	schema := Union([]AnySchemaLike{arm0, arm1.Check(failContinue)})
	res := schema.SafeParse(map[string]any{
		"date":      nil,
		"startDate": 1.0,
		"endDate":   1.0,
	})
	if res.Success {
		t.Fatal("expected failure")
	}
	if len(res.Error.Issues) != 1 || res.Error.Issues[0].Code != IssueCustom {
		t.Fatalf("want bare custom issue, got %+v", res.Error.Issues)
	}
	if res.Error.Issues[0].Message != "startDate and endDate must be different" {
		t.Fatalf("message = %q", res.Error.Issues[0].Message)
	}
}

func TestUnionCustomErrorMessage(t *testing.T) {
	schema := Union([]AnySchemaLike{String(), Number()}, "not string or number")
	res := schema.SafeParse(true)
	if res.Success || res.Error.Issues[0].Message != "not string or number" {
		t.Fatalf("got %+v", res.Error)
	}
}
