package extdata_test

import (
	"extdata"
	"testing"
)

func TestExternal(t *testing.T) {
	if extdata.A != "extdata" {
		t.Errorf("expected %q, got %q", "extdata", extdata.A)
	}
}
