package build

// gccgo toolchain

import (
	"path/filepath"
)

type gccgoToolchain struct {
	toolchain
	gccgo string // path to gccgo
}

func newGccgoToolchain(c *Context) (Toolchain, error) {
	tooldir := filepath.Join(c.goroot, "pkg", "tool", c.goos+"_"+c.goarch)
	return &gccgoToolchain{
		toolchain: toolchain{
			cgo:     filepath.Join(tooldir, "cgo"),
			gcc:     "/usr/bin/gcc",
			Context: c,
		},
		gccgo: "gccgo",
	}, nil
}

func (t *gccgoToolchain) name() string { return "gc" }

func (t *gccgoToolchain) Gc(importpath, srcdir, outfile string, files []string) error {
	args := []string{"-c", "-g", "-m64"}
	for _, d := range t.SearchPaths {
		args = append(args, "-I", d)
	}
	args = append(args, "-fgo-pkgpath="+importpath)
	args = append(args, "-fgo-relative-import-path=_"+srcdir)
	args = append(args, "-o", outfile)
	args = append(args, files...)
	return run(srcdir, t.gccgo, args...)
}

func (t *gccgoToolchain) Cc(srcdir, objdir, outfile, cfile string) error {
	args := []string{"-F", "-V", "-w", "-I", objdir, "-I", filepath.Join(t.goroot, "pkg", t.goos+"_"+t.goarch)}
	args = append(args, "-o", outfile)
	args = append(args, cfile)
	return run(srcdir, t.gccgo, args...)
}

func (t *gccgoToolchain) Pack(afile string, ofiles ...string) error {
	// hack afile
	dir, file := filepath.Split(afile)
	args := []string{"cru", filepath.Join(dir, "lib"+file)}
	args = append(args, ofiles...)
	return run(filepath.Dir(afile), "ar", args...)
}

func (t *gccgoToolchain) Asm(srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, sfile}
	return run(srcdir, t.gccgo, args...)
}

func (t *gccgoToolchain) Ld(outfile, afile string) error {
	args := []string{"-o", outfile}
	for _, d := range t.SearchPaths {
		args = append(args, "-L", d)
	}
	args = append(args, afile)
	return run(t.Workdir(), t.gccgo, args...)
}
