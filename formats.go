package zod

import "regexp"

// FormatEmail attaches Zod's email format check.
func FormatEmail(params ...any) *Check {
	return stringFormatFn("email", patternEmail, isEmail, params...)
}

// FormatURL attaches Zod's URL format check (trims; optionally normalizes).
// Pass URLOpts{Normalize: true} among params to enable href normalization.
func FormatURL(params ...any) *Check {
	p := normalizeParams(params)
	normalize := false
	for _, x := range params {
		if o, ok := x.(URLOpts); ok {
			normalize = o.Normalize
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
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
		out, ok := normalizeURLHref(s, normalize)
		if !ok {
			// URL issues omit Origin (matches Zod $ZodURL).
			payload.AddIssue(ch.Issue(Issue{
				Code:   IssueInvalidFormat,
				Format: "url",
				Input:  payload.Value,
			}))
			return
		}
		payload.Value = out
	}
	return ch
}

// URLOpts customizes FormatURL / StringSchema.URL.
type URLOpts struct {
	Normalize bool
	Error     ErrorMap
	Abort     bool
}

// FormatUUID attaches Zod's uuid format check (all versions).
func FormatUUID(params ...any) *Check {
	return stringFormatPattern("uuid", reUUID, jsPattern(reUUID), params...)
}

// FormatUUIDv4 attaches uuid version 4 (issue format remains "uuid").
func FormatUUIDv4(params ...any) *Check {
	return stringFormatPattern("uuid", reUUIDv4, jsPattern(reUUIDv4), params...)
}

// FormatUUIDv6 attaches uuid version 6.
func FormatUUIDv6(params ...any) *Check {
	return stringFormatPattern("uuid", reUUIDv6, jsPattern(reUUIDv6), params...)
}

// FormatUUIDv7 attaches uuid version 7.
func FormatUUIDv7(params ...any) *Check {
	return stringFormatPattern("uuid", reUUIDv7, jsPattern(reUUIDv7), params...)
}

// FormatGUID attaches Zod's guid format check.
func FormatGUID(params ...any) *Check {
	return stringFormatPattern("guid", reGUID, jsPattern(reGUID), params...)
}

// FormatNanoID attaches Zod's nanoid format check.
func FormatNanoID(params ...any) *Check {
	return stringFormatPattern("nanoid", reNanoID, jsPattern(reNanoID), params...)
}

// FormatCUID attaches Zod's cuid format check.
func FormatCUID(params ...any) *Check {
	return stringFormatPattern("cuid", reCUID, jsPattern(reCUID), params...)
}

// FormatCUID2 attaches Zod's cuid2 format check.
func FormatCUID2(params ...any) *Check {
	return stringFormatPattern("cuid2", reCUID2, jsPattern(reCUID2), params...)
}

// FormatULID attaches Zod's ulid format check.
func FormatULID(params ...any) *Check {
	return stringFormatPattern("ulid", reULID, jsPattern(reULID), params...)
}

// FormatKSUID attaches Zod's ksuid format check.
func FormatKSUID(params ...any) *Check {
	return stringFormatPattern("ksuid", reKSUID, jsPattern(reKSUID), params...)
}

// FormatXID attaches Zod's xid format check.
func FormatXID(params ...any) *Check {
	return stringFormatPattern("xid", reXID, jsPattern(reXID), params...)
}

// FormatBase64 attaches Zod's base64 format check (custom validator).
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

// FormatBase64URL attaches Zod's base64url format check.
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

// FormatHex attaches Zod's hex format check.
func FormatHex(params ...any) *Check {
	return stringFormatPattern("hex", reHex, jsPattern(reHex), params...)
}

// JWTOpts customizes FormatJWT algorithm constraint.
type JWTOpts struct {
	Alg   string
	Error ErrorMap
	Abort bool
}

// FormatJWT attaches Zod's jwt format check.
func FormatJWT(params ...any) *Check {
	p := normalizeParams(params)
	alg := ""
	for _, x := range params {
		switch o := x.(type) {
		case JWTOpts:
			alg = o.Alg
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
			}
		case map[string]string:
			alg = o["alg"]
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

// FormatE164 attaches Zod's e164 format check.
func FormatE164(params ...any) *Check {
	return stringFormatPattern("e164", reE164, jsPattern(reE164), params...)
}

// FormatEmoji attaches Zod's emoji format check.
func FormatEmoji(params ...any) *Check {
	return stringFormatFn("emoji", patternEmoji, isEmoji, params...)
}

// FormatIPv4 attaches Zod's ipv4 format check.
func FormatIPv4(params ...any) *Check {
	return stringFormatPattern("ipv4", reIPv4, jsPattern(reIPv4), params...)
}

// FormatIPv6 attaches Zod's ipv6 format check (URL-based, like Zod).
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

// FormatCIDRv4 attaches Zod's cidrv4 format check.
func FormatCIDRv4(params ...any) *Check {
	return stringFormatPattern("cidrv4", reCIDRv4, jsPattern(reCIDRv4), params...)
}

// FormatCIDRv6 attaches Zod's cidrv6 format check.
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

// FormatMAC attaches Zod's mac format check.
func FormatMAC(params ...any) *Check {
	p := normalizeParams(params)
	delim := ":"
	for _, x := range params {
		if o, ok := x.(MACOpts); ok {
			if o.Delimiter != "" {
				delim = o.Delimiter
			}
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
			}
		}
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

// FormatISODate attaches Zod's ISO date format check.
func FormatISODate(params ...any) *Check {
	return stringFormatPattern("date", reDate, jsPattern(reDate), params...)
}

// ISOTimeOpts customizes time precision (nil = Zod default).
type ISOTimeOpts struct {
	Precision *int
	Error     ErrorMap
	Abort     bool
}

// FormatISOTime attaches Zod's ISO time format check.
func FormatISOTime(params ...any) *Check {
	p := normalizeParams(params)
	var precision *int
	for _, x := range params {
		if o, ok := x.(ISOTimeOpts); ok {
			precision = o.Precision
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
			}
		}
	}
	re := timeRegexp(precision)
	ch := stringFormatPattern("time", re, jsPattern(re))
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

// FormatISODateTime attaches Zod's ISO datetime format check.
func FormatISODateTime(params ...any) *Check {
	p := normalizeParams(params)
	var precision *int
	offset, local := false, false
	for _, x := range params {
		if o, ok := x.(ISODateTimeOpts); ok {
			precision = o.Precision
			offset = o.Offset
			local = o.Local
			if o.Error != nil {
				p.Error = o.Error
			}
			if o.Abort {
				p.Abort = true
			}
		}
	}
	re := datetimeRegexp(precision, offset, local)
	ch := stringFormatPattern("datetime", re, jsPattern(re))
	ch.Error = p.Error
	ch.Abort = p.Abort
	return ch
}

// FormatISODuration attaches Zod's ISO duration format check.
func FormatISODuration(params ...any) *Check {
	return stringFormatFn("duration", patternDuration, isISODuration, params...)
}
