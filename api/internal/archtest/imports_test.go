// Package archtest enforces the layering rules by reading imports. It is a
// test-only package; `go test ./...` (and so CI) fails when a rule is broken.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Layers, innermost first. Dependencies point inward only.
//
//	domain  -> standard library only
//	app     -> domain
//	adapters-> domain, app, platform, third-party (never another adapter)
//	platform-> no context
//	a context -> its own packages and platform, never another context
//	cmd     -> anything (composition root)

func modulePath(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimSpace(strings.TrimPrefix(line, "module"))
}

func isStdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".")
}

// violation returns why a file in package dir (relative to internal/, slash
// separated) may not import imp, or "" if it may.
func violation(mod, dir, imp string) string {
	internal := mod + "/internal/"
	if !strings.HasPrefix(imp, internal) {
		if strings.HasPrefix(imp, mod) {
			return "imports outside internal/"
		}
		if !isStdlib(imp) && (isLayer(dir, "domain") || isLayer(dir, "app")) {
			return "domain and app may use only the standard library (and each other)"
		}
		return ""
	}
	target := strings.TrimPrefix(imp, internal)
	ctx, _, _ := strings.Cut(dir, "/")

	switch {
	case ctx == "platform":
		if !strings.HasPrefix(target, "platform/") {
			return "platform must not depend on a context"
		}
	case !strings.HasPrefix(target, ctx+"/") && !strings.HasPrefix(target, "platform/"):
		return "a context may import only itself and platform, not " + strings.SplitN(target, "/", 2)[0]
	}
	switch {
	case isLayer(dir, "domain") && !strings.HasPrefix(target, ctx+"/domain"):
		return "domain must not import other layers or platform"
	case isLayer(dir, "app") && !strings.HasPrefix(target, ctx+"/domain"):
		return "app may import only domain"
	case strings.Contains(dir, "/adapters/") && strings.Contains(target, "/adapters/") && dir != target:
		return "an adapter must not import another adapter"
	}
	return ""
}

func isLayer(dir, layer string) bool {
	parts := strings.Split(dir, "/")
	return len(parts) >= 2 && parts[1] == layer
}

func TestImportRules(t *testing.T) {
	mod := modulePath(t)
	root := ".." // internal/
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		dir := filepath.ToSlash(rel)
		if dir == "archtest" {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, s := range f.Imports {
			imp, _ := strconv.Unquote(s.Path.Value)
			if why := violation(mod, dir, imp); why != "" {
				t.Errorf("%s imports %s: %s", filepath.ToSlash(path), imp, why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The rules must actually bite: each case below is a violation the walker should catch.
func TestViolationsAreDetected(t *testing.T) {
	const mod = "example.com/m"
	bad := []struct{ dir, imp string }{
		{"analytics/domain", "github.com/jackc/pgx/v5"},
		{"analytics/domain", mod + "/internal/analytics/app"},
		{"analytics/domain", mod + "/internal/platform/db"},
		{"analytics/app", "github.com/jackc/pgx/v5"},
		{"analytics/app", mod + "/internal/analytics/adapters/postgres"},
		{"analytics/app", mod + "/internal/platform/db"},
		{"analytics/adapters/http", mod + "/internal/analytics/adapters/postgres"},
		{"analytics/adapters/cache", mod + "/internal/analytics/adapters/postgres"},
		{"analytics/adapters/postgres", mod + "/internal/othercontext/domain"},
		{"platform/httpx", mod + "/internal/analytics/domain"},
		{"platform/cache", mod + "/internal/analytics/domain"},
		{"analytics/app", mod + "/cmd/api"},
	}
	for _, c := range bad {
		if violation(mod, c.dir, c.imp) == "" {
			t.Errorf("%s importing %s should be rejected", c.dir, c.imp)
		}
	}
	ok := []struct{ dir, imp string }{
		{"analytics/domain", "time"},
		{"analytics/app", "context"},
		{"analytics/app", mod + "/internal/analytics/domain"},
		{"analytics/adapters/postgres", "github.com/jackc/pgx/v5"},
		{"analytics/adapters/postgres", mod + "/internal/analytics/domain"},
		{"analytics/adapters/http", mod + "/internal/analytics/app"},
		{"analytics/adapters/http", mod + "/internal/platform/httpx"},
		{"analytics/adapters/cache", mod + "/internal/analytics/app"},
		{"analytics/adapters/cache", mod + "/internal/platform/cache"},
		{"platform/cache", "github.com/redis/go-redis/v9"},
		{"platform/httpx", "net/http"},
		{"platform/rdb", "github.com/redis/go-redis/v9"},
	}
	for _, c := range ok {
		if why := violation(mod, c.dir, c.imp); why != "" {
			t.Errorf("%s importing %s should be allowed, got: %s", c.dir, c.imp, why)
		}
	}
}
