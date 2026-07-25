package zod

import (
	"errors"
	"testing"
)

// Ported from v4/classic/tests/transform.test.ts (subset; sync only).

func TestTransformAddIssue(t *testing.T) {
	strs := map[string]bool{"foo": true, "bar": true}
	schema := Transform(String(), func(data any, ctx *RefinementCtx) (any, error) {
		s := data.(string)
		if !strs[s] {
			ctx.AddIssue(Issue{
				Code:    IssueCustom,
				Message: s + " is not one of our allowed strings",
				Input:   data,
			})
		}
		return len(s), nil
	})
	res := schema.SafeParse("asdf")
	if res.Success {
		t.Fatal("want failure")
	}
	if res.Error.Issues[0].Message != "asdf is not one of our allowed strings" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestTransformBasic(t *testing.T) {
	schema := Transform(String(), func(data any, _ *RefinementCtx) (any, error) {
		return len(data.(string)), nil
	})
	got, err := schema.Parse("asdf")
	if err != nil || got != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestTransformErrorReturn(t *testing.T) {
	schema := Transform(String(), func(any, *RefinementCtx) (any, error) {
		return nil, errors.New("bad")
	})
	res := schema.SafeParse("x")
	if res.Success || res.Error.Issues[0].Message != "bad" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestTransformToTyped(t *testing.T) {
	schema := TransformTo(String(), func(v any) (int, error) {
		return len(v.(string)), nil
	})
	got, err := schema.Parse("abcd")
	if err != nil || got != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestPreprocess(t *testing.T) {
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
		t.Fatal("preprocess still validates")
	}
}

func TestTransformNEVERStyle(t *testing.T) {
	schema := Transform(Optional(String()), func(val any, ctx *RefinementCtx) (any, error) {
		if IsMissing(val) || val == nil || val == "" {
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
