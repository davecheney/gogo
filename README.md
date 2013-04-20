# gogo


`gogo` is an experimental alternative build tool for the Go programming language.

## goals

 1. An alternative build tool that is independent of Go itself

 2. Extensible, ie other go program's should be able to build Go code without having to shell out to the `go` tool, and other code can reuse `gogo` components to make their own build automation.

 3. Project based, not package based, supporting some (as yet to be defined) package version system.

 4. Potentially support code generation phases and other pre and post targets.

## installation

The `gogo` command is `go get`able

    go get github.com/davecheney/gogo/gogo

## usage

`gogo` is not ready for use, you'd be mad to use it. In the case that you _are_ mad, create a project workspace 

    mkdir $PROJECT

Inside your `gogo` project, you should arrange your Go source, and its dependencies into the usual `$PACKAGE/src` subfolder.

### gogo build

`gogo` can build a package or a command, using the `build` subcommand. When commands are built, they are placed in `$PROJECT/bin/$GOOS/$GOARCH/` (this path is subject to change)

    cd $PROJECT
    gogo build $SOME_PACKAGE_OR_COMMAND

### gogo test

`gogo` can invoke the standard `testing` package tests. 

    cd $PROJECT
    gogo test $SOME_PACKAGE

## faq

 * Q. Will `gogo` build Go itself ? A. No.
 * Q. Will relative imports be supported ? A. No, they are evil.

## todo

 * better package parsing (support all file types)
 * cgo support for build and test
 * project auto discovery; be able to invoke `gogo` from anywhere inside the project.
 * `gogo` should be able to ~~build~~ test itself.
 * build tags support
 * support external tests in package, XTestGoFiles
 * implement incremental builds; only build what needs to be built

## licence

`gogo` is licenced under a BSD licence. Various parts of gogo are copyright the Go Authors 2012.
