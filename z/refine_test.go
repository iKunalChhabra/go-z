package z

import "testing"

// Ported from v4/classic/tests/refine.test.ts (string-focused subset).

func TestRefineBasic(t *testing.T) {
	schema := Refine(String(), func(v any) bool {
		return v.(string) != "bad"
	}, "no bad")
	if _, err := schema.Parse("good"); err != nil {
		t.Fatalf("ok: %v", err)
	}
	res := schema.SafeParse("bad")
	if res.Success || res.Error.Issues[0].Message != "no bad" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestRefineAbort(t *testing.T) {
	schema := Refine(
		Refine(String(), func(any) bool { return false }, Params{Abort: true, Error: MessageFromString("A")}),
		func(any) bool { return false },
		"B",
	)
	res := schema.SafeParse("")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "A" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestSuperRefineContinueDefault(t *testing.T) {
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

func TestSuperRefineContinueFalse(t *testing.T) {
	schema := Refine(
		SuperRefine(String(), func(_ any, ctx *RefinementCtx) {
			ctx.AddIssue(Issue{Code: IssueCustom, Message: "First issue"}.WithAbort())
		}),
		func(any) bool { return false },
		"Second issue",
	)
	res := schema.SafeParse("test")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "First issue" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestSuperRefineMultipleInSame(t *testing.T) {
	schema := SuperRefine(String(), func(_ any, ctx *RefinementCtx) {
		ctx.AddIssue(Issue{Code: IssueCustom, Message: "First"}.WithAbort())
		ctx.AddIssue(Issue{Code: IssueCustom, Message: "Second"})
	})
	res := schema.SafeParse("test")
	if res.Success || len(res.Error.Issues) != 2 {
		t.Fatalf("want 2 issues in same refinement, got %+v", res.Error)
	}
}

func TestSuperRefineStringShorthand(t *testing.T) {
	schema := SuperRefine(String(), func(_ any, ctx *RefinementCtx) {
		ctx.AddMessage("bad stuff")
	})
	res := schema.SafeParse("asdf")
	if res.Success || res.Error.Issues[0].Message != "bad stuff" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestRefinePath(t *testing.T) {
	schema := Refine(String(), func(any) bool { return false }, RefineOpts{
		Error: MessageFromString("nope"),
		Path:  []any{"confirm"},
	})
	res := schema.SafeParse("x")
	if res.Success {
		t.Fatal("want failure")
	}
	if len(res.Error.Issues[0].Path) != 1 || res.Error.Issues[0].Path[0] != "confirm" {
		t.Fatalf("path = %#v", res.Error.Issues[0].Path)
	}
}

func TestOverwriteSchema(t *testing.T) {
	schema := OverwriteSchema(String(), func(v any) any {
		return v.(string) + "!"
	})
	got, err := schema.Parse("hi")
	if err != nil || got != "hi!" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestCustom(t *testing.T) {
	schema := Custom(func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) > 0
	}, "need non-empty string")
	if _, err := schema.Parse("x"); err != nil {
		t.Fatalf("ok: %v", err)
	}
	res := schema.SafeParse("")
	if res.Success {
		t.Fatal("want failure")
	}
	// custom defaults abort; message from params
	if res.Error.Issues[0].Message != "need non-empty string" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestCheckSchemaComposes(t *testing.T) {
	ch := refineCheck(func(v any) bool { return v == "ok" }, "nope")
	schema := CheckSchema(String(), ch)
	if _, err := schema.Parse("ok"); err != nil {
		t.Fatalf("ok: %v", err)
	}
	if schema.SafeParse("no").Success {
		t.Fatal("want failure")
	}
}
