package z

import (
	"go/parser"
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
	fence := regexp.MustCompile("(?sm)^```go[^\n]*\n(.*?)^```")

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

	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		for _, loc := range fence.FindAllStringSubmatchIndex(text, -1) {
			body := text[loc[2]:loc[3]]
			line := 1 + strings.Count(text[:loc[0]], "\n")
			if err := parseSnippet(body); err != nil {
				t.Errorf("%s:%d: snippet is not valid Go: %v", file, line, err)
			}
		}
	}
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
