package z

import "testing"

func TestBoolBasic(t *testing.T) {
	s := Bool()
	got, err := s.Parse(true)
	if err != nil || !got {
		t.Fatalf("got %v err=%v", got, err)
	}
	got, err = s.Parse(false)
	if err != nil || got {
		t.Fatalf("got %v err=%v", got, err)
	}
	res := s.SafeParse("true")
	if res.Success {
		t.Fatal("string should fail without coerce")
	}
	if res.Error.Issues[0].Message != "Invalid input: expected boolean, received string" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestBoolCoerce(t *testing.T) {
	s := Bool(Params{Coerce: true})
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"false", false},
		{"TRUE", true},
		{"1", true},
		{"0", false},
		{1, true},
		{0, false},
		{float64(1), true},
		{float64(0), false},
	}
	for _, c := range cases {
		got, err := s.Parse(c.in)
		if err != nil || got != c.want {
			t.Fatalf("Parse(%v)=%v err=%v want %v", c.in, got, err, c.want)
		}
	}
	if res := s.SafeParse("yes"); res.Success {
		t.Fatal("yes should not coerce")
	}
	if res := s.SafeParse(2); res.Success {
		t.Fatal("2 should not coerce")
	}
}
