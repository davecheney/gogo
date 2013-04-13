gogo
====

`gogo` is an experimental alternative build tool for the Go programming language.

goals
-----

 1. An alternative build tool that is independent of go itself

 2. Extensible, ie other go program's should be able to build go code without having to shell out to the go tool, and other code can reuse `gogo` components to make their own build automation.

 3. Project based, not package based, supporting some (as yet to be defined) package version system.

 4. Potentially support code generation phases and other pre and post targets.

usage
-----

`gogo` is not ready for use, you'd be mad to use it. In the case that you _are_ mad, create a project workspace 

    mkdir $PROJECT
    mkdir $PROJECT/.project

    cd $PROJECT
    gogo build $SOME_PACKAGE_OR_COMMAND

`gogo` expects all your source to be in $PROJECT/src in GOPATH format.

faq
---

 * Q. Will `gogo` build Go itself ? A. No.

todo
----

 * support cgo packages.
 * add test command.
 * `gogo` should be able to build and test itself.

licence
-------

`gogo` is licenced under a BSD licence.
