// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !go1.1

package test

// imported from $GOROOT/src/cmd/go/test.go

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/davecheney/gogo/project"
)

// isTest tells whether name looks like a test (or benchmark, according to prefix).
// It is a Test (say) if there is a character after Test that is not a lower-case letter.
// We don't want TesticularCancer.
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	rune, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(rune)
}

// writeTestmain writes the _testmain.go file for package p to
// the file named out.
func writeTestmain(out string, p *project.Package) error {
	t := &testFuncs{
		Package: p,
	}
	for _, file := range p.TestGoFiles {
		if err := t.load(filepath.Join(p.Srcdir, file), "_test", &t.NeedTest); err != nil {
			return err
		}
	}
	for _, file := range p.XTestGoFiles {
		if err := t.load(filepath.Join(p.Srcdir, file), "_xtest", &t.NeedXtest); err != nil {
			return err
		}
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := testmainTmpl.Execute(f, t); err != nil {
		return err
	}

	return nil
}

type testFuncs struct {
	Tests      []testFunc
	Benchmarks []testFunc
	Examples   []testFunc
	*gogo.Package
	NeedTest  bool
	NeedXtest bool
}

type testFunc struct {
	Package string // imported package name (_test or _xtest)
	Name    string // function name
	Output  string // output, for examples
}

var testFileSet = token.NewFileSet()

func (t *testFuncs) load(filename, pkg string, seen *bool) error {
	f, err := parser.ParseFile(testFileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, d := range f.Decls {
		n, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if n.Recv != nil {
			continue
		}
		name := n.Name.String()
		switch {
		case isTest(name, "Test"):
			t.Tests = append(t.Tests, testFunc{pkg, name, ""})
			*seen = true
		case isTest(name, "Benchmark"):
			t.Benchmarks = append(t.Benchmarks, testFunc{pkg, name, ""})
			*seen = true
		}
	}
	for _, e := range doc.Examples(f) {
		if e.Output == "" {
			// Don't run examples with no output.
			continue
		}
		t.Examples = append(t.Examples, testFunc{pkg, "Example" + e.Name, e.Output})
		*seen = true
	}
	return nil
}

var testmainTmpl = template.Must(template.New("main").Parse(`
package main

import (
	"regexp"
	"testing"

{{if .NeedTest}}
	_test {{.Package.ImportPath | printf "%q"}}
{{end}}
{{if .NeedXtest}}
	_xtest {{.Package.ImportPath | printf "%s_test" | printf "%q"}}
{{end}}
)

var tests = []testing.InternalTest{
{{range .Tests}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var benchmarks = []testing.InternalBenchmark{
{{range .Benchmarks}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var examples = []testing.InternalExample{
{{range .Examples}}
	{"{{.Name}}", {{.Package}}.{{.Name}}, {{.Output | printf "%q"}}},
{{end}}
}

var matchPat string
var matchRe *regexp.Regexp

func matchString(pat, str string) (result bool, err error) {
	if matchRe == nil || matchPat != pat {
		matchPat = pat
		matchRe, err = regexp.Compile(matchPat)
		if err != nil {
			return
		}
	}
	return matchRe.MatchString(str), nil
}

func main() {
	testing.Main(matchString, tests, benchmarks, examples)
}

`))
