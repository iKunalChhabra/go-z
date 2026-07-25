package zod

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

// Parity ports from:
//   packages/zod/src/v4/classic/tests/pipe.test.ts
//   packages/zod/src/v4/classic/tests/transform.test.ts
//   packages/zod/src/v4/classic/tests/preprocess.test.ts

func TestParityPipeStringToNumber(t *testing.T) {
	// Port: "string to number pipe"
	schema := Pipe(
		Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
			n, err := strconv.ParseFloat(v.(string), 64)
			return n, err
		}),
		Number(),
	)
	got, err := schema.Parse("1234")
	if err != nil || got != 1234.0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParityPipeAsyncUnsupported(t *testing.T) {
	// Port: "string to number pipe async"
	t.Skip("async transforms not supported in go-zod")
}

func TestParityPipeDefaultFallback(t *testing.T) {
	// Port: "string with default fallback"
	schema := Catch(
		Pipe(
			Transform(Any(), func(v any, _ *RefinementCtx) (any, error) {
				if v == "none" {
					return Missing, nil
				}
				return v, nil
			}),
			String(),
		),
		"default",
	)
	cases := []struct {
		in   any
		want string
	}{
		{"ok", "ok"},
		{Missing, "default"},
		{"none", "default"},
		{15, "default"},
	}
	for _, tc := range cases {
		got, err := schema.Parse(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("Parse(%v) = %v, %v want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestParityPipeContinueOnNonFatal(t *testing.T) {
	// Port: "continue on non-fatal errors"
	schema := Refine(
		Transform(
			Refine(String(), func(c any) bool { return c == "1234" }, "A"),
			func(val any, _ *RefinementCtx) (any, error) {
				n, _ := strconv.ParseFloat(val.(string), 64)
				return n, nil
			},
		),
		func(c any) bool { return c == 1234.0 },
		"B",
	)
	if _, err := schema.Parse("1234"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse("4321")
	if res.Success || len(res.Error.Issues) != 1 || res.Error.Issues[0].Message != "A" {
		// Non-fatal refine before transform: transform may be skipped when issues exist.
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityPipeBreakOnFatal(t *testing.T) {
	// Port: "break on fatal errors"
	schema := Transform(
		Refine(String(), func(c any) bool { return c == "1234" }, Params{Abort: true, Error: MessageFromString("A")}),
		func(val any, _ *RefinementCtx) (any, error) {
			return 1234.0, nil
		},
	)
	if _, err := schema.Parse("1234"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse("4321")
	if res.Success || res.Error.Issues[0].Message != "A" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityPipeCodecsUnsupported(t *testing.T) {
	// Port: "reverse parsing with pipe" (encode/decode / codecs)
	t.Skip("codecs / encode-decode not supported in go-zod")
}

func TestParityPipeInOutAccessors(t *testing.T) {
	a, b := String(), Number()
	p := Pipe(a, b)
	if p.In() != a || p.Out() != b {
		t.Fatal("In/Out accessors")
	}
}

func TestParityTransformAddIssue(t *testing.T) {
	// Port: "transform ctx.addIssue with parse"
	strs := map[string]bool{"foo": true, "bar": true}
	schema := Transform(String(), func(data any, ctx *RefinementCtx) (any, error) {
		s := data.(string)
		if !strs[s] {
			ctx.AddIssue(Issue{
				Code: IssueCustom, Message: s + " is not one of our allowed strings", Input: data,
			})
		}
		return len(s), nil
	})
	res := schema.SafeParse("asdf")
	if res.Success || res.Error.Issues[0].Message != "asdf is not one of our allowed strings" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityTransformAsyncUnsupported(t *testing.T) {
	t.Skip("async transforms not supported in go-zod")
}

func TestParityTransformNEVER(t *testing.T) {
	// Port: "z.NEVER in transform"
	schema := Transform(Optional(Number()), func(val any, ctx *RefinementCtx) (any, error) {
		if IsMissing(val) || val == nil {
			ctx.AddIssue(Issue{Code: IssueCustom, Message: "bad", Input: val})
			return Missing, nil
		}
		return val, nil
	})
	res := schema.SafeParse(Missing)
	if res.Success || res.Error.Issues[0].Message != "bad" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityTransformBasic(t *testing.T) {
	// Port: "basic transformations"
	schema := Transform(String(), func(data any, _ *RefinementCtx) (any, error) {
		return len(data.(string)), nil
	})
	got, err := schema.Parse("asdf")
	if err != nil || got != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParityTransformCoercionInObject(t *testing.T) {
	// Port: "coercion"
	numToString := Transform(Number(), func(n any, _ *RefinementCtx) (any, error) {
		return strconv.FormatFloat(n.(float64), 'f', -1, 64), nil
	})
	schema := Object(Shape{"id": numToString})
	got, err := schema.Parse(map[string]any{"id": 5.0})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]any{"id": "5"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestParityTransformErrorReturn(t *testing.T) {
	schema := Transform(String(), func(any, *RefinementCtx) (any, error) {
		return nil, errors.New("bad")
	})
	res := schema.SafeParse("x")
	if res.Success || res.Error.Issues[0].Message != "bad" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityTransformToTyped(t *testing.T) {
	schema := TransformTo(String(), func(v any) (int, error) {
		return len(v.(string)), nil
	})
	got, err := schema.Parse("abcd")
	if err != nil || got != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParityTransformTable(t *testing.T) {
	schema := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		return v.(string) + "!", nil
	})
	cases := []struct {
		in     any
		wantOK bool
		want   any
	}{
		{"hi", true, "hi!"},
		{123, false, nil},
		{"", true, "!"},
	}
	for _, tc := range cases {
		res := schema.SafeParse(tc.in)
		if res.Success != tc.wantOK {
			t.Fatalf("in=%v success=%v", tc.in, res.Success)
		}
		if tc.wantOK && res.Data != tc.want {
			t.Fatalf("data=%v want %v", res.Data, tc.want)
		}
	}
}

func TestParityPreprocess(t *testing.T) {
	// Port: "preprocess"
	schema := Preprocess(func(data any) any {
		return []any{data}
	}, Array(String()))
	got, err := schema.Parse("asdf")
	if err != nil || !reflect.DeepEqual(got, []any{"asdf"}) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParityPreprocessAsyncUnsupported(t *testing.T) {
	t.Skip("async preprocess not supported in go-zod")
}

func TestParityPreprocessCtxUnsupported(t *testing.T) {
	// Port: preprocess ctx.addIssue variants — Go Preprocess is fn(any) any only.
	t.Skip("preprocess ctx.addIssue not supported (no ctx in Preprocess signature)")
}

func TestParityPreprocessStillValidates(t *testing.T) {
	schema := Preprocess(func(v any) any {
		if s, ok := v.(string); ok {
			return s + "!"
		}
		return v
	}, String())
	got, err := schema.Parse("hi")
	if err != nil || got != "hi!" {
		t.Fatalf("got %v, %v", got, err)
	}
	if schema.SafeParse(123).Success {
		t.Fatal("preprocess still validates output type")
	}
}

func TestParityPreprocessNumberTrim(t *testing.T) {
	schema := Preprocess(func(v any) any {
		if s, ok := v.(string); ok {
			n, err := strconv.ParseFloat(s, 64)
			if err == nil {
				return n
			}
		}
		return v
	}, Number())
	cases := []struct {
		in     any
		wantOK bool
		want   float64
	}{
		{"3.14", true, 3.14},
		{"x", false, 0},
		{2.0, true, 2.0},
	}
	for _, tc := range cases {
		res := schema.SafeParse(tc.in)
		if res.Success != tc.wantOK {
			t.Fatalf("in=%v success=%v err=%v", tc.in, res.Success, res.Error)
		}
		if tc.wantOK && res.Data != tc.want {
			t.Fatalf("data=%v want %v", res.Data, tc.want)
		}
	}
}
