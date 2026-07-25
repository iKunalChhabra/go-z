package z

import (
	"math/big"
	"testing"
)

// A *big.Int is a pointer, so returning the caller's own value would let them
// mutate something the schema had already validated.
func TestBigIntOutputDoesNotAliasTheInput(t *testing.T) {
	in := big.NewInt(7)
	got, err := BigInt().Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if got == in {
		t.Fatal("the parsed value should be a copy, not the caller's pointer")
	}
	in.SetInt64(999)
	if got.Int64() != 7 {
		t.Fatalf("validated value changed to %v when the caller mutated the input", got)
	}

	// The same on the coercion path.
	in = big.NewInt(7)
	got, err = BigInt(Params{Coerce: true}).Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	in.SetInt64(999)
	if got.Int64() != 7 {
		t.Fatalf("coerced value changed to %v", got)
	}
}
