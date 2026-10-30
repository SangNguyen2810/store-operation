// Package archtest keeps the gateway independent: it may use the standard
// library and its own packages, nothing else, so it never depends on the API.
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

func TestOnlyStandardLibraryAndOwnPackages(t *testing.T) {
	b, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(string(b), "\n")
	module := strings.TrimSpace(strings.TrimPrefix(line, "module"))

	fset := token.NewFileSet()
	checked := 0
	err = filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(p, "/")
			if strings.Contains(first, ".") && p != module && !strings.HasPrefix(p, module+"/") {
				t.Errorf("%s imports %s: the gateway may only use the standard library", path, p)
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

func TestDoesNotImportOtherServicesOrShared(t *testing.T) {
	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "imports_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "store-operation/app/api") || strings.Contains(string(b), "store-operation/shared") {
			t.Errorf("%s refers to the api or shared module", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
