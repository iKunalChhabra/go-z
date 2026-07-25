# Installation

Add go-z to your module with the standard Go toolchain. No code generation, no CGO, no runtime config files.

## Requirements

| Requirement | Details |
|---|---|
| **Go** | **1.26 or newer** |
| **Module path** | `github.com/iKunalChhabra/go-z` |
| **Import path** | `github.com/iKunalChhabra/go-z/z` |
| **Core dependency** | `golang.org/x/text` (Unicode NFC for `Normalize()`) — and nothing else |
| **Gin (optional)** | A separate module, `github.com/iKunalChhabra/go-z/zgin` |

:::tip Go modules only
go-z is a normal Go module. There is no `go-z init` CLI and no schema compiler step. You write schemas in Go and call `Parse`.
:::

## Install the core package

From your project root (where `go.mod` lives):

```bash
go get github.com/iKunalChhabra/go-z/z
```

That adds a `require` line similar to:

```text
require github.com/iKunalChhabra/go-z v0.2.0
```

Import it and use the `z.` prefix (as these docs do throughout):

```go
import "github.com/iKunalChhabra/go-z/z"
```

:::tip Docs import style
The package is named `z` and lives at `/z`, so a plain import gives you the `z.` prefix (`z.String()`, `z.Object`, …) with no alias to remember.
:::

## Optional: Gin integration

`zgin` is a **separate Go module** in the same repository, so Gin never enters
your dependency graph unless you ask for it. Installing the core gives you a
two-line `go.sum`; installing `zgin` brings in Gin and the core together:

```bash
go get github.com/iKunalChhabra/go-z/zgin
```

```go
import (
	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-z/z"
	"github.com/iKunalChhabra/go-z/zgin"
)
```

If your service is plain `net/http`, Chi, Echo, or Fiber, you only need the core package — call `schema.Parse` / `SafeParse` yourself and shape the HTTP response.

## Verify with a tiny program

Create `main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/iKunalChhabra/go-z/z"
)

func main() {
	schema := z.String().Min(5).Email()

	out, err := schema.Parse("ada@example.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, z.Prettify(err.(*z.Error)))
		os.Exit(1)
	}
	fmt.Println("ok:", out)
}
```

Initialize a module (if you don’t have one yet) and run:

```bash
go mod init example.com/goz-smoke
go get github.com/iKunalChhabra/go-z/z
go run .
```

Expected output:

```text
ok: ada@example.com
```

Try a failing input to confirm errors work:

```go
_, err := schema.Parse("nope")
fmt.Println(err.(*z.Error).Issues[0].Code) // invalid_format
```

## Module layout tips

Keep schemas next to the HTTP boundary or in a dedicated package:

```text
myapp/
  go.mod
  cmd/api/main.go
  internal/schemas/user.go   // z.Object(...) definitions
  internal/http/handlers.go  // Parse / zgin.Validate
```

Schemas are plain Go values. Share them as package-level variables — they are immutable and safe for concurrent use (see [Immutability & concurrency](#/guide/concurrency)).

```go
package schemas

import "github.com/iKunalChhabra/go-z/z"

var CreateUser = z.Object(z.Shape{
	"name":  z.String().Min(2),
	"email": z.String().Email(),
})
```

## Version pinning

Pin a specific version in CI the usual way:

```bash
go get github.com/iKunalChhabra/go-z/z@v0.2.0
```

Or edit `go.mod` and run `go mod tidy`.

:::info Go 1.26 language features
go-z targets Go 1.26+ so it can use range-over-function iterators, `sync.WaitGroup.Go`, `testing/synctest`, and `new(expr)`. Older toolchains will refuse the module’s `go` directive; with `GOTOOLCHAIN=auto` (the default) Go downloads a suitable toolchain for you.
:::

## Next steps

- [Quickstart](#/guide/quickstart) — define an object schema and handle `Error`
- [Why go-z?](#/guide/why) — what was ported from the upstream library and why
- [Gin (zgin)](#/integrations/gin) — middleware and bind helpers when you’re ready
