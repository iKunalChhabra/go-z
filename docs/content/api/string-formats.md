# String formats

Format helpers attach Zod-compatible `invalid_format` checks to a [`String`](/api/string) schema. Call them as fluent methods on `zod.String()`.

```go
email := zod.String().Email()
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
| `Email()` | `email` | `a@b.cd` | `Invalid email address` |
| `URL()` | `url` | `https://example.com` | `Invalid URL` |
| `UUID()` | `uuid` | any RFC UUID (v1–v8-ish) | `Invalid UUID` |
| `UUIDv4()` | `uuid` | version **4** only | `Invalid UUID` |
| `UUIDv6()` | `uuid` | version **6** only | `Invalid UUID` |
| `UUIDv7()` | `uuid` | version **7** only | `Invalid UUID` |
| `GUID()` | `guid` | looser GUID (variant 0 OK) | (locale default) |
| `NanoID()` | `nanoid` | `lfNZluvAxMkf7Q8C5H-QS` | (locale default) |
| `CUID()` | `cuid` | `ckopqwoedu0013g5hseu82ta1` | `Invalid cuid` |
| `CUID2()` | `cuid2` | lowercase cuid2 | (locale default) |
| `ULID()` | `ulid` | `01ARZ3NDEKTSV4RRFFQ69G5FAV` | `Invalid ULID` |
| `KSUID()` | `ksuid` | 27-char KSUID | `Invalid KSUID` |
| `XID()` | `xid` | `9m4e2mr0ui3e8a215n4g` | `Invalid XID` |
| `Base64()` | `base64` | `SGVsbG8gV29ybGQ=` | `Invalid base64-encoded string` |
| `Base64URL()` | `base64url` | URL-safe alphabet | (locale default) |
| `Hex()` | `hex` | `DEADBEEF`, `""` | (locale default) |
| `JWT()` | `jwt` | three base64url segments | (locale default) |
| `E164()` | `e164` | `+14155552671` | (locale default) |
| `Emoji()` | `emoji` | emoji-only string | (locale default) |
| `IPv4()` | `ipv4` | `192.168.0.1` | (locale default) |
| `IPv6()` | `ipv6` | `::1`, compressed forms | (locale default) |
| `CIDRv4()` | `cidrv4` | `10.0.0.0/8` | (locale default) |
| `CIDRv6()` | `cidrv6` | IPv6 prefix | (locale default) |
| `MAC()` | `mac` | `01:23:45:67:89:ab` | (locale default) |
| `ISODate()` | `date` | `2024-01-15` | (locale default) |
| `ISOTime()` | `time` | `14:30:00` | (locale default) |
| `ISODateTime()` | `datetime` | `2024-01-15T14:30:00Z` | (locale default) |
| `ISODuration()` | `duration` | `P1DT2H` | (locale default) |

:::info UUID methods share format `"uuid"`
`UUIDv4`, `UUIDv6`, and `UUIDv7` tighten the pattern but still report `Format: "uuid"` — matching Zod v4.
:::

## Email

```go
email := zod.String().Email()

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

`URL()` trims surrounding whitespace. Pass `zod.URLOpts{Normalize: true}` to rewrite via Go’s URL parser (href-style normalization).

```go
u := zod.String().URL()
u.MustParse("https://google.com/asdf?q=1#hash")
u.MustParse("http://localhost")

got := u.MustParse("  https://example.com  ")
// got == "https://example.com"

res := u.SafeParse("asdf")
// Message: "Invalid URL"
// Note: URL issues omit Origin (matches Zod $ZodURL)

norm := zod.String().URL(zod.URLOpts{Normalize: true})
_ = norm.MustParse("https://example.com?key=value")
```

## UUID / GUID

```go
uuid := zod.String().UUID()
uuid.MustParse("9491d710-3185-4e06-bea0-6a2f275345e0")
uuid.MustParse("00000000-0000-0000-0000-000000000000")

_ = uuid.SafeParse("invalid uuid") // fail
_ = uuid.SafeParse("9491d710-3185-0e06-bea0-6a2f275345e0") // version 0

v4 := zod.String().UUIDv4()
v4.MustParse("9491d710-3185-4e06-bea0-6a2f275345e0")
_ = v4.SafeParse("9491d710-3185-1e06-bea0-6a2f275345e0") // v1 → fail

// GUID is looser (e.g. variant nibble 1 is accepted)
guid := zod.String().GUID()
guid.MustParse("b3ce60f8-e8b9-40f5-1150-172ede56ff74")

res := uuid.SafeParse("purr")
// Message: "Invalid UUID"
```

## ID formats (NanoID, CUID, ULID, …)

```go
zod.String().NanoID().MustParse("lfNZluvAxMkf7Q8C5H-QS")
zod.String().CUID().MustParse("ckopqwoedu0013g5hseu82ta1")
zod.String().CUID2().MustParse("tz4a98xxat96iws9zmbrgj3a")
// CUID2 rejects uppercase:
_ = zod.String().CUID2().SafeParse("tz4a98xxat96iws9zMbrgj3a")

zod.String().ULID().MustParse("01ARZ3NDEKTSV4RRFFQ69G5FAV")
zod.String().KSUID().MustParse("2GcR3dN8zH1KpLqWvYxTjBfMsEa")
zod.String().XID().MustParse("9m4e2mr0ui3e8a215n4g")

res := zod.String().CUID().SafeParse("bad")
// Message: "Invalid cuid"
```

## Base64, Base64URL, Hex

```go
b64 := zod.String().Base64()
b64.MustParse("SGVsbG8gV29ybGQ=")
b64.MustParse("") // empty is valid
res := b64.SafeParse("@@@")
// Message: "Invalid base64-encoded string"

zod.String().Base64URL().MustParse("SGVsbG8")
// Padding `=` is typically rejected by the URL-safe validator

hex := zod.String().Hex()
hex.MustParse("DEADBEEF")
hex.MustParse("0123456789abcdefABCDEF")
hex.MustParse("")
_ = hex.SafeParse("xyz")
_ = hex.SafeParse("123-abc")
```

## JWT

Structural JWT check (three segments). Optionally constrain the header `alg`:

```go
jwt := zod.String().JWT()
// Accepts well-formed header.payload.signature (base64url)

algHS256 := zod.String().JWT(zod.JWTOpts{Alg: "HS256"})
// Rejects tokens whose header alg ≠ HS256
```

## Phone, emoji, network

```go
zod.String().E164().MustParse("+14155552671")

zod.String().IPv4().MustParse("192.168.1.1")
zod.String().IPv6().MustParse("2001:db8::1")

zod.String().CIDRv4().MustParse("10.0.0.0/8")
zod.String().CIDRv6().MustParse("2001:db8::/32")

// Default delimiter ":"
zod.String().MAC().MustParse("01:23:45:67:89:ab")
zod.String().MAC(zod.MACOpts{Delimiter: "-"}).MustParse("01-23-45-67-89-ab")
```

## ISO date / time / datetime / duration

```go
zod.String().ISODate().MustParse("2024-01-15")
zod.String().ISOTime().MustParse("14:30:00")

// Precision / offset / local via opts
prec := 3
zod.String().ISOTime(zod.ISOTimeOpts{Precision: &prec})

zod.String().ISODateTime().MustParse("2024-01-15T14:30:00Z")
zod.String().ISODateTime(zod.ISODateTimeOpts{
    Offset: true, // allow ±hh:mm offsets
    Local:  true, // allow timezone-less local datetimes
})

zod.String().ISODuration().MustParse("P1DT2H")
res := zod.String().ISODuration().SafeParse("not-a-duration")
// Format == "duration"
```

:::warn Time vs `zod.Time()`
`ISODate` / `ISODateTime` validate **strings**. For Go `time.Time` values, use [`zod.Time()`](/api/time).
:::

## Custom messages & abort

```go
import "regexp"

s := zod.String().Email("bad email")
res := s.SafeParse("x")
// Message: "bad email"

// Abort prevents subsequent format/regex issues from stacking
s = zod.String().
    Email(zod.Params{Abort: true, Error: zod.MessageFromString("bad email")}).
    Regex(regexp.MustCompile(`^x$`))
res = s.SafeParse("x")
// Single issue: "bad email"
```

## Inspecting the issue

```go
res := zod.String().Email().SafeParse("nope")
iss := res.Error.Issues[0]

iss.Code    // "invalid_format"
iss.Format  // "email"
iss.Origin  // "string" for most formats; URL omits Origin
iss.Message // "Invalid email address"
```
