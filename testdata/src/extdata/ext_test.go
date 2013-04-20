package extdata_test

import ( 
	"testing"
	"extdata"
)

func TestExternal(t *testing.T) {
	if extdata.A != "extdata" {
		t.Errorf("expected %q, got %q", "extdata", extdata.A)
	}
}
