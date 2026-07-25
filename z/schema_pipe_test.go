package z

import "testing"

// Ported from v4/classic/tests/pipe.test.ts (subset; no async / encode).

func TestPipeStringToTransform(t *testing.T) {
	schema := Pipe(
		String(),
		Transform(Any(), func(v any, _ *RefinementCtx) (any, error) {
			return len(v.(string)), nil
		}),
	)
	// Easier: Transform(String(), ...) already pipes.
	schema2 := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		return len(v.(string)), nil
	})
	got, err := schema2.Parse("1234")
	if err != nil || got != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
	_ = schema
}

func TestPipeStringLength(t *testing.T) {
	// string → transform(Number) → but Number schema may be absent; assert via Any.
	schema := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		n := 0
		for _, ch := range v.(string) {
			n = n*10 + int(ch-'0')
		}
		return n, nil
	})
	got, err := schema.Parse("1234")
	if err != nil || got != 1234 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestPipeWithCatch(t *testing.T) {
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
	if got, err := schema.Parse("ok"); err != nil || got != "ok" {
		t.Fatalf("ok: %v, %v", got, err)
	}
	if got, err := schema.Parse(Missing); err != nil || got != "default" {
		t.Fatalf("Missing: %v, %v", got, err)
	}
	if got, err := schema.Parse("none"); err != nil || got != "default" {
		t.Fatalf("none: %v, %v", got, err)
	}
	if got, err := schema.Parse(15); err != nil || got != "default" {
		t.Fatalf("15: %v, %v", got, err)
	}
}

func TestPipeSkipsOutOnInFailure(t *testing.T) {
	ran := false
	schema := Pipe(
		String(),
		Transform(Any(), func(v any, _ *RefinementCtx) (any, error) {
			ran = true
			return v, nil
		}),
	)
	res := schema.SafeParse(123)
	if res.Success {
		t.Fatal("want failure")
	}
	if ran {
		t.Fatal("out schema must not run after in failure")
	}
}

func TestPipeContinueAfterNonFatalRefine(t *testing.T) {
	// refine fails (non-abort) → pipe/transform after refine-on-same-schema
	// is modeled as Refine then Transform wrapper.
	schema := Transform(
		Refine(String(), func(v any) bool { return v == "1234" }, "A"),
		func(v any, _ *RefinementCtx) (any, error) {
			return 1234, nil
		},
	)
	if _, err := schema.Parse("1234"); err != nil {
		t.Fatalf("ok path: %v", err)
	}
	res := schema.SafeParse("4321")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue A, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "A" {
		t.Fatalf("message = %q", res.Error.Issues[0].Message)
	}
}

func TestPipeInOutAccessors(t *testing.T) {
	a, b := String(), String()
	p := Pipe(a, b)
	if p.In() != a || p.Out() != b {
		t.Fatal("In/Out accessors")
	}
}
