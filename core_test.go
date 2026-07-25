package zod

import (
	"strings"
	"testing"
)

func TestAnyUnknownAcceptEverything(t *testing.T) {
	for _, v := range []any{nil, "x", 1.5, true, map[string]any{}, []any{1}} {
		if got, err := Any().Parse(v); err != nil {
			t.Fatalf("Any().Parse(%v) error: %v", v, err)
		} else if gotSlice, ok := got.([]any); ok {
			_ = gotSlice
		}
		if _, err := Unknown().Parse(v); err != nil {
			t.Fatalf("Unknown().Parse(%v) error: %v", v, err)
		}
	}
}

func TestCheckAbortAndContinueSemantics(t *testing.T) {
	failing := func(name string, abort bool) *Check {
		ch := &Check{Name: name, Abort: abort}
		ch.Fn = func(p *Payload) {
			p.AddIssue(ch.Issue(Issue{Code: IssueCustom, Input: p.Value}))
		}
		return ch
	}

	// Two non-aborting failures: both issues are collected (Zod's
	// continue-after-failure default).
	s := Any().Check(failing("a", false), failing("b", false))
	res := s.SafeParse("x")
	if res.Success || len(res.Error.Issues) != 2 {
		t.Fatalf("want 2 issues, got %+v", res.Error)
	}

	// Aborting first check suppresses the second.
	s = Any().Check(failing("a", true), failing("b", false))
	res = s.SafeParse("x")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue after abort, got %+v", res.Error)
	}
}

func TestWhenGateRunsDespiteEarlierFailure(t *testing.T) {
	fail := &Check{Name: "fail"}
	fail.Fn = func(p *Payload) { p.AddIssue(fail.Issue(Issue{Code: IssueCustom, Input: p.Value})) }
	fail.Abort = false

	ran := false
	gated := &Check{Name: "gated", When: func(p *Payload) bool { return true }}
	gated.Fn = func(p *Payload) { ran = true }

	// Unset-continue (aborting) issue first: `when` check still runs because
	// only explicit aborts suppress it.
	abortFail := &Check{Name: "abort", Abort: true}
	abortFail.Fn = func(p *Payload) {
		p.AddIssue(abortFail.Issue(Issue{Code: IssueCustom, Input: p.Value}))
	}
	Any().Check(abortFail, gated).SafeParse("x")
	if !ran {
		t.Fatal("when-gated check should run after non-explicit abort")
	}
}

func TestErrorMapPrecedence(t *testing.T) {
	mk := func(withCheckError bool) Schema[any] {
		ch := &Check{Name: "c"}
		if withCheckError {
			ch.Error = MessageFromString("check-level")
		}
		ch.Fn = func(p *Payload) { p.AddIssue(ch.Issue(Issue{Code: IssueCustom, Input: p.Value})) }
		return Any().Check(ch)
	}

	// 1. check-level wins
	res := mk(true).SafeParse("x")
	if res.Error.Issues[0].Message != "check-level" {
		t.Fatalf("want check-level, got %q", res.Error.Issues[0].Message)
	}

	// 2. per-parse ctx map
	s := mk(false)
	v, err := s.(*AnySchema).ParseCtx("x", &ParseCtx{Error: MessageFromString("ctx-level")})
	_ = v
	if err == nil || !strings.Contains(err.Error(), "ctx-level") {
		t.Fatalf("want ctx-level, got %v", err)
	}

	// 3. global custom map
	prev := Configure(Config{CustomError: MessageFromString("global-level")})
	res = mk(false).SafeParse("x")
	Configure(prev)
	if res.Error.Issues[0].Message != "global-level" {
		t.Fatalf("want global-level, got %q", res.Error.Issues[0].Message)
	}

	// 4. locale fallback ("Invalid input" for custom issues in en)
	res = mk(false).SafeParse("x")
	if res.Error.Issues[0].Message != "Invalid input" {
		t.Fatalf("want locale fallback, got %q", res.Error.Issues[0].Message)
	}
}

func TestEnLocaleMessages(t *testing.T) {
	cases := []struct {
		iss  Issue
		want string
	}{
		{Issue{Code: IssueInvalidType, Expected: "string", Input: 42.0},
			"Invalid input: expected string, received number"},
		{Issue{Code: IssueInvalidType, Expected: "number", Input: nil},
			"Invalid input: expected number, received null"},
		{Issue{Code: IssueTooSmall, Origin: "string", Minimum: 5, Inclusive: true},
			"Too small: expected string to have >=5 characters"},
		{Issue{Code: IssueTooBig, Origin: "number", Maximum: 10.0, Inclusive: false},
			"Too big: expected number to be <10"},
		{Issue{Code: IssueInvalidFormat, Format: "email"}, "Invalid email address"},
		{Issue{Code: IssueInvalidFormat, Format: "starts_with", Prefix: "ab"},
			`Invalid string: must start with "ab"`},
		{Issue{Code: IssueNotMultipleOf, Divisor: 2}, "Invalid number: must be a multiple of 2"},
		{Issue{Code: IssueUnrecognizedKeys, Keys: []string{"a", "b"}},
			`Unrecognized keys: "a", "b"`},
		{Issue{Code: IssueInvalidValue, Values: []any{"a", "b"}},
			`Invalid option: expected one of "a"|"b"`},
	}
	for _, c := range cases {
		if got := EnLocale(&c.iss); got != c.want {
			t.Errorf("EnLocale(%s): got %q want %q", c.iss.Code, got, c.want)
		}
	}
}

func TestPathPrepend(t *testing.T) {
	p := AcquirePayload(nil)
	defer ReleasePayload(p)
	p.AddIssue(Issue{Code: IssueCustom, Path: []any{"inner"}})
	p.PrependPath(0, "outer")
	if len(p.Issues[0].Path) != 2 || p.Issues[0].Path[0] != "outer" || p.Issues[0].Path[1] != "inner" {
		t.Fatalf("bad path: %v", p.Issues[0].Path)
	}
}
