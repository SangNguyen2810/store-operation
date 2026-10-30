// Package archtest keeps the redis service independent: it may use the
// standard library, third-party packages and its own packages, and never the
// code of another service. It talks to the others over the network only.
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

func TestDoesNotImportAnotherService(t *testing.T) {
	const own = "store-operation/app/redis"
	fset := token.NewFileSet()
	checked := 0
	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if strings.Contains(p, "store-operation/") && !strings.Contains(p, own) {
				t.Errorf("%s imports %s: services talk over the network, not by importing each other", path, p)
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
