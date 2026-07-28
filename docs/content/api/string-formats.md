# String formats

Format helpers attach schema-compatible `invalid_format` checks to a [`String`](/api/string) schema. Call them as fluent methods on `z.String()`.

```go
email := z.String().Email()
email.MustParse("user@example.com")

res := email.SafeParse("not-an-email")
// Code: invalid_format
// Format: "email"
// Message: "Invalid email address"
```

:::tip Format field
Every failed format check sets `Issue.Format` to the string in the table below. Use it for API clients, analytics, or custom error maps.
:::

## Format → `Issue.Format`

| Method | `Issue.Format` | Typical success | Typical failure message |
|--------|----------------|-----------------|-------------------------|
| `Email` | `email` | `a@b.cd` | `Invalid email address` |
| `URL` | `url` | `https://example.com` | `Invalid URL` |
| `HttpURL` | `url` | `https://example.com` (http/https only) | `Invalid URL` |
| `Hostname` | `hostname` | `api.example.com` | (locale default) |
| `Hash` | `hash` | hex digest of the given algorithm | (locale default) |
| `UUID` | `uuid` | any RFC UUID (v1–v8-ish) | `Invalid UUID` |
| `UUIDv4` | `uuid` | version **4** only | `Invalid UUID` |
| `UUIDv6` | `uuid` | version **6** only | `Invalid UUID` |
| `UUIDv7` | `uuid` | version **7** only | `Invalid UUID` |
| `GUID` | `guid` | looser GUID (variant 0 OK) | (locale default) |
| `NanoID` | `nanoid` | `lfNZluvAxMkf7Q8C5H-QS` | (locale default) |
| `CUID` | `cuid` | `ckopqwoedu0013g5hseu82ta1` | `Invalid cuid` |
| `CUID2` | `cuid2` | lowercase cuid2 | (locale default) |
| `ULID` | `ulid` | `01ARZ3NDEKTSV4RRFFQ69G5FAV` | `Invalid ULID` |
| `KSUID` | `ksuid` | 27-char KSUID | `Invalid KSUID` |
| `XID` | `xid` | `9m4e2mr0ui3e8a215n4g` | `Invalid XID` |
| `Base64` | `base64` | `SGVsbG8gV29ybGQ=` | `Invalid base64-encoded string` |
| `Base64URL` | `base64url` | URL-safe alphabet | (locale default) |
| `Hex` | `hex` | `DEADBEEF`, `""` | (locale default) |
| `JWT` | `jwt` | three base64url segments | (locale default) |
| `E164` | `e164` | `+14155552671` | (locale default) |
| `Emoji` | `emoji` | emoji-only string | (locale default) |
| `IPv4` | `ipv4` | `192.168.0.1` | (locale default) |
| `IPv6` | `ipv6` | `::1`, compressed forms | (locale default) |
| `CIDRv4` | `cidrv4` | `10.0.0.0/8` | (locale default) |
| `CIDRv6` | `cidrv6` | IPv6 prefix | (locale default) |
| `MAC` | `mac` | `01:23:45:67:89:ab` | (locale default) |
| `ISODate` | `date` | `2024-01-15` | (locale default) |
| `ISOTime` | `time` | `14:30:00` | (locale default) |
| `ISODateTime` | `datetime` | `2024-01-15T14:30:00Z` | (locale default) |
| `ISODuration` | `duration` | `P1DT2H` | (locale default) |

:::info UUID methods share format `"uuid"`
`UUIDv4`, `UUIDv6`, and `UUIDv7` tighten the pattern but still report `Format: "uuid"` — matching the upstream library.
:::

## Email

```go
email := z.String().Email()

email.MustParse("firstname+lastname@domain.com")
email.MustParse("x@example.com")

email.MustParse("email@domain.com")

res := email.SafeParse("plainaddress")
// Format == "email"
// Message == "Invalid email address"

_ = email.SafeParse("@domain.com")             // fail
_ = email.SafeParse("email..email@domain.com") // fail
```

## URL

`URL` trims surrounding whitespace. Pass `z.URLOpts{Normalize: true}` to rewrite via Go’s URL parser (href-style normalization).

```go
u := z.String().URL()
u.MustParse("https://google.com/asdf?q=1#hash")
u.MustParse("http://localhost")

got := u.MustParse("  https://example.com  ")
// got == "https://example.com"

res := u.SafeParse("asdf")
// Message: "Invalid URL"
// Note: URL issues omit Origin

norm := z.String().URL(z.URLOpts{Normalize: true})
_ = norm.MustParse("https://example.com?key=value")
```

## HttpURL / Hostname

`HttpURL` is `URL` restricted to the `http` and `https` schemes with a domain hostname — handy for webhooks and callbacks where `ftp://` or `file://` must not pass.

```go
h := z.String().HttpURL()
h.MustParse("https://api.example.com/hook")
_ = h.SafeParse("ftp://example.com/file") // fail — wrong scheme

host := z.String().Hostname()
host.MustParse("api.example.com")
_ = host.SafeParse("not a host!") // fail
```

## Hash

`Hash(alg)` validates a lowercase/uppercase hex digest of the given algorithm. Supported algorithms: `md5`, `sha1`, `sha256`, `sha384`, `sha512`. Any other name panics at schema construction.

```go
md5 := z.String().Hash("md5")
md5.MustParse("5d41402abc4b2a76b9719d911017c592")

sha256 := z.String().Hash("sha256")
sha256.MustParse("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")

_ = md5.SafeParse("not-hex!") // fail
```

## UUID / GUID

```go
uuid := z.String().UUID()
uuid.MustParse("9491d710-3185-4e06-bea0-6a2f275345e0")
uuid.MustParse("00000000-0000-0000-0000-000000000000")

_ = uuid.SafeParse("invalid uuid") // fail
_ = uuid.SafeParse("9491d710-3185-0e06-bea0-6a2f275345e0") // version 0

v4 := z.String().UUIDv4()
v4.MustParse("9491d710-3185-4e06-bea0-6a2f275345e0")
_ = v4.SafeParse("9491d710-3185-1e06-bea0-6a2f275345e0") // v1 → fail

// GUID is looser (e.g. variant nibble 1 is accepted)
guid := z.String().GUID()
guid.MustParse("b3ce60f8-e8b9-40f5-1150-172ede56ff74")

res := uuid.SafeParse("purr")
// Message: "Invalid UUID"
```

## ID formats (NanoID, CUID, ULID, …)

```go
z.String().NanoID().MustParse("lfNZluvAxMkf7Q8C5H-QS")
z.String().CUID().MustParse("ckopqwoedu0013g5hseu82ta1")
z.String().CUID2().MustParse("tz4a98xxat96iws9zmbrgj3a")
// CUID2 rejects uppercase:
_ = z.String().CUID2().SafeParse("tz4a98xxat96iws9zMbrgj3a")

z.String().ULID().MustParse("01ARZ3NDEKTSV4RRFFQ69G5FAV")
z.String().KSUID().MustParse("2GcR3dN8zH1KpLqWvYxTjBfMsEa")
z.String().XID().MustParse("9m4e2mr0ui3e8a215n4g")

res := z.String().CUID().SafeParse("bad")
// Message: "Invalid cuid"
```

## Base64, Base64URL, Hex

```go
b64 := z.String().Base64()
b64.MustParse("SGVsbG8gV29ybGQ=")
b64.MustParse("") // empty is valid
res := b64.SafeParse("@@@")
// Message: "Invalid base64-encoded string"

z.String().Base64URL().MustParse("SGVsbG8")
// Padding `=` is typically rejected by the URL-safe validator

hex := z.String().Hex()
hex.MustParse("DEADBEEF")
hex.MustParse("0123456789abcdefABCDEF")
hex.MustParse("")
_ = hex.SafeParse("xyz")
_ = hex.SafeParse("123-abc")
```

## JWT

Structural JWT check (three segments). Optionally constrain the header `alg`:

```go
jwt := z.String().JWT()
// Accepts well-formed header.payload.signature (base64url)

algHS256 := z.String().JWT(z.JWTOpts{Alg: "HS256"})
// Rejects tokens whose header alg ≠ HS256
```

## Phone, emoji, network

```go
z.String().E164().MustParse("+14155552671")

z.String().IPv4().MustParse("192.168.1.1")
z.String().IPv6().MustParse("2001:db8::1")

z.String().CIDRv4().MustParse("10.0.0.0/8")
z.String().CIDRv6().MustParse("2001:db8::/32")

// Default delimiter ":"
z.String().MAC().MustParse("01:23:45:67:89:ab")
z.String().MAC(z.MACOpts{Delimiter: "-"}).MustParse("01-23-45-67-89-ab")
```

## ISO date / time / datetime / duration

```go
z.String().ISODate().MustParse("2024-01-15")
z.String().ISOTime().MustParse("14:30:00")

// Precision / offset / local via opts
prec := 3
z.String().ISOTime(z.ISOTimeOpts{Precision: &prec})

z.String().ISODateTime().MustParse("2024-01-15T14:30:00Z")
z.String().ISODateTime(z.ISODateTimeOpts{
    Offset: true, // allow ±hh:mm offsets
    Local:  true, // allow timezone-less local datetimes
})

z.String().ISODuration().MustParse("P1DT2H")
res := z.String().ISODuration().SafeParse("not-a-duration")
// Format == "duration"
```

:::warn Time vs `z.Time()`
`ISODate` / `ISODateTime` validate **strings**. For Go `time.Time` values, use [`z.Time()`](/api/time).
:::

## Custom formats with `z.StringFormat`

When the built-ins don’t cover you, define your own named format with `z.StringFormat(name, matcher, params...)`. The matcher is either a `*regexp.Regexp` or a `func(string) bool` (anything else panics). Failures produce a regular `invalid_format` issue with `Issue.Format` set to your name, so custom formats flow through locales and error maps like the built-ins.

```go
import "regexp"

tag := z.StringFormat("tag", regexp.MustCompile(`^[a-z][a-z0-9-]{2,31}$`))
tag.MustParse("billing-team")
_ = tag.SafeParse("NOPE") // fail, Format == "tag"

evenLen := z.StringFormat("even-len", func(v string) bool { return len(v)%2 == 0 })
evenLen.MustParse("abcd")
_ = evenLen.SafeParse("abc") // fail
```

## Custom messages & abort

```go
import "regexp"

s := z.String().Email("bad email")
res := s.SafeParse("x")
// Message: "bad email"

// Abort prevents subsequent format/regex issues from stacking
s = z.String().
    Email(z.Params{Abort: true, Error: z.MessageFromString("bad email")}).
    Regex(regexp.MustCompile(`^x$`))
res = s.SafeParse("x")
// Single issue: "bad email"
```

## Inspecting the issue

```go
res := z.String().Email().SafeParse("nope")
iss := res.Error.Issues[0]

// iss.Code    == "invalid_format"
// iss.Format  == "email"
// iss.Origin  == "string" for most formats; URL omits Origin
// iss.Message == "Invalid email address"
```
