package project

import (
	"bytes"
	"fmt"
	"go/ast"
	"runtime"
	"strings"
	"unicode"
)

// Spec represents a build specification.
type Spec struct {
	goos, goarch, toolchain string
	buildTags, releaseTags  []string
	cgoEnabled              bool
}

// DefaultSpec returns a Spec that represents this machine.
func DefaultSpec() Spec {
	return Spec{goos: runtime.GOOS, goarch: runtime.GOARCH, cgoEnabled: true}
}

// from $GOROOT/src/pkg/go/build/build.go

// goodOSArchFile returns false if the name contains a $GOOS or $GOARCH
// suffix which does not match the current system.
// The recognized name formats are:
//
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
//
func (ctxt *Spec) goodOSArchFile(name string) bool {
	if dot := strings.Index(name, "."); dot != -1 {
		name = name[:dot]
	}
	l := strings.Split(name, "_")
	if n := len(l); n > 0 && l[n-1] == "test" {
		l = l[:n-1]
	}
	n := len(l)
	if n >= 2 && knownOS[l[n-2]] && knownArch[l[n-1]] {
		return l[n-2] == ctxt.goos && l[n-1] == ctxt.goarch
	}
	if n >= 1 && knownOS[l[n-1]] {
		return l[n-1] == ctxt.goos
	}
	if n >= 1 && knownArch[l[n-1]] {
		return l[n-1] == ctxt.goarch
	}
	return true
}

var knownOS = make(map[string]bool)
var knownArch = make(map[string]bool)

func init() {
	for _, v := range strings.Fields(goosList) {
		knownOS[v] = true
	}
	for _, v := range strings.Fields(goarchList) {
		knownArch[v] = true
	}
}

// match returns true if the name is one of:
//
//      $GOOS
//      $GOARCH
//      cgo (if cgo is enabled)
//      !cgo (if cgo is disabled)
//      ctxt.Compiler
//      !ctxt.Compiler
//      tag (if tag is listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      !tag (if tag is not listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      a comma-separated list of any of these
//
func (c *Spec) match(name string) bool {
	if name == "" {
		return false
	}
	if i := strings.Index(name, ","); i >= 0 {
		// comma-separated list
		return c.match(name[:i]) && c.match(name[i+1:])
	}
	if strings.HasPrefix(name, "!!") { // bad syntax, reject always
		return false
	}
	if strings.HasPrefix(name, "!") { // negation
		return len(name) > 1 && !c.match(name[1:])
	}

	// Tags must be letters, digits, underscores or dots.
	// Unlike in Go identifiers, all digits are fine (e.g., "386").
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
			return false
		}
	}

	// special tags
	if c.cgoEnabled && name == "cgo" {
		return true
	}
	if name == c.goos || name == c.goarch || name == c.toolchain {
		return true
	}

	// other tags
	for _, tag := range c.buildTags {
		if tag == name {
			return true
		}
	}
	for _, tag := range c.releaseTags {
		if tag == name {
			return true
		}
	}

	return false
}

var slashslash = []byte("//")

// shouldBuild reports whether it is okay to use this file,
// The rule is that in the file's leading run of // comments
// and blank lines, which must be followed by a blank line
// (to avoid including a Go package clause doc comment),
// lines beginning with '// +build' are taken as build directives.
//
// The file is accepted only if each such line lists something
// matching the file.  For example:
//
//      // +build windows linux
//
// marks the file as applicable only on Windows and Linux.
//
func (ctxt *Spec) shouldBuild(content []byte) bool {
	// Pass 1. Identify leading run of // comments and blank lines,
	// which must be followed by a blank line.
	end := 0
	p := content
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 { // Blank line
			end = len(content) - len(p)
			continue
		}
		if !bytes.HasPrefix(line, slashslash) { // Not comment line
			break
		}
	}
	content = content[:end]

	// Pass 2.  Process each line in the run.
	p = content
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, slashslash) {
			line = bytes.TrimSpace(line[len(slashslash):])
			if len(line) > 0 && line[0] == '+' {
				// Looks like a comment +line.
				f := strings.Fields(string(line))
				if f[0] == "+build" {
					ok := false
					for _, tok := range f[1:] {
						if ctxt.match(tok) {
							ok = true
							break
						}
					}
					if !ok {
						return false // this one doesn't match
					}
				}
			}
		}
	}
	return true // everything matches
}

// from $GOROOT/src/pkg/go/build/build.go

// saveCgo saves the information from the #cgo lines in the import "C" comment.
// These lines set CFLAGS and LDFLAGS and pkg-config directives that affect
// the way cgo's C code is built.
//
// TODO(rsc): This duplicates code in cgo.
// Once the dust settles, remove this code from cgo.
func (ctx *Spec) saveCgo(pkg *Package, filename string, cg *ast.CommentGroup) error {
	text := cg.Text()
	for _, line := range strings.Split(text, "\n") {
		orig := line

		// Line is
		//      #cgo [GOOS/GOARCH...] LDFLAGS: stuff
		//
		line = strings.TrimSpace(line)
		if len(line) < 5 || line[:4] != "#cgo" || (line[4] != ' ' && line[4] != '\t') {
			continue
		}

		// Split at colon.
		line = strings.TrimSpace(line[4:])
		i := strings.Index(line, ":")
		if i < 0 {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}
		line, argstr := line[:i], line[i+1:]

		// Parse GOOS/GOARCH stuff.
		f := strings.Fields(line)
		if len(f) < 1 {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}

		cond, verb := f[:len(f)-1], f[len(f)-1]
		if len(cond) > 0 {
			ok := false
			for _, c := range cond {
				if ctx.match(c) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		args, err := splitQuoted(argstr)
		if err != nil {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}
		for _, arg := range args {
			if !safeName(arg) {
				return fmt.Errorf("%s: malformed #cgo argument: %s", filename, arg)
			}
		}

		switch verb {
		case "CFLAGS":
			pkg.CgoCFLAGS = append(pkg.CgoCFLAGS, args...)
		case "LDFLAGS":
			pkg.CgoLDFLAGS = append(pkg.CgoLDFLAGS, args...)
		case "pkg-config":
			pkg.CgoPkgConfig = append(pkg.CgoPkgConfig, args...)
		default:
			return fmt.Errorf("%s: invalid #cgo verb: %s", filename, orig)
		}
	}
	return nil
}
