// Package archtest keeps the shared library free of any service: it may use
// the standard library, third-party packages and its own packages, but never
// code from app/api or app/gateway, so every service can depend on it.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDoesNotDependOnAService(t *testing.T) {
	fset := token.NewFileSet()
	checked := 0
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if strings.Contains(p, "store-operation/app/") {
				t.Errorf("%s imports %s: shared code must not depend on a service", path, p)
			}
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no Go files were checked")
	}
}
