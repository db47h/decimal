package decimal

import "testing"

func TestMaxBase(t *testing.T) {
	if MaxBase != len(digits) {
		t.Fatalf("%d != %d", MaxBase, len(digits))
	}
}
