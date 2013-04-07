package gogo

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Toolchain interface {
	gc(*Context, *Package) error
}

type gcToolchain struct {
}

func (tc *gcToolchain) gc(ctx *Context, pkg *Package) error {
	var args []string
	for _, f := range pkg.GoFiles() {
		args = append(args, f)
	}
	tooldir := filepath.Join(runtime.GOROOT(), "pkg", "tool", "linux_amd64")
	return run(pkg.srcdir(), filepath.Join(tooldir, "6g"), args...)
}

func run(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[%s] %s %s", dir, command, strings.Join(args, " "))
	return cmd.Run()
}
