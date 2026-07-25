package z

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Parsing a snippet only proves it is shaped like Go. Type-checking proves the
// API it names exists: that `schema.Shape` needs a call, that
// `zgin.ValidateQuery` was never a function, that `Options` is a field and not a
// method. This test compiles every documentation snippet against the real
// packages and reports the errors that are not an artifact of quoting a fragment.
//
// Fragments legitimately reference identifiers defined in an earlier block, so
// "undefined: base" is ignored. Anything that names a member of a known type is
// not ignored, which is the class this exists to catch.
func TestDocSnippetsTypeCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles ~200 packages; skipped under -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain available")
	}

	snippets := collectSnippets(t)
	if len(snippets) < 50 {
		t.Fatalf("expected the docs corpus, found %d snippets", len(snippets))
	}

	dir := t.TempDir()
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), `module docsnippets

go 1.26

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/iKunalChhabra/go-z v0.0.0
	github.com/iKunalChhabra/go-z/zgin v0.0.0
)

replace github.com/iKunalChhabra/go-z => `+repo+`

replace github.com/iKunalChhabra/go-z/zgin => `+filepath.Join(repo, "zgin")+`
`)
	// Reuse the integration module's checksums so Gin resolves from the module
	// cache instead of the network.
	if sum, err := os.ReadFile(filepath.Join(repo, "zgin", "go.sum")); err == nil {
		writeFile(t, filepath.Join(dir, "go.sum"), string(sum))
	}

	// One package per snippet: docs reuse names like `User` and `schema`, which
	// would collide inside a single package.
	for i, s := range snippets {
		pkg := "s" + strconv.Itoa(i)
		src, wrapped, ok := snippetSource(pkg, s.body)
		if !ok {
			continue
		}
		snippets[i].wrappedInFunc = wrapped
		if err := os.MkdirAll(filepath.Join(dir, pkg), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, pkg, "snippet.go"), src)
	}

	// -e removes the "too many errors" cap, so one bad snippet cannot hide others.
	cmd := exec.Command("go", "build", "-gcflags=-e", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	report := string(out)
	if strings.Contains(report, "dial tcp") || strings.Contains(report, "no such host") ||
		strings.Contains(report, "missing go.sum entry") {
		t.Skipf("dependencies unavailable offline:\n%s", report)
	}

	for _, problem := range interestingTypeErrors(report, snippets) {
		t.Error(problem)
	}
}

// defaultImports are the packages documentation snippets reference without
// necessarily importing them, with a symbol to anchor each one.
var defaultImports = []struct{ name, path, anchor string }{
	{"z", "github.com/iKunalChhabra/go-z/z", "String"},
	{"zgin", "github.com/iKunalChhabra/go-z/zgin", "ContextKey"},
	{"fmt", "fmt", "Sprint"},
	{"json", "encoding/json", "Marshal"},
	{"time", "time", "Now"},
	{"strings", "strings", "TrimSpace"},
	{"regexp", "regexp", "MustCompile"},
	{"errors", "errors", "New"},
	{"os", "os", "Stdout"},
	{"context", "context", "Background"},
	{"http", "net/http", "StatusOK"},
	{"sync", "sync", "OnceFunc"},
	{"math", "math", "MaxInt64"},
	{"big", "math/big", "NewInt"},
	{"gin", "github.com/gin-gonic/gin", "New"},
	{"sort", "sort", "Strings"},
	{"strconv", "strconv", "Itoa"},
}

// importedNames returns the identifiers an import block binds.
func importedNames(imports string) map[string]bool {
	bound := map[string]bool{}
	file, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package p\n"+imports, parser.SkipObjectResolution)
	if err != nil {
		return bound
	}
	for _, spec := range file.Imports {
		if spec.Name != nil {
			bound[spec.Name.Name] = true
			continue
		}
		path := strings.Trim(spec.Path.Value, `"`)
		if i := strings.LastIndex(path, "/"); i >= 0 {
			path = path[i+1:]
		}
		bound[path] = true
	}
	return bound
}

type docSnippet struct {
	file string
	line int
	body string
	// wrappedInFunc records that the harness supplied the enclosing function, so
	// a return-count error is its artifact rather than a documentation problem.
	wrappedInFunc bool
}

// snippetSource wraps a snippet so it compiles as a package of its own, or
// reports false when the snippet is a reference listing that no package can
// contain: a bodiless signature, or a method declared on a type from elsewhere.
func snippetSource(pkg, body string) (src string, wrappedInFunc bool, ok bool) {
	body = stripPackageClause(body)
	imports, rest := hoistImports(body)
	if listingOnly(imports + rest) {
		return "", false, false
	}

	// Inject the packages the docs use under the names they use, but only those
	// the snippet has not bound itself: comparison.md imports another library as
	// `z`, and a duplicate import would be the only error the compiler reports.
	bound := importedNames(imports)
	var injected []string
	for _, imp := range defaultImports {
		if bound[imp.name] {
			continue
		}
		injected = append(injected, imp.name+" \""+imp.path+"\"")
	}

	var b strings.Builder
	b.WriteString("package " + pkg + "\n")
	b.WriteString(imports)
	if len(injected) > 0 {
		b.WriteString("\nimport (\n\t" + strings.Join(injected, "\n\t") + "\n)\n")
		// Anchor each injected import so "imported and not used" cannot be the
		// only error a snippet produces.
		b.WriteString("\nvar (\n")
		for _, imp := range defaultImports {
			if bound[imp.name] {
				continue
			}
			b.WriteString("\t_ = " + imp.name + "." + imp.anchor + "\n")
		}
		b.WriteString(")\n")
	}

	// Declarations belong at file scope; statements need a function around them.
	if parseGo("package p\n"+imports+rest) == nil && containsDeclaration(rest) {
		b.WriteString(rest)
		b.WriteString("\n")
		return b.String(), false, true
	}
	b.WriteString("func _snippet() {\n")
	b.WriteString(rest)
	b.WriteString("\n}\n")
	return b.String(), true, true
}

// hoistImports lifts every import declaration to the top, wherever the snippet
// put it. Docs interleave imports with the code they illustrate, which is fine
// prose and invalid Go once the rest is wrapped in a function.
func hoistImports(body string) (imports, rest string) {
	var importBlocks []string
	lines := strings.Split(body, "\n")
	var kept []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(trimmed, "import ("):
			block := []string{trimmed}
			for i+1 < len(lines) {
				i++
				block = append(block, lines[i])
				if strings.TrimSpace(lines[i]) == ")" {
					break
				}
			}
			importBlocks = append(importBlocks, strings.Join(block, "\n"))
		case strings.HasPrefix(trimmed, "import \""), strings.HasPrefix(trimmed, "import _ \""):
			importBlocks = append(importBlocks, trimmed)
		default:
			kept = append(kept, lines[i])
		}
	}
	if len(importBlocks) == 0 {
		return "", body
	}
	return strings.Join(importBlocks, "\n") + "\n", strings.Join(kept, "\n")
}

// stripPackageClause removes a leading package clause: the harness supplies its own.
func stripPackageClause(body string) string {
	trimmed := strings.TrimLeft(body, " \t\n")
	if !strings.HasPrefix(trimmed, "package ") {
		return body
	}
	if i := strings.Index(trimmed, "\n"); i >= 0 {
		return trimmed[i+1:]
	}
	return ""
}

// listingOnly reports whether a snippet is an API reference rather than code: a
// function without a body, or a method on a type the snippet does not declare.
func listingOnly(body string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "snippet.go", "package p\n"+body, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Body == nil || fn.Recv != nil {
			return true
		}
	}
	return false
}

func containsDeclaration(body string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "snippet.go", "package p\n"+body, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			return true
		case *ast.GenDecl:
			if d.Tok != token.IMPORT {
				return true
			}
		}
	}
	return false
}

var errorLine = regexp.MustCompile(`^(?:\./)?s(\d+)[/\\]snippet\.go:(\d+):\d+: (.*)$`)

// interestingTypeErrors keeps the errors that indicate wrong documentation and
// drops those that only reflect a snippet being an excerpt.
func interestingTypeErrors(report string, snippets []docSnippet) []string {
	var problems []string
	for _, line := range strings.Split(report, "\n") {
		m := errorLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		index, _ := strconv.Atoi(m[1])
		message := m[3]
		if index >= len(snippets) {
			continue
		}
		s := snippets[index]
		if ignorableTypeError(message) || (s.wrappedInFunc && isReturnCountError(message)) {
			continue
		}
		problems = append(problems, s.file+":"+strconv.Itoa(s.line)+": "+message)
	}
	return problems
}

func ignorableTypeError(message string) bool {
	// A fragment may use identifiers an earlier block defined. Package-qualified
	// names are not excused: "undefined: zgin.ValidateQuery" is a real error.
	if strings.HasPrefix(message, "undefined: ") && !strings.Contains(message, ".") {
		return true
	}
	for _, artifact := range []string{
		"declared and not used",
		"imported and not used",
		"no new variables on left side of :=",
		"is not used",
		"evaluated but not used",
		"missing return",
		"redeclared in this block", // a fragment repeating a declaration
	} {
		if strings.Contains(message, artifact) {
			return true
		}
	}
	return false
}

// isReturnCountError reports the errors produced by a fragment that returns a
// value while the harness's wrapper declares no results.
func isReturnCountError(message string) bool {
	return strings.Contains(message, "too many return values") ||
		strings.Contains(message, "not enough return values")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
