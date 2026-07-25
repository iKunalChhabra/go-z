package zod

import (
	"math/rand"
	"regexp"
	"testing"
)

// The hand-written matchers must accept exactly the language of the regex they
// replace. Each test pins known edge cases and then differential-tests against
// the regex over a large random corpus drawn from an alphabet that exercises
// every branch (separators, boundary characters, wrong case, invalid bytes).

func differential(t *testing.T, name string, fast func(string) bool, re *regexp.Regexp, fixed []string, gen func(*rand.Rand) string, iterations int) {
	t.Helper()
	for _, s := range fixed {
		if got, want := fast(s), re.MatchString(s); got != want {
			t.Errorf("%s(%q) = %v, regex = %v", name, s, got, want)
		}
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < iterations; i++ {
		s := gen(rng)
		if got, want := fast(s), re.MatchString(s); got != want {
			t.Fatalf("%s(%q) = %v, regex = %v", name, s, got, want)
		}
	}
}

func randomFrom(rng *rand.Rand, alphabet string, minLen, maxLen int) string {
	n := minLen + rng.Intn(maxLen-minLen+1)
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rng.Intn(len(alphabet))]
	}
	return string(b)
}

func TestIsEmailMatchesRegex(t *testing.T) {
	// isEmail also enforces the two lookaheads the regex cannot express, so
	// compare against the same composition the old implementation used.
	reference := func(s string) bool {
		if s == "" || s[0] == '.' {
			return false
		}
		for i := 0; i+1 < len(s); i++ {
			if s[i] == '.' && s[i+1] == '.' {
				return false
			}
		}
		return reEmailBody.MatchString(s)
	}
	fixed := []string{
		"", ".", "a@b.co", "a@b.c", "foo.bar@example.com", "foo..bar@example.com",
		".foo@example.com", "foo.@example.com", "foo@.example.com", "foo@example..com",
		"foo@example.com.", "foo@-example.com", "foo@example-.com", "foo@ex-ample.co.uk",
		"foo+tag@example.io", "foo'bar@example.io", "foo_bar@example.io", "foo-bar@example.io",
		"@example.com", "foo@", "foo@com", "foo@@example.com", "foo@exa_mple.com",
		"UPPER@EXAMPLE.COM", "1@2.co", "a@b.c1", "a@b.1c", "üñí@example.com",
	}
	rng := rand.New(rand.NewSource(3))
	for _, s := range fixed {
		if got, want := isEmail(s), reference(s); got != want {
			t.Errorf("isEmail(%q) = %v, reference = %v", s, got, want)
		}
	}
	for i := 0; i < 300000; i++ {
		s := randomFrom(rng, "abzAZ09._'+-@\x7f", 1, 15)
		if got, want := isEmail(s), reference(s); got != want {
			t.Fatalf("isEmail(%q) = %v, reference = %v", s, got, want)
		}
	}
}

func TestIsUUIDMatchesRegex(t *testing.T) {
	fixed := []string{
		"", "not-a-uuid",
		"123e4567-e89b-12d3-a456-426614174000",
		"123e4567-e89b-42d3-a456-426614174000",
		"123E4567-E89B-42D3-A456-426614174000",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", // max UUID is lowercase-only
		"123e4567-e89b-92d3-a456-426614174000",
		"123e4567-e89b-02d3-a456-426614174000",
		"123e4567-e89b-42d3-c456-426614174000",
		"123e4567e89b42d3a456426614174000",
		"123e4567-e89b-42d3-a456-42661417400",
	}
	differential(t, "isUUID", isUUID, reUUID, fixed, func(rng *rand.Rand) string {
		s := []byte(randomFrom(rng, "0123456789abcdefABCDEFgx-", 34, 38))
		if len(s) == 36 && rng.Intn(2) == 0 {
			s[8], s[13], s[18], s[23] = '-', '-', '-', '-'
		}
		return string(s)
	}, 300000)
}

func TestIsUUIDVersionMatchesRegex(t *testing.T) {
	for _, tc := range []struct {
		version byte
		re      *regexp.Regexp
	}{{'4', reUUIDv4}, {'6', reUUIDv6}, {'7', reUUIDv7}} {
		version := tc.version
		fast := func(s string) bool { return isUUIDVersion(s, version) }
		fixed := []string{
			"123e4567-e89b-" + string(version) + "2d3-a456-426614174000",
			"123e4567-e89b-12d3-a456-426614174000",
			"00000000-0000-0000-0000-000000000000",
			"ffffffff-ffff-ffff-ffff-ffffffffffff",
		}
		differential(t, "isUUIDVersion("+string(version)+")", fast, tc.re, fixed, func(rng *rand.Rand) string {
			s := []byte(randomFrom(rng, "01234789abcdefABCDEF-", 36, 36))
			if rng.Intn(2) == 0 {
				s[8], s[13], s[18], s[23] = '-', '-', '-', '-'
			}
			return string(s)
		}, 200000)
	}
}

func TestIsGUIDMatchesRegex(t *testing.T) {
	fixed := []string{
		"", "123e4567-e89b-12d3-a456-426614174000",
		"123e4567-e89b-92d3-c456-426614174000",
		"00000000-0000-0000-0000-000000000000",
		"123e4567e89b42d3a456426614174000",
	}
	differential(t, "isGUID", isGUID, reGUID, fixed, func(rng *rand.Rand) string {
		s := []byte(randomFrom(rng, "0123456789abcdefABCDEFzZ-", 35, 37))
		if len(s) == 36 && rng.Intn(2) == 0 {
			s[8], s[13], s[18], s[23] = '-', '-', '-', '-'
		}
		return string(s)
	}, 200000)
}

func TestIsIPv4MatchesRegex(t *testing.T) {
	fixed := []string{
		"", "0.0.0.0", "255.255.255.255", "192.168.1.1", "1.2.3.4",
		"256.1.1.1", "01.2.3.4", "1.2.3", "1.2.3.4.5", "1.2.3.", ".1.2.3",
		"1..2.3", "1.2.3.4 ", "a.b.c.d", "999.999.999.999", "10.0.0.01",
	}
	differential(t, "isIPv4", isIPv4, reIPv4, fixed, func(rng *rand.Rand) string {
		return randomFrom(rng, "0129.5", 1, 16)
	}, 300000)
}

// Realistic-shaped corpora: the random alphabets above rarely produce valid
// values, so also fuzz around well-formed inputs with single mutations.
func TestMatchersAgainstMutatedValidValues(t *testing.T) {
	seeds := map[string]struct {
		valid []string
		fast  func(string) bool
		re    *regexp.Regexp
	}{
		"uuid": {
			valid: []string{"123e4567-e89b-42d3-a456-426614174000", "00000000-0000-0000-0000-000000000000"},
			fast:  isUUID,
			re:    reUUID,
		},
		"ipv4": {
			valid: []string{"192.168.100.14", "8.8.8.8", "255.0.255.0"},
			fast:  isIPv4,
			re:    reIPv4,
		},
	}
	rng := rand.New(rand.NewSource(5))
	mutations := "0189abfFzZ-.:"
	for name, seed := range seeds {
		for i := 0; i < 200000; i++ {
			base := []byte(seed.valid[rng.Intn(len(seed.valid))])
			switch rng.Intn(3) {
			case 0:
				base[rng.Intn(len(base))] = mutations[rng.Intn(len(mutations))]
			case 1:
				base = base[:rng.Intn(len(base))]
			case 2:
				base = append(base, mutations[rng.Intn(len(mutations))])
			}
			s := string(base)
			if got, want := seed.fast(s), seed.re.MatchString(s); got != want {
				t.Fatalf("%s(%q) = %v, regex = %v", name, s, got, want)
			}
		}
	}
}
