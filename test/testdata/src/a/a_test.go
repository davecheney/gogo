package a

import "testing"

func TestHello(t *testing.T) { if Hello() != "helloworld" { t.Fatalf("test failed") } }
