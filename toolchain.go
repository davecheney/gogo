package gogo

import (
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"go/build"
)

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(importpath, srcdir, outfile string, files []string) error
	Asm(srcdir, ofile, sfile string) error
	Pack(string, string, ...string) error
	Ld(string, string) error
	Cgo(objdir string, cgofiles []string) error

	name() string
}

type toolchain struct {
	cgo string
	*Context
}

func (t *toolchain) Cgo(objdir string, cgofiles []string) error {
	args := []string{"-objdir", objdir, "--", "-I", objdir}
	args = append(args, cgofiles...)
	return run(t.Workdir(), t.cgo, args...)
}

type gcToolchain struct {
	toolchain
	gc, ld, as, pack string
}

func newGcToolchain(c *Context) (Toolchain, error) {
	tooldir := filepath.Join(c.goroot, "pkg", "tool", c.goos+"_"+c.goarch)
	archchar, err := build.ArchChar(c.goarch)
	if err != nil {
		return nil, err
	}
	return &gcToolchain{
		toolchain: toolchain{
			cgo:     filepath.Join(tooldir, "cgo"),
			Context: c,
		},
		gc:   filepath.Join(tooldir, archchar+"g"),
		ld:   filepath.Join(tooldir, archchar+"l"),
		as:   filepath.Join(tooldir, archchar+"a"),
		pack: filepath.Join(tooldir, "pack"),
	}, nil
}

func (t *gcToolchain) name() string { return "gc" }

func (t *gcToolchain) Gc(importpath, srcdir, outfile string, files []string) error {
	args := []string{"-p", importpath}
	for _, d := range t.SearchPaths {
		args = append(args, "-I", d)
	}
	args = append(args, "-o", outfile)
	args = append(args, files...)
	return run(srcdir, t.gc, args...)
}

func (t *gcToolchain) Pack(afile, objdir string, ofiles ...string) error {
	args := []string{"grcP", t.Workdir(), filepath.Join(t.Workdir(), afile)}
	args = append(args, ofiles...)
	return run(objdir, t.pack, args...)
}

func (t *gcToolchain) Asm(srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, sfile}
	return run(srcdir, t.as, args...)
}

func (t *gcToolchain) Ld(outfile, afile string) error {
	args := []string{"-o", outfile}
	for _, d := range t.SearchPaths {
		args = append(args, "-L", d)
	}
	args = append(args, afile)
	return run(t.Workdir(), t.ld, args...)
}

func run(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("cd %s; %s %s", dir, command, strings.Join(args, " "))
		log.Printf("%s", output)
	}
	return err
}
