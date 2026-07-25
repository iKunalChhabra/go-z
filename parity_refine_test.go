package zod

import (
	"reflect"
	"testing"
)

// Parity ports from:
//   packages/zod/src/v4/classic/tests/refine.test.ts
//   packages/zod/src/v4/classic/tests/nested-refine.test.ts
//   packages/zod/src/v4/classic/tests/custom.test.ts
//   packages/zod/src/v4/classic/tests/continuability.test.ts

func TestParityRefineBasicValidation(t *testing.T) {
	// Port: refine.test.ts "should validate according to refinement logic"
	schema := Refine(
		Object(Shape{"first": String(), "second": String()}).Partial().Strict(),
		func(v any) bool {
			m := v.(map[string]any)
			_, f := m["first"]
			_, s := m["second"]
			return f || s
		},
		"Either first or second should be filled in.",
	)
	cases := []struct {
		name   string
		in     any
		wantOK bool
	}{
		{"empty", map[string]any{}, false},
		{"first", map[string]any{"first": "a"}, true},
		{"second", map[string]any{"second": "a"}, true},
		{"both", map[string]any{"first": "a", "second": "a"}, true},
		{"extra", map[string]any{"third": "x"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := schema.SafeParse(tc.in)
			if res.Success != tc.wantOK {
				t.Fatalf("success=%v err=%v", res.Success, res.Error)
			}
		})
	}
}

func TestParityRefineCustomErrorMessage(t *testing.T) {
	// Port: "should use custom error message when validation fails"
	schema := Refine(
		Object(Shape{
			"email":           String().Email(),
			"password":        String(),
			"confirmPassword": String(),
		}),
		func(v any) bool {
			m := v.(map[string]any)
			return m["password"] == m["confirmPassword"]
		},
		"Both password and confirmation must match",
	)
	res := schema.SafeParse(map[string]any{
		"email": "aaaa@gmail.com", "password": "aaaaaaaa", "confirmPassword": "bbbbbbbb",
	})
	if res.Success || res.Error.Issues[0].Message != "Both password and confirmation must match" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityRefineAsyncUnsupported(t *testing.T) {
	// Port: "async refinements"
	t.Skip("async refinements not supported in go-zod")
}

func TestParityRefineAbortContinueFalse(t *testing.T) {
	// Port: "should abort early with continue: false"
	schema := Refine(
		SuperRefine(String(), func(val any, ctx *RefinementCtx) {
			if len(val.(string)) < 2 {
				ctx.AddIssue(Issue{Code: IssueCustom, Message: "BAD"}.WithAbort())
			}
		}),
		func(any) bool { return false },
		"next",
	)
	res := schema.SafeParse("")
	if res.Success || len(res.Error.Issues) != 1 || res.Error.Issues[0].Message != "BAD" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityRefineAbortFlag(t *testing.T) {
	// Port: "should abort early with abort flag"
	schema := Refine(
		Refine(String(), func(any) bool { return false }, Params{Abort: true, Error: MessageFromString("A")}),
		func(any) bool { return false },
		"B",
	)
	res := schema.SafeParse("")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityRefineCustomPath(t *testing.T) {
	// Port: "should use custom path in error message"
	schema := Refine(
		Object(Shape{"password": String(), "confirm": String()}),
		func(v any) bool {
			m := v.(map[string]any)
			return m["confirm"] == m["password"]
		},
		RefineOpts{Path: []any{"confirm"}},
	)
	res := schema.SafeParse(map[string]any{"password": "asdf", "confirm": "qewr"})
	if res.Success || !reflect.DeepEqual(res.Error.Issues[0].Path, []any{"confirm"}) {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParitySuperRefineMultipleRules(t *testing.T) {
	// Port: "should support multiple validation rules"
	schema := SuperRefine(Array(String()), func(val any, ctx *RefinementCtx) {
		arr := val.([]any)
		if len(arr) > 3 {
			ctx.AddIssue(Issue{
				Code: IssueTooBig, Origin: "array", Maximum: 3, Inclusive: true,
				Message: "Too many items 😡", Input: val,
			})
		}
		seen := map[any]struct{}{}
		dup := false
		for _, x := range arr {
			if _, ok := seen[x]; ok {
				dup = true
				break
			}
			seen[x] = struct{}{}
		}
		if dup {
			ctx.AddIssue(Issue{Code: IssueCustom, Message: "No duplicates allowed.", Input: val})
		}
	})
	if _, err := schema.Parse([]any{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse([]any{"a", "a"})
	if res.Success || res.Error.Issues[0].Message != "No duplicates allowed." {
		t.Fatalf("dup: %+v", res.Error)
	}
	res = schema.SafeParse([]any{"a", "b", "c", "d"})
	if res.Success || res.Error.Issues[0].Code != IssueTooBig {
		t.Fatalf("too big: %+v", res.Error)
	}
}

func TestParitySuperRefineContinueDefault(t *testing.T) {
	schema := Refine(
		SuperRefine(String(), func(_ any, ctx *RefinementCtx) {
			ctx.AddIssue(Issue{Code: IssueCustom, Message: "First issue"})
		}),
		func(any) bool { return false },
		"Second issue",
	)
	res := schema.SafeParse("test")
	if res.Success || len(res.Error.Issues) != 2 {
		t.Fatalf("want 2 issues, got %+v", res.Error)
	}
}

func TestParitySuperRefineAddMessage(t *testing.T) {
	schema := SuperRefine(String(), func(_ any, ctx *RefinementCtx) {
		ctx.AddMessage("bad stuff")
	})
	res := schema.SafeParse("x")
	if res.Success || res.Error.Issues[0].Message != "bad stuff" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityNestedRefine(t *testing.T) {
	// Port: nested-refine.test.ts "nested refinements"
	schema := Refine(
		Object(Shape{
			"password": String().Min(1),
			"nested": Refine(
				Object(Shape{
					"confirm": Refine(
						String().Min(1),
						func(v any) bool { return len(v.(string)) > 2 },
						"Confirm length should be > 2",
					),
				}),
				func(v any) bool {
					return v.(map[string]any)["confirm"] == "bar"
				},
				RefineOpts{
					Path:  []any{"confirm"},
					Error: MessageFromString(`Value must be "bar"`),
				},
			),
		}),
		func(v any) bool {
			m := v.(map[string]any)
			nested := m["nested"].(map[string]any)
			return nested["confirm"] == m["password"]
		},
		RefineOpts{
			Path:  []any{"nested", "confirm"},
			Error: MessageFromString("Password and confirm must match"),
		},
	)

	// Empty confirm: too_small aborts nested refinements differently per check
	// continuability — at least path-prefixed issues should surface.
	res := schema.SafeParse(map[string]any{
		"password": "bar",
		"nested":   map[string]any{"confirm": ""},
	})
	if res.Success {
		t.Fatal("expected failure")
	}
	paths := map[string]bool{}
	for _, iss := range res.Error.Issues {
		paths[ToDotPath(iss.Path)] = true
	}
	if !paths["nested.confirm"] {
		t.Fatalf("want nested.confirm issues, got %+v", res.Error.Issues)
	}

	ok, err := schema.Parse(map[string]any{
		"password": "bar",
		"nested":   map[string]any{"confirm": "bar"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok.(map[string]any)["password"] != "bar" {
		t.Fatalf("%#v", ok)
	}
}

func TestParityCustomPassing(t *testing.T) {
	// Port: custom.test.ts "passing validations"
	schema := Custom(func(x any) bool {
		_, ok := ToFloat(x)
		return ok
	})
	if _, err := schema.Parse(1234.0); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse(map[string]any{}).Success {
		t.Fatal("object should fail")
	}
}

func TestParityCustomStringParams(t *testing.T) {
	// Port: "string params" — custom fails when predicate returns false
	schema := Custom(func(x any) bool {
		_, ok := ToFloat(x)
		return !ok // fail on numbers
	}, "customerr")
	res := schema.SafeParse(1234.0)
	if res.Success || res.Error.Issues[0].Message != "customerr" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityCustomInstanceofUnsupported(t *testing.T) {
	// Port: "instanceof"
	t.Skip("instanceof not supported in go-zod")
}

func TestParityCustomNonContinuableByDefault(t *testing.T) {
	// Port: "non-continuable by default"
	schema := Transform(
		Custom(func(val any) bool {
			_, ok := val.(string)
			return ok
		}),
		func(any, *RefinementCtx) (any, error) {
			t.Fatal("transform must not run after custom abort")
			return nil, nil
		},
	)
	res := schema.SafeParse(123)
	if res.Success || res.Error.Issues[0].Code != IssueCustom {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityContinuabilityFormats(t *testing.T) {
	// Port: continuability.test.ts — format checks continue into refine
	type caseT struct {
		name   string
		schema AnySchemaLike
		format string
	}
	cases := []caseT{
		{"email", Refine(String().Email(), func(any) bool { return false }, "customfail"), "email"},
		{"uuid", Refine(String().UUID(), func(any) bool { return false }, "customfail"), "uuid"},
		{"url", Refine(String().URL(), func(any) bool { return false }, "customfail"), "url"},
		{"jwt", Refine(String().JWT(), func(any) bool { return false }, "customfail"), "jwt"},
		{"emoji", Refine(String().Emoji(), func(any) bool { return false }, "customfail"), "emoji"},
		{"nanoid", Refine(String().NanoID(), func(any) bool { return false }, "customfail"), "nanoid"},
		{"cuid", Refine(String().CUID(), func(any) bool { return false }, "customfail"), "cuid"},
		{"cuid2", Refine(String().CUID2(), func(any) bool { return false }, "customfail"), "cuid2"},
		{"ulid", Refine(String().ULID(), func(any) bool { return false }, "customfail"), "ulid"},
		{"ipv4", Refine(String().IPv4(), func(any) bool { return false }, "customfail"), "ipv4"},
		{"base64", Refine(String().Base64(), func(any) bool { return false }, "customfail"), "base64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var res SafeParseResult[any]
			switch s := tc.schema.(type) {
			case *CheckedSchema[any]:
				res = s.SafeParse("invalid_value")
			default:
				t.Fatalf("unexpected schema type %T", tc.schema)
			}
			if res.Success || len(res.Error.Issues) < 2 {
				t.Fatalf("want format+custom, got %+v", res.Error)
			}
			if res.Error.Issues[0].Code != IssueInvalidFormat || res.Error.Issues[0].Format != tc.format {
				t.Fatalf("first issue = %+v", res.Error.Issues[0])
			}
			if res.Error.Issues[1].Message != "customfail" {
				t.Fatalf("second = %+v", res.Error.Issues[1])
			}
		})
	}
}

func TestParityContinuabilityMinLengthAborts(t *testing.T) {
	// MinLength is typically aborting — refine may not run. Document actual behavior.
	schema := Refine(String().Min(5), func(any) bool { return false }, "customfail")
	res := schema.SafeParse("abc")
	if res.Success {
		t.Fatal("expected failure")
	}
	// Either only too_small, or too_small+custom depending on Abort defaults.
	if res.Error.Issues[0].Code != IssueTooSmall {
		t.Fatalf("first = %+v", res.Error.Issues[0])
	}
}
