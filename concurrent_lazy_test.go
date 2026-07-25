package zod

import (
	"sync"
	"testing"
)

// Regression: ToJSONSchema/Unwrap on a Lazy used to write to shared Internals
// on every call, racing with concurrent Parse of an object holding that Lazy.
func TestConcurrentLazyIntrospectionAndParse(t *testing.T) {
	var node *ObjectSchema
	lazyNode := Lazy(func() AnySchemaLike { return node })
	node = Object(Shape{
		"name":  String(),
		"child": Optional(lazyNode),
	})
	data := map[string]any{"name": "root", "child": map[string]any{"name": "leaf"}}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if i%2 == 0 {
					if _, err := node.Parse(data); err != nil {
						t.Error(err)
						return
					}
				} else {
					_ = lazyNode.Inner()
				}
			}
		}(i)
	}
	wg.Wait()
}
