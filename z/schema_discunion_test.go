package z

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// Ports cases from v4/classic/tests/discriminated-unions.test.ts.

// Port: "valid parse - object"
func TestDiscUnionValidParse(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	got, err := schema.Parse(map[string]any{"type": "a", "a": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["type"] != "a" || m["a"] != "abc" {
		t.Fatalf("got %#v", got)
	}
}

// Port: "valid - optional discriminator (object)"
func TestDiscUnionOptionalDiscriminator(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Optional(Literal("a")), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	got, err := schema.Parse(map[string]any{"type": "a", "a": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["a"] != "abc" {
		t.Fatalf("got %#v", got)
	}
	got, err = schema.Parse(map[string]any{"a": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["a"] != "abc" {
		t.Fatalf("got %#v", got)
	}
}

// Port: "valid - discriminator value of various primitive types" (subset)
func TestDiscUnionVariousPrimitives(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("1"), "val": String()}),
		Object(Shape{"type": Literal(1.0), "val": String()}),
		Object(Shape{"type": Literal(true), "val": String()}),
		Object(Shape{"type": Nil(), "val": String()}),
	})
	cases := []any{"1", 1.0, true, nil}
	for _, disc := range cases {
		got, err := schema.Parse(map[string]any{"type": disc, "val": "v"})
		if err != nil {
			t.Fatalf("disc=%v: %v", disc, err)
		}
		if got.(map[string]any)["val"] != "v" {
			t.Fatalf("disc=%v got %#v", disc, got)
		}
	}
}

// Port: "invalid - null"
func TestDiscUnionInvalidNull(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(nil)
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidType || iss.Expected != "object" {
		t.Fatalf("got %+v", iss)
	}
}

// Port: "invalid discriminator value"
func TestDiscUnionInvalidDiscriminator(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(map[string]any{"type": "x", "a": "abc"})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion {
		t.Fatalf("code = %s", iss.Code)
	}
	if iss.Discriminator != "type" {
		t.Fatalf("discriminator = %q", iss.Discriminator)
	}
	if len(iss.Path) != 1 || iss.Path[0] != "type" {
		t.Fatalf("path = %v", iss.Path)
	}
	if len(iss.Values) != 2 {
		t.Fatalf("values = %v", iss.Values)
	}
	wantMsg := "Invalid discriminator value. Expected 'a' | 'b'"
	if iss.Message != wantMsg {
		t.Fatalf("message = %q want %q", iss.Message, wantMsg)
	}
}

// Port: "valid discriminator value, invalid data"
func TestDiscUnionValidDiscInvalidData(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(map[string]any{"type": "a", "b": "abc"})
	if res.Success {
		t.Fatal("expected failure")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueInvalidType && iss.Expected == "string" {
			found = true
			if len(iss.Path) == 0 || iss.Path[0] != "a" {
				t.Fatalf("path = %v", iss.Path)
			}
		}
	}
	if !found {
		t.Fatalf("want invalid_type for a, got %+v", res.Error.Issues)
	}
}

// Port: "invalid discriminator value - unionFallback"
func TestDiscUnionFallback(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	}, DiscUnionParams{UnionFallback: true})
	res := schema.SafeParse(map[string]any{"type": "x", "a": "abc"})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || len(iss.Errors) != 2 {
		t.Fatalf("want wrapped union errors, got %+v", iss)
	}
}

// Port: empty options
func TestDiscUnionEmpty(t *testing.T) {
	schema := DiscriminatedUnion("type", nil)
	res := schema.SafeParse("nope")
	if res.Success || res.Error.Issues[0].Code != IssueInvalidType {
		t.Fatalf("non-object: %+v", res.Error)
	}
	res = schema.SafeParse(map[string]any{"type": "x"})
	if res.Success || res.Error.Issues[0].Code != IssueInvalidUnion {
		t.Fatalf("unknown disc: %+v", res.Error)
	}
}

func TestDiscUnionDuplicatePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(fmt.Sprint(r), "Duplicate discriminator") {
			t.Fatalf("recover = %v", r)
		}
	}()
	_ = DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("a"), "b": String()}),
	})
}

func TestDiscUnionMissingDiscPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(fmt.Sprint(r), "Invalid discriminated union option") {
			t.Fatalf("recover = %v", r)
		}
	}()
	_ = DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"value": String()}),
	})
}

func TestDiscUnionForwardReferencedLazyOption(t *testing.T) {
	// Regression: a Lazy option whose target is assigned after construction
	// (recursive schemas) panicked at build time resolving the discriminator
	// map. The table is now built on first Parse.
	var node AnySchemaLike
	nodeLazy := Lazy(func() AnySchemaLike { return node })
	u := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("leaf"), "v": String()}),
		nodeLazy,
	})
	node = Object(Shape{"type": Literal("node"), "child": Optional(u)})

	got, err := u.Parse(map[string]any{"type": "leaf", "v": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["v"] != "x" {
		t.Fatalf("%#v", got)
	}
	got, err = u.Parse(map[string]any{
		"type":  "node",
		"child": map[string]any{"type": "leaf", "v": "inner"},
	})
	if err != nil {
		t.Fatal(err)
	}
	child := got.(map[string]any)["child"].(map[string]any)
	if child["v"] != "inner" {
		t.Fatalf("%#v", got)
	}
}

func TestDiscUnionSelfReferentialLazyOptionPanics(t *testing.T) {
	// A Lazy chain with no concrete schema behind it must fail with the usual
	// invalid-option panic, not spin forever in unwrapLazy.
	var l *LazySchema
	l = Lazy(func() AnySchemaLike { return l })
	u := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a")}),
		l,
	})
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(fmt.Sprint(r), "Invalid discriminated union option") {
			t.Fatalf("recover = %v", r)
		}
	}()
	_, _ = u.Parse(map[string]any{"type": "a"})
}

func TestDiscUnionFailedLazyBuildPanicsConsistently(t *testing.T) {
	// Regression: sync.Once is consumed even when the build panics, so the
	// union stayed half-built — the first Parse panicked but every later
	// Parse silently returned invalid_union for valid input. A failed build
	// must fail the same way on every Parse.
	var l *LazySchema
	l = Lazy(func() AnySchemaLike { return l })
	u := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a")}),
		l,
	})
	for attempt := 1; attempt <= 3; attempt++ {
		func() {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(fmt.Sprint(r), "Invalid discriminated union option") {
					t.Fatalf("attempt %d: recover = %v, want invalid-option panic", attempt, r)
				}
			}()
			_, _ = u.Parse(map[string]any{"type": "a"})
		}()
	}
}

func TestDiscUnionResolve(t *testing.T) {
	// Resolve forces the lazy build so misconfiguration panics at startup.
	var node AnySchemaLike
	nodeLazy := Lazy(func() AnySchemaLike { return node })
	u := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("leaf")}),
		nodeLazy,
	})
	node = Object(Shape{"type": Literal("node"), "child": Optional(u)})
	u.Resolve() // must not panic once the Lazy target is assigned
	if _, err := u.Parse(map[string]any{"type": "leaf"}); err != nil {
		t.Fatal(err)
	}

	var l *LazySchema
	l = Lazy(func() AnySchemaLike { return l })
	bad := DiscriminatedUnion("type", []AnySchemaLike{l})
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(fmt.Sprint(r), "Invalid discriminated union option") {
			t.Fatalf("recover = %v", r)
		}
	}()
	bad.Resolve()
}

func TestDiscUnionLazyBuildConcurrentParseAndCompose(t *testing.T) {
	// Regression (race detector): the lazy build used to write Internals.
	// PropValues during Parse while newObject (via Extend/Pick/Merge) reads
	// them at construction with no synchronization.
	var node AnySchemaLike
	nodeLazy := Lazy(func() AnySchemaLike { return node })
	u := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("leaf")}),
		nodeLazy,
	})
	node = Object(Shape{"type": Literal("node"), "child": Optional(u)})

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for range 50 {
				if g%2 == 0 {
					if _, err := u.Parse(map[string]any{"type": "leaf"}); err != nil {
						t.Error(err)
						return
					}
				} else {
					_ = Object(Shape{"u": u, "x": String()}).Pick("u")
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestNestedLazyDiscUnionAsOption(t *testing.T) {
	// A lazy-built union nested as an option of another union exposes its
	// discriminator values via its dispatch table, not Internals.PropValues.
	var inner AnySchemaLike
	innerLazy := Lazy(func() AnySchemaLike { return inner })
	inner = DiscriminatedUnion("kind", []AnySchemaLike{
		Object(Shape{"kind": Literal("x")}),
		Object(Shape{"kind": Literal("y")}),
	})
	outer := DiscriminatedUnion("kind", []AnySchemaLike{
		Object(Shape{"kind": Literal("z")}),
		innerLazy,
	})
	for _, k := range []string{"x", "y", "z"} {
		if _, err := outer.Parse(map[string]any{"kind": k}); err != nil {
			t.Fatalf("kind %q: %v", k, err)
		}
	}
}
