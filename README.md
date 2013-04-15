# gogo


`gogo` is an experimental alternative build tool for the Go programming language.

## goals

 1. An alternative build tool that is independent of go itself

 2. Extensible, ie other go program's should be able to build go code without having to shell out to the go tool, and other code can reuse `gogo` components to make their own build automation.

 3. Project based, not package based, supporting some (as yet to be defined) package version system.

 4. Potentially support code generation phases and other pre and post targets.

### usage

`gogo` is not ready for use, you'd be mad to use it. In the case that you _are_ mad, create a project workspace 

    mkdir $PROJECT

Inside your `gogo` project, you should arrange your Go source, and its dependencies into the usual `$PACKAGE/src` subfolder.

### gogo build

`gogo` can build a package or a command, using the `build` subcommand. When commands are built, they are placed in `$PROJECT/bin/$GOOS/$GOARCH/` (this path is subject to change)

    cd $PROJECT
    gogo build $SOME_PACKAGE_OR_COMMAND

### gogo test

`gogo` can invoke the standard `testing` tests. Only packages are supported at this time, commands will be implemented later.

    cd $PROJECT
    gogo test $SOME_PACKAGE

## faq

 * Q. Will `gogo` build Go itself ? A. No.

## todo

 * support cgo packages.
 * ~~Add test command.~~
 * Add support for testing commands.
 * be able to invoke `gogo` from anywhere inside the project.
 * ~~`gogo` should be able to build and test itself.~~
 * implement incremental builds; only build what needs to be built

## licence

`gogo` is licenced under a BSD licence.
