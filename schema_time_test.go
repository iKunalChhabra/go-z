package zod

import (
	"testing"
	"time"
)

// Ported from classic/tests/date.test.ts

func TestTimeBasic(t *testing.T) {
	s := Time()
	now := time.Now().UTC().Truncate(time.Second)
	got, err := s.Parse(now)
	if err != nil || !got.Equal(now) {
		t.Fatalf("got %v err=%v", got, err)
	}
	got, err = s.Parse(&now)
	if err != nil || !got.Equal(now) {
		t.Fatalf("ptr: got %v err=%v", got, err)
	}
	res := s.SafeParse("not-a-date")
	if res.Success || res.Error.Issues[0].Message != "Invalid input: expected date, received string" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestTimeMinMax(t *testing.T) {
	before := time.Date(2022, 11, 4, 0, 0, 0, 0, time.UTC)
	benchmark := time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)
	after := time.Date(2022, 11, 6, 0, 0, 0, 0, time.UTC)

	minCheck := Time().Min(benchmark)
	maxCheck := Time().Max(benchmark)

	if _, err := minCheck.Parse(benchmark); err != nil {
		t.Fatal(err)
	}
	if _, err := minCheck.Parse(after); err != nil {
		t.Fatal(err)
	}
	if _, err := maxCheck.Parse(benchmark); err != nil {
		t.Fatal(err)
	}
	if _, err := maxCheck.Parse(before); err != nil {
		t.Fatal(err)
	}

	res := minCheck.SafeParse(before)
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Message != "Too small: expected date to be >=1667606400000" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}

	res = maxCheck.SafeParse(after)
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Message != "Too big: expected date to be <=1667606400000" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	if res.Error.Issues[0].Origin != "date" || res.Error.Issues[0].Code != IssueTooBig {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}
}

func TestTimeCoerce(t *testing.T) {
	s := Time(Params{Coerce: true})
	got, err := s.Parse("2022-11-05T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if res := s.SafeParse("not-rfc3339"); res.Success {
		t.Fatal("bad string should fail")
	}
}
