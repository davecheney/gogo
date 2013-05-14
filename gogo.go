// gogo is an alternative build tool for the Go programming language.
//
// Package gogo provides the basic types and interfaces to describe Projects and
// Packages.
//
// The gogo command can be installed with go get
//
//	go get github.com/davecheney/gogo/gogo
//
package gogo

import (
	"os/exec"
	"strings"

	"log"
)

// A Future represents some work to be performed.
type Future interface {
	// Result returns the result of the work as an error, or nil if the work
	// was performed successfully.
	// Implementers must observe these invariants
	// 1. There may be multiple concurrent callers to Result, or Result may
	//     be called many times in sequence, it must always return the same
	//     value.
	// 2. Result blocks until the work has been performed.
	Result() error
}

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(importpath, srcdir, outfile string, files []string) error
	Asm(srcdir, ofile, sfile string) error
	Pack(string, string, ...string) error
	Ld(string, string) error
	Cc(srcdir, objdir, ofile, cfile string) error

	Cgo(string, []string) error
	Gcc(string, []string) error
	Libgcc() (string, error)

	name() string
}

type toolchain struct {
	cgo string
	gcc string
	*Context
}

func (t *toolchain) Cgo(cwd string, args []string) error {
	return run(cwd, t.cgo, args...)
}

func (t *toolchain) Gcc(cwd string, args []string) error {
	return run(cwd, t.gcc, args...)
}

func (t *toolchain) Libgcc() (string, error) {
	libgcc, err := runOut(".", t.gcc, "-print-libgcc-file-name")
	return strings.Trim(string(libgcc), "\r\n"), err
}

var toolchains = map[string]func(*Context) (Toolchain, error){
	"gc":    newGcToolchain,
	"gccgo": newGccgoToolchain,
}

func run(dir, command string, args ...string) error {
	_, err := runOut(dir, command, args...)
	return err
}

func runOut(dir, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("cd %s; %s %s", dir, command, strings.Join(args, " "))
		log.Printf("%s", output)
	}
	return output, err
}
