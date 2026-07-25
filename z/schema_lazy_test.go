package z

import (
	"sync/atomic"
	"testing"
)

// Ports cases from v4/classic/tests/lazy.test.ts and recursive-types.test.ts.

// Port: "schema getter"
func TestLazyBasic(t *testing.T) {
	schema := Lazy(func() AnySchemaLike { return String() })
	if _, err := schema.Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse(1.0)
	if res.Success {
		t.Fatal("expected failure")
	}
}

func TestLazyMemoizesGetter(t *testing.T) {
	var calls atomic.Int32
	schema := Lazy(func() AnySchemaLike {
		calls.Add(1)
		return String().Min(2)
	})
	_, _ = schema.Parse("ab")
	_, _ = schema.Parse("cd")
	_ = schema.SafeParse("x")
	if calls.Load() != 1 {
		t.Fatalf("getter calls = %d, want 1", calls.Load())
	}
}

func TestLazyInnerSharedAcrossCheckClones(t *testing.T) {
	var calls atomic.Int32
	base := Lazy(func() AnySchemaLike {
		calls.Add(1)
		return String()
	})
	cloned := base.Check()
	_, _ = base.Parse("a")
	_, _ = cloned.Parse("b")
	if calls.Load() != 1 {
		t.Fatalf("shared state should memoize once, got %d", calls.Load())
	}
	if base.Inner() != cloned.Inner() {
		t.Fatal("clones should share inner instance")
	}
}

// Port: "recursion with z.lazy" (Category)
func TestLazyRecursionCategory(t *testing.T) {
	var category *LazySchema
	category = Lazy(func() AnySchemaLike {
		return Object(Shape{
			"name":          String(),
			"subcategories": Array(category),
		})
	})
	data := map[string]any{
		"name": "I",
		"subcategories": []any{
			map[string]any{
				"name": "A",
				"subcategories": []any{
					map[string]any{
						"name": "1",
						"subcategories": []any{
							map[string]any{
								"name":          "a",
								"subcategories": []any{},
							},
						},
					},
				},
			},
		},
	}
	got, err := category.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["name"] != "I" {
		t.Fatalf("got %#v", got)
	}
}

// Port: "recursive union wit z.lazy" (LinkedList)
func TestLazyRecursiveUnion(t *testing.T) {
	var ll *LazySchema
	ll = Lazy(func() AnySchemaLike {
		return Union([]AnySchemaLike{
			Nil(),
			Object(Shape{
				"value": Number(),
				"next":  ll,
			}),
		})
	})
	data := map[string]any{
		"value": 1.0,
		"next": map[string]any{
			"value": 2.0,
			"next": map[string]any{
				"value": 3.0,
				"next": map[string]any{
					"value": 4.0,
					"next":  nil,
				},
			},
		},
	}
	if _, err := ll.Parse(data); err != nil {
		t.Fatal(err)
	}
	if _, err := ll.Parse(nil); err != nil {
		t.Fatal(err)
	}
}

// Port: "mutual recursion with lazy"
func TestLazyMutualRecursion(t *testing.T) {
	var a, b *LazySchema
	a = Lazy(func() AnySchemaLike {
		return Object(Shape{
			"val": Number(),
			"b":   b,
		})
	})
	b = Lazy(func() AnySchemaLike {
		return Object(Shape{
			"val": Number(),
			"a":   Optional(a),
		})
	})
	testData := map[string]any{
		"val": 1.0,
		"b": map[string]any{
			"val": 5.0,
			"a": map[string]any{
				"val": 3.0,
				"b": map[string]any{
					"val": 4.0,
					"a": map[string]any{
						"val": 2.0,
						"b": map[string]any{
							"val": 1.0,
						},
					},
				},
			},
		},
	}
	if _, err := a.Parse(testData); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Parse(map[string]any{"val": "asdf"}); err == nil {
		t.Fatal("expected type failure")
	}
}

func TestLazyNilGetterPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = Lazy(nil)
}

func TestLazyInnerSyncsTraits(t *testing.T) {
	inner := Optional(String())
	schema := Lazy(func() AnySchemaLike { return inner })
	if schema.Internals().OptIn {
		t.Fatal("OptIn should be unset before Inner()")
	}
	_ = schema.Inner()
	if !schema.Internals().OptIn {
		t.Fatal("OptIn should sync from inner after Inner()")
	}
}
