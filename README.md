# gogo


`gogo` is an experimental alternative build tool for the Go programming language.

## goals

 1. An alternative build tool that is independent of Go itself.

 2. Extensible, ie. other Go programs should be able to build Go code without having to shell out to the `go` tool, other code can reuse `gogo` components to make their own build automation.

 3. Project based, not package based, supporting some (as yet to be defined) package version system.

 4. Potentially support code generation phases and other pre and post targets.

## installation

The `gogo` command is `go get`able

    go get github.com/davecheney/gogo/gogo

## usage

`gogo` is not ready for use, you'd be mad to use it. In the case that you _are_ mad, create a project workspace and a `.gogo` subdirectory. The `.gogo` subdirectory is used by the `gogo` tool to locate the root of your project.

    mkdir -p $PROJECT/.gogo

Inside your `gogo` project, you should arrange your Go source, and its dependencies into the usual `$PROJECT/src` subfolder. You can also use your existing $GOPATH directory as a project location, just `mkdir -p $GOPATH/.gogo`. `gogo` will not overwrite the output of the `go` tool.

### common flags

#### logging output

`gogo` has three logging levels, DEBUG, INFO, and ERROR. The following flags control logging output.

    -q suppresses log messages below ERROR level

    -v enables log messges below INFO level

### gogo build

`gogo` can build a package or a command, using the `build` subcommand. When commands are built, they are placed in `$PROJECT/bin/$GOOS/$GOARCH/` (this path is subject to change)

    cd $PROJECT   # or a subdirectory of your project
    gogo build $SOME_PACKAGE_OR_COMMAND

`gogo` also supports a `-a` flag which will build all packages inside your $PROJECT/src directory.

    cd $PROJECT
    gogo build -a

### gogo test

`gogo` can invoke the standard `testing` package tests. Note, external tests are not yet supported.

    cd $PROJECT
    gogo test $SOME_PACKAGE

`gogo` also supports the `-a` flag which will build and test each package inside your $PROJECT/src directory.

    cd $PROJECT
    gogo test -a

## documentation

[godoc.org/github.com/davecheney/gogo](http://godoc.org/github.com/davecheney/gogo)

## build status

[![Build Status](https://drone.io/github.com/davecheney/gogo/status.png)](https://drone.io/github.com/davecheney/gogo/latest)

## mailing list

A mailing list for `gogo`, `gogo-users` has been created, [https://groups.google.com/d/forum/gogo-users](https://groups.google.com/d/forum/gogo-users)

## changelog

[http://starlogs.net/#davecheney/gogo](changelog)

## faq

 * Q. Can `gogo` build to Go standard library ? A. No.
 * Q. Will relative imports be supported ? A. No, they are evil.

## todo

 * better package parsing (support all file types)
 * improve cgo support
 * use the correct archchar for 5,6, and 8g.
 * build tags support
 * support external tests in package, XTestGoFiles
 * implement incremental builds; only build what needs to be built

## licence

`gogo` is licenced under a BSD licence. Various parts of `gogo` are copyright the Go Authors 2012.
