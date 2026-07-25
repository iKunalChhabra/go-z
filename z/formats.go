package z

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// FormatEmail attaches email format check.
func FormatEmail(params ...any) *Check {
	return stringFormatFn("email", patternEmail, reEmailBody, isEmail, params...)
}

// FormatURL attaches URL format check (trims; optionally normalizes).
// Pass URLOpts{Normalize: true} among params to enable href normalization.
// Optional Hostname / Protocol regexes constrain the parsed URL .
func FormatURL(params ...any) *Check {
	p := normalizeParams(params)
	var opts URLOpts
	for _, x := range params {
		switch o := x.(type) {
		case URLOpts:
			mergeURLOpts(&opts, &o)
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
			}
		case *URLOpts:
			if o != nil {
				mergeURLOpts(&opts, o)
				if o.Error != nil {
					p.Error = o.Error
				}
				if o.Abort {
					p.Abort = true
				}
			}
		}
	}
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "url"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok {
			return
		}
		trimmed := strings.TrimSpace(s)

		// When protocol is http(s) and normalize is off, require ://
		if !opts.Normalize && opts.Protocol != nil && opts.Protocol.String() == reHTTPProtocol.String() {
			if !reHTTPSchemeSlashes.MatchString(trimmed) {
				payload.AddIssue(ch.Issue(Issue{
					Code:   IssueInvalidFormat,
					Format: "url",
					Input:  payload.Value,
				}))
				return
			}
		}

		if !isValidURL(trimmed) {
			payload.AddIssue(ch.Issue(Issue{
				Code:   IssueInvalidFormat,
				Format: "url",
				Input:  payload.Value,
			}))
			return
		}
		u, err := url.Parse(trimmed)
		if err != nil || u.Scheme == "" {
			payload.AddIssue(ch.Issue(Issue{
				Code:   IssueInvalidFormat,
				Format: "url",
				Input:  payload.Value,
			}))
			return
		}

		if opts.Hostname != nil && !opts.Hostname.MatchString(u.Hostname()) {
			payload.AddIssue(ch.Issue(Issue{
				Code:    IssueInvalidFormat,
				Format:  "url",
				Pattern: opts.Hostname.String(),
				Input:   payload.Value,
			}))
			if ch.Abort {
				return
			}
		}
		if opts.Protocol != nil && !protocolMatches(opts.Protocol, u.Scheme) {
			payload.AddIssue(ch.Issue(Issue{
				Code:    IssueInvalidFormat,
				Format:  "url",
				Pattern: opts.Protocol.String(),
				Input:   payload.Value,
			}))
			if ch.Abort {
				return
			}
		}

		if opts.Normalize {
			payload.Value = u.String()
		} else {
			payload.Value = trimmed
		}
	}
	return ch
}

// FormatHttpURL is z.httpUrl() — URL restricted to http/https with a
// domain hostname (not bare localhost / IP-only hosts). Protocol/hostname are
// fixed (omits them from httpUrl params); only Normalize/Error/Abort merge.
func FormatHttpURL(params ...any) *Check {
	base := URLOpts{
		Hostname: reDomain,
		Protocol: reHTTPProtocol,
	}
	args := make([]any, 0, len(params)+1)
	for _, x := range params {
		switch o := x.(type) {
		case URLOpts:
			base.Normalize = o.Normalize
			base.Error = o.Error
			base.Abort = o.Abort
		case *URLOpts:
			if o != nil {
				base.Normalize = o.Normalize
				base.Error = o.Error
				base.Abort = o.Abort
			}
		default:
			args = append(args, x)
		}
	}
	return FormatURL(append([]any{base}, args...)...)
}

// URLOpts customizes FormatURL / StringSchema.URL.
type URLOpts struct {
	Normalize bool
	Hostname  *regexp.Regexp // optional hostname constraint
	Protocol  *regexp.Regexp // optional protocol constraint ("https:" or scheme)
	Error     ErrorMap
	Abort     bool
}

func (o URLOpts) params() Params { return Params{Error: o.Error, Abort: o.Abort} }

// formatOpt returns the last option of type T in params, accepting either a
// value or a pointer. Format helpers use it to read their own fields;
// Error/Abort are already folded in by normalizeParams.
func formatOpt[T any](params []any) (T, bool) {
	var out T
	found := false
	for _, raw := range params {
		p, ok := derefParam(raw)
		if !ok {
			continue
		}
		if v, ok := p.(T); ok {
			out, found = v, true
		}
	}
	return out, found
}

func mergeURLOpts(dst, src *URLOpts) {
	if src.Normalize {
		dst.Normalize = true
	}
	if src.Hostname != nil {
		dst.Hostname = src.Hostname
	}
	if src.Protocol != nil {
		dst.Protocol = src.Protocol
	}
	if src.Error != nil {
		dst.Error = src.Error
	}
	if src.Abort {
		dst.Abort = true
	}
}

// protocolMatches tests protocol constraints. Callers may pass a
// regex for "https:" (with colon) or bare "https"; both are accepted.
func protocolMatches(re *regexp.Regexp, scheme string) bool {
	if re == nil {
		return true
	}
	if re.MatchString(scheme + ":") {
		return true
	}
	return re.MatchString(scheme)
}

// StringFormat is z.stringFormat(name, pattern|predicate) — a string schema
// with a custom invalid_format check.
func StringFormat(name string, matcher any, params ...any) *StringSchema {
	p := normalizeParams(params)
	def := &Def{Type: "string", Error: p.Error, Coerce: p.Coerce}
	return newString(def.withChecks(customStringFormat(name, matcher, params...)))
}

func customStringFormat(name string, matcher any, params ...any) *Check {
	switch m := matcher.(type) {
	case *regexp.Regexp:
		return stringFormatPattern(name, m, jsPattern(m), params...)
	case func(string) bool:
		return stringFormatFn(name, "", nil, m, params...)
	default:
		panic(fmt.Sprintf("go-z: StringFormat matcher must be *regexp.Regexp or func(string) bool, got %T", matcher))
	}
}

// FormatHostname attaches hostname format check.
func FormatHostname(params ...any) *Check {
	return stringFormatFn("hostname", patternHostname, reHostnameBody, isHostname, params...)
}

var hashHexLengths = map[string]int{
	"md5":    32,
	"sha1":   40,
	"sha256": 64,
	"sha384": 96,
	"sha512": 128,
}

// FormatHash attaches a hex hash format check (md5|sha1|sha256|sha384|sha512).
func FormatHash(alg string, params ...any) *Check {
	n, ok := hashHexLengths[alg]
	if !ok {
		panic(fmt.Sprintf("go-z: unrecognized hash algorithm %q (want md5|sha1|sha256|sha384|sha512)", alg))
	}
	re := regexp.MustCompile(fmt.Sprintf(`^[0-9a-fA-F]{%d}$`, n))
	format := alg + "_hex"
	return stringFormatPattern(format, re, jsPattern(re), params...)
}

// FormatUUID attaches uuid format check (all versions).
func FormatUUID(params ...any) *Check {
	return stringFormatFn("uuid", jsPattern(reUUID), reUUID, isUUID, params...)
}

// FormatUUIDv4 attaches uuid version 4 (issue format remains "uuid").
func FormatUUIDv4(params ...any) *Check {
	return stringFormatFn("uuid", jsPattern(reUUIDv4), reUUIDv4, func(s string) bool {
		return isUUIDVersion(s, '4')
	}, params...)
}

// FormatUUIDv6 attaches uuid version 6.
func FormatUUIDv6(params ...any) *Check {
	return stringFormatFn("uuid", jsPattern(reUUIDv6), reUUIDv6, func(s string) bool {
		return isUUIDVersion(s, '6')
	}, params...)
}

// FormatUUIDv7 attaches uuid version 7.
func FormatUUIDv7(params ...any) *Check {
	return stringFormatFn("uuid", jsPattern(reUUIDv7), reUUIDv7, func(s string) bool {
		return isUUIDVersion(s, '7')
	}, params...)
}

// FormatGUID attaches guid format check.
func FormatGUID(params ...any) *Check {
	return stringFormatFn("guid", jsPattern(reGUID), reGUID, isGUID, params...)
}

// FormatNanoID attaches nanoid format check.
func FormatNanoID(params ...any) *Check {
	return stringFormatPattern("nanoid", reNanoID, jsPattern(reNanoID), params...)
}

// FormatCUID attaches cuid format check.
func FormatCUID(params ...any) *Check {
	return stringFormatPattern("cuid", reCUID, jsPattern(reCUID), params...)
}

// FormatCUID2 attaches cuid2 format check.
func FormatCUID2(params ...any) *Check {
	return stringFormatPattern("cuid2", reCUID2, jsPattern(reCUID2), params...)
}

// FormatULID attaches ulid format check.
func FormatULID(params ...any) *Check {
	return stringFormatPattern("ulid", reULID, jsPattern(reULID), params...)
}

// FormatKSUID attaches ksuid format check.
func FormatKSUID(params ...any) *Check {
	return stringFormatPattern("ksuid", reKSUID, jsPattern(reKSUID), params...)
}

// FormatXID attaches xid format check.
func FormatXID(params ...any) *Check {
	return stringFormatPattern("xid", reXID, jsPattern(reXID), params...)
}

// FormatBase64 attaches base64 format check (custom validator).
func FormatBase64(params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "base64"
				in.Bag["contentEncoding"] = "base64"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || isValidBase64(s) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Format: "base64",
			Input:  payload.Value,
		}))
	}
	return ch
}

// FormatBase64URL attaches base64url format check.
func FormatBase64URL(params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "base64url"
				in.Bag["contentEncoding"] = "base64url"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || isValidBase64URL(s) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Format: "base64url",
			Input:  payload.Value,
		}))
	}
	return ch
}

// FormatHex attaches hex format check.
func FormatHex(params ...any) *Check {
	return stringFormatPattern("hex", reHex, jsPattern(reHex), params...)
}

// JWTOpts customizes FormatJWT algorithm constraint.
type JWTOpts struct {
	Alg   string
	Error ErrorMap
	Abort bool
}

func (o JWTOpts) params() Params { return Params{Error: o.Error, Abort: o.Abort} }

// FormatJWT attaches jwt format check.
func FormatJWT(params ...any) *Check {
	p := normalizeParams(params)
	alg := ""
	if o, ok := formatOpt[JWTOpts](params); ok {
		alg = o.Alg
	}
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "jwt"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || isValidJWT(s, alg) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Format: "jwt",
			Input:  payload.Value,
		}))
	}
	return ch
}

// FormatE164 attaches e164 format check.
func FormatE164(params ...any) *Check {
	return stringFormatPattern("e164", reE164, jsPattern(reE164), params...)
}

// FormatEmoji attaches emoji format check.
func FormatEmoji(params ...any) *Check {
	return stringFormatFn("emoji", patternEmoji, nil, isEmoji, params...)
}

// FormatIPv4 attaches ipv4 format check.
func FormatIPv4(params ...any) *Check {
	return stringFormatFn("ipv4", jsPattern(reIPv4), reIPv4, isIPv4, params...)
}

// FormatIPv6 attaches an ipv6 format check (URL-based).
func FormatIPv6(params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "ipv6"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || isIPv6(s) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Format: "ipv6",
			Input:  payload.Value,
		}))
	}
	return ch
}

// FormatCIDRv4 attaches cidrv4 format check.
func FormatCIDRv4(params ...any) *Check {
	return stringFormatPattern("cidrv4", reCIDRv4, jsPattern(reCIDRv4), params...)
}

// FormatCIDRv6 attaches cidrv6 format check.
func FormatCIDRv6(params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "string_format",
		Error: p.Error,
		Abort: p.Abort,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["format"] = "cidrv6"
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		s, ok := payload.Value.(string)
		if !ok || isCIDRv6(s) {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:   IssueInvalidFormat,
			Format: "cidrv6",
			Input:  payload.Value,
		}))
	}
	return ch
}

// MACOpts customizes MAC delimiter (default ":").
type MACOpts struct {
	Delimiter string
	Error     ErrorMap
	Abort     bool
}

func (o MACOpts) params() Params { return Params{Error: o.Error, Abort: o.Abort} }

// FormatMAC attaches mac format check.
func FormatMAC(params ...any) *Check {
	p := normalizeParams(params)
	delim := ":"
	if o, ok := formatOpt[MACOpts](params); ok && o.Delimiter != "" {
		delim = o.Delimiter
	}
	var re *regexp.Regexp
	if delim == ":" {
		re = reMACColon
	} else {
		re = macRegexp(delim)
	}
	ch := stringFormatPattern("mac", re, jsPattern(re), params...)
	ch.Error = p.Error
	ch.Abort = p.Abort
	return ch
}

// FormatISODate attaches ISO date format check.
func FormatISODate(params ...any) *Check {
	return stringFormatFn("date", jsPattern(reDate), reDate, isISODate, params...)
}

// ISOTimeOpts customizes time precision (nil = default).
type ISOTimeOpts struct {
	Precision *int
	Error     ErrorMap
	Abort     bool
}

func (o ISOTimeOpts) params() Params { return Params{Error: o.Error, Abort: o.Abort} }

// FormatISOTime attaches ISO time format check.
func FormatISOTime(params ...any) *Check {
	p := normalizeParams(params)
	var precision *int
	if o, ok := formatOpt[ISOTimeOpts](params); ok {
		precision = o.Precision
	}
	re := timeRegexp(precision)
	var ch *Check
	if precision == nil {
		// Default precision is the common case and has a fast matcher.
		ch = stringFormatFn("time", jsPattern(re), re, isISOTimeDefault)
	} else {
		ch = stringFormatPattern("time", re, jsPattern(re))
	}
	ch.Error = p.Error
	ch.Abort = p.Abort
	return ch
}

// ISODateTimeOpts customizes datetime validation.
type ISODateTimeOpts struct {
	Precision *int
	Offset    bool
	Local     bool
	Error     ErrorMap
	Abort     bool
}

func (o ISODateTimeOpts) params() Params { return Params{Error: o.Error, Abort: o.Abort} }

// FormatISODateTime attaches ISO datetime format check.
func FormatISODateTime(params ...any) *Check {
	p := normalizeParams(params)
	var precision *int
	offset, local := false, false
	if o, ok := formatOpt[ISODateTimeOpts](params); ok {
		precision, offset, local = o.Precision, o.Offset, o.Local
	}
	re := datetimeRegexp(precision, offset, local)
	var ch *Check
	if precision == nil && !offset && !local {
		// Default settings are the common case and have a fast matcher.
		ch = stringFormatFn("datetime", jsPattern(re), re, isISODateTimeDefault)
	} else {
		ch = stringFormatPattern("datetime", re, jsPattern(re))
	}
	ch.Error = p.Error
	ch.Abort = p.Abort
	return ch
}

// FormatISODuration attaches ISO duration format check.
func FormatISODuration(params ...any) *Check {
	return stringFormatFn("duration", patternDuration, nil, isISODuration, params...)
}
