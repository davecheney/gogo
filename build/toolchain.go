package build

var toolchains = map[string]func(*Context) (Toolchain, error){
	"gc":    newGcToolchain,
	"gccgo": newGccgoToolchain,
}
