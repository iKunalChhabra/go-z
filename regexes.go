package zod

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Zod original regex string literals (for issue.Pattern when RE2 can't compile them).
const (
	patternDuration = `/^P(?:(\d+W)|(?!.*W)(?=\d|T\d)(\d+Y)?(\d+M)?(\d+D)?(T(?=\d)(\d+H)?(\d+M)?(\d+([.,]\d+)?S)?)?)$/`
	patternEmail    = `/^(?!\.)(?!.*\.\.)([A-Za-z0-9_'+\-\.]*)[A-Za-z0-9_+-]@([A-Za-z0-9][A-Za-z0-9\-]*\.)+[A-Za-z]{2,}$/`
	patternEmoji    = `/^(\p{Extended_Pictographic}|\p{Emoji_Component})+$/u`
)

// Precompiled RE2-compatible regexes ported from Zod core/regexes.ts.
var (
	reCUID   = regexp.MustCompile(`^[cC][0-9a-z]{6,}$`)
	reCUID2  = regexp.MustCompile(`^[0-9a-z]+$`)
	reULID   = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26}$`)
	reXID    = regexp.MustCompile(`^[0-9a-vA-V]{20}$`)
	reKSUID  = regexp.MustCompile(`^[A-Za-z0-9]{27}$`)
	reNanoID = regexp.MustCompile(`^[a-zA-Z0-9_-]{21}$`)

	reGUID = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`)

	reUUID   = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$`)
	reUUIDv4 = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`)
	reUUIDv6 = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-6[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`)
	reUUIDv7 = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`)

	// Email body without lookaheads (enforced in isEmail).
	reEmailBody = regexp.MustCompile(`^([A-Za-z0-9_'+\-\.]*)[A-Za-z0-9_+-]@([A-Za-z0-9][A-Za-z0-9\-]*\.)+[A-Za-z]{2,}$`)

	reIPv4 = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])$`)

	reCIDRv4 = regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\/([0-9]|[1-2][0-9]|3[0-2])$`)

	reBase64    = regexp.MustCompile(`^$|^(?:[0-9a-zA-Z+/]{4})*(?:(?:[0-9a-zA-Z+/]{2}==)|(?:[0-9a-zA-Z+/]{3}=))?$`)
	reBase64URL = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

	reE164 = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

	reDate = regexp.MustCompile(`^(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))$`)

	reLowercase = regexp.MustCompile(`^[^A-Z]*$`)
	reUppercase = regexp.MustCompile(`^[^a-z]*$`)
	reHex       = regexp.MustCompile(`^[0-9a-fA-F]*$`)

	reMACColon = macRegexp(":")

	reDurationWeeks = regexp.MustCompile(`^\d+W$`)
)

const dateSource = `(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))`

// jsPattern returns a Zod/JS-style `/source/` string for issue.Pattern fields.
func jsPattern(re *regexp.Regexp) string {
	if re == nil {
		return ""
	}
	return "/" + re.String() + "/"
}

func macRegexp(delimiter string) *regexp.Regexp {
	esc := regexp.QuoteMeta(delimiter)
	return regexp.MustCompile(`^(?:[0-9A-F]{2}` + esc + `){5}[0-9A-F]{2}$|^(?:[0-9a-f]{2}` + esc + `){5}[0-9a-f]{2}$`)
}

// isEmail ports Zod's practical email regex; lookaheads are checked explicitly.
func isEmail(s string) bool {
	if s == "" || strings.HasPrefix(s, ".") || strings.Contains(s, "..") {
		return false
	}
	return reEmailBody.MatchString(s)
}

// isISODuration ports Zod's duration regex (lookaheads hand-coded).
func isISODuration(s string) bool {
	if !strings.HasPrefix(s, "P") || s == "P" {
		return false
	}
	rest := s[1:]
	if reDurationWeeks.MatchString(rest) {
		return true
	}
	if strings.Contains(rest, "W") {
		return false
	}
	if rest == "" {
		return false
	}
	if rest[0] != 'T' && (rest[0] < '0' || rest[0] > '9') {
		return false
	}
	if rest[0] == 'T' && (len(rest) < 2 || rest[1] < '0' || rest[1] > '9') {
		return false
	}

	i := 0
	consumeNumUnit := func(unit byte) bool {
		start := i
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i == start || i >= len(rest) || rest[i] != unit {
			i = start
			return false
		}
		i++
		return true
	}
	_ = consumeNumUnit('Y')
	_ = consumeNumUnit('M')
	_ = consumeNumUnit('D')
	if i < len(rest) {
		if rest[i] != 'T' {
			return false
		}
		i++
		if i >= len(rest) || rest[i] < '0' || rest[i] > '9' {
			return false
		}
		_ = consumeNumUnit('H')
		_ = consumeNumUnit('M')
		// seconds: \d+([.,]\d+)?S
		start := i
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i > start {
			if i < len(rest) && (rest[i] == '.' || rest[i] == ',') {
				i++
				frac := i
				for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
					i++
				}
				if i == frac {
					return false
				}
			}
			if i >= len(rest) || rest[i] != 'S' {
				return false
			}
			i++
		}
	}
	return i == len(rest)
}

// isEmoji ports Zod's emoji regex (Extended_Pictographic | Emoji_Component).
func isEmoji(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isExtendedPictographic(r) && !isEmojiComponent(r) {
			return false
		}
	}
	return true
}

func isEmojiComponent(r rune) bool {
	switch {
	case r == '#' || r == '*':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == 0x200D || r == 0x20E3:
		return true
	case r == 0xFE0E || r == 0xFE0F:
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return true
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return true
	case r >= 0x1F9B0 && r <= 0x1F9B3:
		return true
	case r >= 0xE0020 && r <= 0xE007F:
		return true
	}
	return false
}

func isExtendedPictographic(r rune) bool {
	switch {
	case r == 0x00A9 || r == 0x00AE:
		return true
	case r == 0x203C || r == 0x2049:
		return true
	case r == 0x2122 || r == 0x2139:
		return true
	case r >= 0x2194 && r <= 0x2199:
		return true
	case r == 0x21A9 || r == 0x21AA:
		return true
	case r == 0x231A || r == 0x231B || r == 0x2328 || r == 0x23CF:
		return true
	case r >= 0x23E9 && r <= 0x23F3:
		return true
	case r >= 0x23F8 && r <= 0x23FA:
		return true
	case r == 0x24C2:
		return true
	case r == 0x25AA || r == 0x25AB || r == 0x25B6 || r == 0x25C0:
		return true
	case r >= 0x25FB && r <= 0x25FE:
		return true
	case r >= 0x2600 && r <= 0x27BF:
		return true
	case r == 0x2934 || r == 0x2935:
		return true
	case r >= 0x2B05 && r <= 0x2B07:
		return true
	case r == 0x2B1B || r == 0x2B1C || r == 0x2B50 || r == 0x2B55:
		return true
	case r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299:
		return true
	case r >= 0x1F000 && r <= 0x1FAFF:
		return true
	}
	return unicode.Is(unicode.So, r) && r >= 0x1F300
}

// isValidURL mirrors Zod $ZodURL (trim + absolute URL parse).
func isValidURL(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || !strings.Contains(trimmed, ":") {
		return false
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "http" || scheme == "https" {
		return u.Host != ""
	}
	return true
}

// normalizeURLHref returns the trimmed input or a normalized href.
func normalizeURLHref(s string, normalize bool) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if !isValidURL(trimmed) {
		return "", false
	}
	if !normalize {
		return trimmed, true
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	return u.String(), true
}

// isIPv6 mirrors Zod's URL(`http://[addr]`) check. IPv4-mapped IPv6 forms
// like ::ffff:192.168.0.1 are accepted (ParseIP.To4() is non-nil for those).
func isIPv6(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	return net.ParseIP(s) != nil
}

// isCIDRv6 mirrors Zod's split/prefix/URL check.
func isCIDRv6(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return false
	}
	addr, prefix := parts[0], parts[1]
	if prefix == "" {
		return false
	}
	n, err := strconv.Atoi(prefix)
	if err != nil || strconv.Itoa(n) != prefix {
		return false
	}
	if n < 0 || n > 128 {
		return false
	}
	return isIPv6(addr)
}

// isValidBase64 mirrors Zod's isValidBase64 helper.
func isValidBase64(data string) bool {
	if data == "" {
		return true
	}
	if strings.ContainsAny(data, " \t\n\r") {
		return false
	}
	if len(data)%4 != 0 {
		return false
	}
	if !reBase64.MatchString(data) {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil
}

// isValidBase64URL mirrors Zod's isValidBase64URL helper.
func isValidBase64URL(data string) bool {
	if !reBase64URL.MatchString(data) {
		return false
	}
	b64 := strings.NewReplacer("-", "+", "_", "/").Replace(data)
	pad := (4 - len(b64)%4) % 4
	b64 += strings.Repeat("=", pad)
	return isValidBase64(b64)
}

// isValidJWT mirrors Zod's isValidJWT.
func isValidJWT(token string, algorithm string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	header := parts[0]
	if header == "" {
		return false
	}
	raw, err := decodeBase64URL(header)
	if err != nil {
		return false
	}
	var hdr map[string]any
	if err := json.Unmarshal(raw, &hdr); err != nil {
		return false
	}
	if typ, ok := hdr["typ"]; ok {
		ts, ok := typ.(string)
		if !ok || ts != "JWT" {
			return false
		}
	}
	alg, _ := hdr["alg"].(string)
	if alg == "" {
		return false
	}
	if algorithm != "" && alg != algorithm {
		return false
	}
	return true
}

func decodeBase64URL(s string) ([]byte, error) {
	// JWT headers are base64url; try RawURLEncoding then URLEncoding.
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// Zod uses atob (standard base64) — also try that.
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

// timeRegexp builds Zod's time regex for the given precision.
func timeRegexp(precision *int) *regexp.Regexp {
	hhmm := `(?:[01]\d|2[0-3]):[0-5]\d`
	var body string
	if precision == nil {
		body = hhmm + `(?::[0-5]\d(?:\.\d+)?)?`
	} else {
		switch *precision {
		case -1:
			body = hhmm
		case 0:
			body = hhmm + `:[0-5]\d`
		default:
			body = hhmm + `:[0-5]\d\.\d{` + strconv.Itoa(*precision) + `}`
		}
	}
	return regexp.MustCompile(`^` + body + `$`)
}

// datetimeRegexp builds Zod's datetime regex.
func datetimeRegexp(precision *int, offset, local bool) *regexp.Regexp {
	hhmm := `(?:[01]\d|2[0-3]):[0-5]\d`
	var timePart string
	if precision == nil {
		timePart = hhmm + `(?::[0-5]\d(?:\.\d+)?)?`
	} else {
		switch *precision {
		case -1:
			timePart = hhmm
		case 0:
			timePart = hhmm + `:[0-5]\d`
		default:
			timePart = hhmm + `:[0-5]\d\.\d{` + strconv.Itoa(*precision) + `}`
		}
	}
	opts := []string{"Z"}
	if local {
		opts = append(opts, "")
	}
	if offset {
		opts = append(opts, `([+-](?:[01]\d|2[0-3]):[0-5]\d)`)
	}
	timeRegex := timePart + `(?:` + strings.Join(opts, "|") + `)`
	return regexp.MustCompile(`^` + dateSource + `T(?:` + timeRegex + `)$`)
}
