# Installation

Add go-zod to your module with the standard Go toolchain. No code generation, no CGO, no runtime config files.

## Requirements

| Requirement | Details |
|---|---|
| **Go** | **1.22 or newer** |
| **Module path** | `github.com/iKunalChhabra/go-zod` |
| **Gin (optional)** | Only if you import `github.com/iKunalChhabra/go-zod/zgin` |

:::tip Go modules only
go-zod is a normal Go module. There is no `zod init` CLI and no schema compiler step. You write schemas in Go and call `Parse`.
:::

## Install the core package

From your project root (where `go.mod` lives):

```bash
go get github.com/iKunalChhabra/go-zod
```

That adds a `require` line similar to:

```go
require github.com/iKunalChhabra/go-zod v0.x.x
```

Import it with the `z` alias (Zod convention — used throughout these docs):

```go
import z "github.com/iKunalChhabra/go-zod"
```

:::tip Docs import style
Examples in this documentation use `import z "github.com/iKunalChhabra/go-zod"` so the API reads like Zod’s `z` (`z.String()`, `z.Object`, …). The Go package name is still `zod` if you import without an alias.
:::

## Optional: Gin integration

The `zgin` subpackage is **not** pulled in unless you import it. Gin is an optional dependency of that subpackage only.

```bash
go get github.com/gin-gonic/gin
go get github.com/iKunalChhabra/go-zod
```

```go
import (
	"github.com/gin-gonic/gin"
	z "github.com/iKunalChhabra/go-zod"
	"github.com/iKunalChhabra/go-zod/zgin"
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

	z "github.com/iKunalChhabra/go-zod"
)

func main() {
	schema := z.String().Min(5).Email()

	out, err := schema.Parse("ada@example.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, z.Prettify(err.(*z.ZodError)))
		os.Exit(1)
	}
	fmt.Println("ok:", out)
}
```

Initialize a module (if you don’t have one yet) and run:

```bash
go mod init example.com/zod-smoke
go get github.com/iKunalChhabra/go-zod
go run .
```

Expected output:

```text
ok: ada@example.com
```

Try a failing input to confirm errors work:

```go
_, err := schema.Parse("nope")
fmt.Println(err.(*z.ZodError).Issues[0].Code) // invalid_format
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

import z "github.com/iKunalChhabra/go-zod"

var CreateUser = z.Object(z.Shape{
	"name":  z.String().Min(2),
	"email": z.String().Email(),
})
```

## Version pinning

Pin a specific version in CI the usual way:

```bash
go get github.com/iKunalChhabra/go-zod@v0.1.0
```

Or edit `go.mod` and run `go mod tidy`.

:::info Go 1.22 language features
go-zod targets Go 1.22+ so it can use modern generics and standard-library APIs. Older toolchains will refuse the module’s `go` directive — upgrade the toolchain rather than forcing an older Go version.
:::

## Next steps

- [Quickstart](#/guide/quickstart) — define an object schema and handle `ZodError`
- [Why go-zod?](#/guide/why) — what was ported from Zod v4 and why
- [Gin (zgin)](#/integrations/gin) — middleware and bind helpers when you’re ready
