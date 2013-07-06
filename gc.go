package gogo

// gc toolchain

import (
	"go/build"
	"path/filepath"
)

type gcToolchain struct {
	toolchain
	gc, cc, ld, as, pack string
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
			gcc:     "/usr/bin/gcc",
			Context: c,
		},
		gc:   filepath.Join(tooldir, archchar+"g"),
		cc:   filepath.Join(tooldir, archchar+"c"),
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

func (t *gcToolchain) Cc(srcdir, objdir, outfile, cfile string) error {
	args := []string{"-F", "-V", "-w", "-I", objdir, "-I", filepath.Join(t.goroot, "pkg", t.goos+"_"+t.goarch)}
	args = append(args, "-o", outfile)
	args = append(args, cfile)
	return run(srcdir, t.cc, args...)
}

func (t *gcToolchain) Pack(pkg *Package, ofiles ...string) error {
	args := []string{"grcP", t.Workdir(), afile}
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
