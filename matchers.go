package zod

import "strings"

// Hand-written matchers for the formats that dominate real schemas.
//
// Go's regexp engine backtracks for these patterns, and email alone accounted
// for roughly 40% of the CPU in a four-field object parse. Each matcher here
// accepts exactly the same language as the regex it replaces
// (matchers_test.go differential-tests them against the regex over hundreds of
// thousands of random inputs), so the regexes stay in regexes.go as the
// specification and as the `pattern` reported on issues.

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

func isASCIIAlnum(c byte) bool { return isASCIIAlpha(c) || isASCIIDigit(c) }

func isASCIIHex(c byte) bool {
	return isASCIIDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// emailLocalByte is the local-part character class [A-Za-z0-9_'+\-.].
func emailLocalByte(c byte) bool {
	return isASCIIAlnum(c) || c == '_' || c == '\'' || c == '+' || c == '-' || c == '.'
}

// emailLocalLastByte is the required final local character [A-Za-z0-9_+-]:
// the local part may end in neither a dot nor an apostrophe.
func emailLocalLastByte(c byte) bool {
	return isASCIIAlnum(c) || c == '_' || c == '+' || c == '-'
}

// isEmail ports Zod's practical email regex:
//
//	^(?!\.)(?!.*\.\.)([A-Za-z0-9_'+\-\.]*)[A-Za-z0-9_+-]@([A-Za-z0-9][A-Za-z0-9\-]*\.)+[A-Za-z]{2,}$
func isEmail(s string) bool {
	if s == "" || s[0] == '.' || strings.Contains(s, "..") {
		return false
	}
	at := strings.IndexByte(s, '@')
	// The local part must be non-empty and the domain must have room for
	// "x.yy", so '@' can be neither first nor last.
	if at <= 0 || at >= len(s)-1 {
		return false
	}
	local, domain := s[:at], s[at+1:]
	for i := 0; i < len(local); i++ {
		if !emailLocalByte(local[i]) {
			return false
		}
	}
	if !emailLocalLastByte(local[len(local)-1]) {
		return false
	}
	return isEmailDomain(domain)
}

// isEmailDomain matches ([A-Za-z0-9][A-Za-z0-9-]*\.)+[A-Za-z]{2,}.
func isEmailDomain(domain string) bool {
	lastDot := strings.LastIndexByte(domain, '.')
	if lastDot <= 0 || lastDot == len(domain)-1 {
		return false
	}
	tld := domain[lastDot+1:]
	if len(tld) < 2 {
		return false
	}
	for i := 0; i < len(tld); i++ {
		if !isASCIIAlpha(tld[i]) {
			return false
		}
	}
	labels := domain[:lastDot]
	start := 0
	for i := 0; i <= len(labels); i++ {
		if i < len(labels) && labels[i] != '.' {
			continue
		}
		label := labels[start:i]
		if len(label) == 0 || !isASCIIAlnum(label[0]) {
			return false
		}
		for j := 1; j < len(label); j++ {
			if c := label[j]; !isASCIIAlnum(c) && c != '-' {
				return false
			}
		}
		start = i + 1
	}
	return true
}

// uuidShape reports whether s has the 8-4-4-4-12 hex layout.
func uuidShape(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !isASCIIHex(s[i]) {
			return false
		}
	}
	return true
}

// uuidVariant reports whether the variant nibble is one of 8, 9, a, b.
func uuidVariant(c byte) bool {
	switch c {
	case '8', '9', 'a', 'b', 'A', 'B':
		return true
	}
	return false
}

// uuidRepeated reports whether every hex position holds want — the nil
// (all zero) and max (all lowercase f) UUIDs, which the regex allows as
// escapes. Zod's pattern spells the max UUID in lowercase only, so
// "FFFFFFFF-..." is not accepted.
func uuidRepeated(s string, want byte) bool {
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if s[i] != want {
			return false
		}
	}
	return true
}

// isUUID ports Zod's uuid regex: any version 1-8 with a valid variant, plus
// the nil and max UUIDs.
func isUUID(s string) bool {
	if !uuidShape(s) {
		return false
	}
	if v := s[14]; v >= '1' && v <= '8' && uuidVariant(s[19]) {
		return true
	}
	return uuidRepeated(s, '0') || uuidRepeated(s, 'f')
}

// isUUIDVersion ports the versioned uuid regexes (v4/v6/v7): a fixed version
// nibble and a valid variant, with no nil/max escape.
func isUUIDVersion(s string, version byte) bool {
	return uuidShape(s) && s[14] == version && uuidVariant(s[19])
}

// isGUID ports Zod's guid regex: the 8-4-4-4-12 hex layout with no version or
// variant constraint.
func isGUID(s string) bool { return uuidShape(s) }

// isISODate ports Zod's date regex, whose alternations encode the Gregorian
// leap-year rule: YYYY-MM-DD with a real calendar day.
func isISODate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for _, i := range [8]int{0, 1, 2, 3, 5, 6, 8, 9} {
		if !isASCIIDigit(s[i]) {
			return false
		}
	}
	year := int(s[0]-'0')*1000 + int(s[1]-'0')*100 + int(s[2]-'0')*10 + int(s[3]-'0')
	month := int(s[5]-'0')*10 + int(s[6]-'0')
	day := int(s[8]-'0')*10 + int(s[9]-'0')
	if month < 1 || month > 12 || day < 1 {
		return false
	}
	return day <= daysInMonth(year, month)
}

func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	default: // February
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
}

// isISOTimeDefault ports Zod's time regex at the default precision:
// HH:MM, optionally :SS, optionally with a fractional part.
func isISOTimeDefault(s string) bool {
	if len(s) < 5 || s[2] != ':' {
		return false
	}
	if !isASCIIDigit(s[0]) || !isASCIIDigit(s[1]) || !isASCIIDigit(s[3]) || !isASCIIDigit(s[4]) {
		return false
	}
	if hour := int(s[0]-'0')*10 + int(s[1]-'0'); hour > 23 {
		return false
	}
	if s[3] > '5' {
		return false
	}
	if len(s) == 5 {
		return true
	}
	// Seconds are optional, and only present with a leading colon.
	if s[5] != ':' || len(s) < 8 {
		return false
	}
	if !isASCIIDigit(s[6]) || !isASCIIDigit(s[7]) || s[6] > '5' {
		return false
	}
	if len(s) == 8 {
		return true
	}
	// Fractional seconds: a dot followed by at least one digit.
	if s[8] != '.' || len(s) == 9 {
		return false
	}
	for i := 9; i < len(s); i++ {
		if !isASCIIDigit(s[i]) {
			return false
		}
	}
	return true
}

// isISODateTimeDefault ports Zod's datetime regex at its default settings
// (no fixed precision, no offset, not local): <date>T<time>Z.
func isISODateTimeDefault(s string) bool {
	if len(s) < 17 || s[10] != 'T' || s[len(s)-1] != 'Z' {
		return false
	}
	return isISODate(s[:10]) && isISOTimeDefault(s[11:len(s)-1])
}

// isIPv4 ports Zod's ipv4 regex: four dot-separated decimal octets, each
// 0-255 with no leading zeros.
func isIPv4(s string) bool {
	octets := 0
	for len(s) > 0 {
		if octets == 4 {
			return false
		}
		end := strings.IndexByte(s, '.')
		var part string
		if end < 0 {
			part, s = s, ""
		} else {
			part, s = s[:end], s[end+1:]
			if s == "" {
				return false // trailing dot
			}
		}
		if !isIPv4Octet(part) {
			return false
		}
		octets++
	}
	return octets == 4
}

func isIPv4Octet(part string) bool {
	if len(part) == 0 || len(part) > 3 {
		return false
	}
	if len(part) > 1 && part[0] == '0' {
		return false // leading zero
	}
	n := 0
	for i := 0; i < len(part); i++ {
		if !isASCIIDigit(part[i]) {
			return false
		}
		n = n*10 + int(part[i]-'0')
	}
	return n <= 255
}
