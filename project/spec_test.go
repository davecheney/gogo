package project

import (
	"runtime"
	"testing"
)

func TestDefaultSpec(t *testing.T) {
	s := DefaultSpec()
	if s.goos != runtime.GOOS || s.goarch != runtime.GOARCH {
		t.Fatalf("expected (%s, %s), got (%s, %s)", s.goos, s.goarch, runtime.GOOS, runtime.GOARCH)
	}
}

var testSpec = Spec{goos: "linux", goarch: "amd64"}

var matchTests = []struct {
	Spec
	name    string
	matched bool
}{
	{testSpec, "linux", true},
	{testSpec, "darwin", false},
	{testSpec, "amd64", true},
	{testSpec, "386", false},
	{testSpec, "linux darwin", false},
	{testSpec, "linux,darwin", false},
	{testSpec, "!linux !darwin", true},
}

func TestSpecMatch(t *testing.T) {
	for _, tt := range matchTests {
		if v := tt.Spec.match(tt.name); v != tt.matched {
			t.Errorf("%#v.match(%s): expected %v, got %v", tt.Spec, tt.name, tt.matched, v)
		}
	}
}

func s(st string) []byte { return []byte(st) }

var shouldBuildTests = []struct {
	Spec
	content []byte
	matched bool
}{
	{testSpec, s("// +build linux"), true},
	{testSpec, s("// +build !linux"), true},
	{testSpec, s("// +build !linux !amd64"), true},
}

func TestSpecShouldBuild(t *testing.T) {
	for _, tt := range shouldBuildTests {
		if v := tt.Spec.shouldBuild(tt.content); v != tt.matched {
			t.Errorf("%#v.sohuldBuild(%q): expected %v, got %v", tt.Spec, string(tt.content), tt.matched, v)
		}
	}
}
