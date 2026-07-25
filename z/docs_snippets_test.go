package z

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every ```go block in the README and the docs site must be syntactically valid
// Go. This does not type-check the samples — docs_claims_test.go compiles and
// runs the load-bearing ones — but it does catch samples that could never
// compile, which is what a reader hits first when they copy one.
func TestDocSnippetsParse(t *testing.T) {
	for _, s := range collectSnippets(t) {
		if err := parseSnippet(s.body); err != nil {
			t.Errorf("%s:%d: snippet is not valid Go: %v", s.file, s.line, err)
			continue
		}
		for _, problem := range danglingSelectors(s.body) {
			t.Errorf("%s:%d: %s", s.file, s.line, problem)
		}
	}
}

// collectSnippets returns every fenced Go block in the README and the docs tree.
func collectSnippets(t *testing.T) []docSnippet {
	t.Helper()
	fence := regexp.MustCompile("(?sm)^```go[^\n]*\n(.*?)^```")
	files := docFiles(t)
	var snippets []docSnippet
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		for _, loc := range fence.FindAllStringSubmatchIndex(text, -1) {
			snippets = append(snippets, docSnippet{
				file: file,
				line: 1 + strings.Count(text[:loc[0]], "\n"),
				body: text[loc[2]:loc[3]],
			})
		}
	}
	return snippets
}

// docFiles is the README plus every markdown file in the docs tree.
func docFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, target := range []string{"../README.md", "../docs/content"} {
		info, err := os.Stat(target)
		if err != nil {
			t.Skipf("docs not present: %v", err)
		}
		if !info.IsDir() {
			files = append(files, target)
			continue
		}
		err = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
				files = append(files, path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(files) < 10 {
		t.Fatalf("expected to find the docs tree, got %d files", len(files))
	}
	return files
}

// parseSnippet accepts a whole file, a list of declarations, or a list of
// statements, since docs samples are written in all three shapes.
func parseSnippet(body string) error {
	if strings.HasPrefix(strings.TrimSpace(body), "package ") {
		return parseGo(body)
	}
	imports, rest := splitImports(body)
	stmtErr := parseGo("package p\n" + imports + "func _() {\n" + rest + "\n}")
	if stmtErr == nil {
		return nil
	}
	declErr := parseGo("package p\n" + imports + rest)
	if declErr == nil {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(rest), "func ") ||
		strings.HasPrefix(strings.TrimSpace(rest), "type ") ||
		strings.Contains(rest, "\nfunc ") {
		return declErr
	}
	return stmtErr
}

// splitImports keeps a leading import block at file scope so the rest of the
// snippet can still be wrapped in a function body.
func splitImports(body string) (imports, rest string) {
	if !strings.HasPrefix(strings.TrimSpace(body), "import ") {
		return "", body
	}
	if i := strings.Index(body, "\n)\n"); i >= 0 && strings.Contains(body[:i], "(") {
		return body[:i+3], body[i+3:]
	}
	if i := strings.Index(body, "\n"); i >= 0 {
		return body[:i+1], body[i+1:]
	}
	return "", body
}

func parseGo(src string) error {
	_, err := parser.ParseFile(token.NewFileSet(), "snippet.go", src, parser.SkipObjectResolution)
	return err
}

// danglingSelectors reports statements that are only a selector — `wg.Wait`,
// `z.String().Min(5).Email` — and selector chains that read a field off a
// function value, like `time.Now.UTC()`. Neither is valid Go, but both parse, so
// the syntax check above cannot see them. This is the exact shape left behind
// when a text pass strips call parentheses.
func danglingSelectors(body string) []string {
	imports, rest := splitImports(body)
	file, err := parser.ParseFile(token.NewFileSet(), "snippet.go",
		"package p\n"+imports+"func _() {\n"+rest+"\n}", parser.SkipObjectResolution)
	if err != nil {
		// Declaration-shaped snippets are checked as a file instead.
		file, err = parser.ParseFile(token.NewFileSet(), "snippet.go", "package p\n"+imports+rest, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
	}

	var problems []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ExprStmt:
			if sel, ok := node.X.(*ast.SelectorExpr); ok {
				problems = append(problems, fmt.Sprintf(
					"%s is a selector used as a statement, which does not compile — a call is missing",
					exprText(sel)))
			}
		case *ast.SelectorExpr:
			// Reading a member off a package-level function, at any depth:
			// time.Now.UTC, time.Now.UTC.Format(…), context.Background.Done.
			if inner, ok := node.X.(*ast.SelectorExpr); ok {
				if pkg, ok := inner.X.(*ast.Ident); ok && looksLikePackage(pkg.Name) && isCallableName(inner.Sel.Name) {
					problems = append(problems, fmt.Sprintf(
						"%s reads a member of the function %s.%s — a call is missing after %s",
						exprText(node), pkg.Name, inner.Sel.Name, inner.Sel.Name))
				}
				// … or off a zero-argument method: v.(time.Time).UTC.Format(…).
				if isZeroArgMethod(inner.Sel.Name) {
					problems = append(problems, fmt.Sprintf(
						"%s reads a member of the method value %s — a call is missing after %s",
						exprText(node), exprText(inner), inner.Sel.Name))
				}
			}
		case *ast.AssignStmt:
			for _, rhs := range node.Rhs {
				if p := bareFunctionValue(rhs); p != "" {
					problems = append(problems, p)
				}
			}
		case *ast.CallExpr:
			for _, arg := range node.Args {
				if p := bareFunctionValue(arg); p != "" {
					problems = append(problems, p)
				}
			}
		}
		return true
	})
	return problems
}

// looksLikePackage reports whether name is one of the standard packages the docs
// use, so a chain rooted at it can be judged without type information.
func looksLikePackage(name string) bool {
	switch name {
	case "time", "context", "json", "fmt", "os", "http", "strings", "regexp", "errors", "sort", "math", "rand":
		return true
	default:
		return false
	}
}

// isCallableName reports whether a standard-library selector is a function rather
// than a type, constant or variable — the docs reference plenty of those
// (time.RFC3339, http.StatusOK, os.Stdout) and they are not missing a call.
func isCallableName(name string) bool {
	switch name {
	// Deliberately excludes functions the docs legitimately pass as values, such
	// as strings.ToUpper in OverwriteOf(schema, strings.ToUpper).
	case "Now", "Background", "TODO", "Marshal", "Unmarshal", "Getenv", "Sprintf",
		"Errorf", "MustCompile", "NaN", "Inf":
		return true
	default:
		return false
	}
}

// bareFunctionValue reports a package function used as a value — `context.Background`
// where `context.Background()` was meant. Passing a function as a value is legal
// Go, so this is judged by name; the list only holds functions the docs call.
func bareFunctionValue(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || !looksLikePackage(pkg.Name) || !isCallableName(sel.Sel.Name) {
		return ""
	}
	return fmt.Sprintf("%s.%s is used as a value — a call is missing", pkg.Name, sel.Sel.Name)
}

// isZeroArgMethod lists zero-argument methods of standard library types the docs
// chain from. "Error" is deliberately absent: res.Error is a field on
// SafeParseResult and appears throughout the docs.
func isZeroArgMethod(name string) bool {
	switch name {
	case "UTC", "Local", "Unix", "UnixMilli", "UnixNano":
		return true
	default:
		return false
	}
}

func exprText(e ast.Expr) string {
	var b strings.Builder
	if err := printer.Fprint(&b, token.NewFileSet(), e); err != nil {
		return "<expr>"
	}
	return b.String()
}

// Shell blocks were corrupted by the same pass that stripped call parentheses —
// "go test -race./..." and "go run." — and no Go-level check can see them, since
// they are not Go. This looks for a go subcommand whose argument lost its space.
func TestDocShellBlocksAreRunnable(t *testing.T) {
	fence := regexp.MustCompile("(?sm)^```(?:bash|sh|shell|console)[^\n]*\n(.*?)^```")
	// "go build./..." or "gofmt -l." — a flag or subcommand fused to its argument.
	fused := regexp.MustCompile(`\b(go|gofmt|golangci-lint)\b[^\n]*?[a-z0-9](\./|\.$)`)

	for _, file := range docFiles(t) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		for _, loc := range fence.FindAllStringSubmatchIndex(text, -1) {
			body := text[loc[2]:loc[3]]
			line := 1 + strings.Count(text[:loc[0]], "\n")
			for _, command := range strings.Split(body, "\n") {
				command = strings.TrimSpace(command)
				if command == "" || strings.HasPrefix(command, "#") {
					continue
				}
				if fused.MatchString(command) {
					t.Errorf("%s:%d: %q is missing a space before its argument", file, line, command)
				}
			}
		}
	}
}
