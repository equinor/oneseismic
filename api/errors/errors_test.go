package errors

import (
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	e1 := E(Op("foo"), fmt.Errorf("bar"))
	e2 := E(Op("bat"), e1)
	in, ok := e2.(*Error)

	if !ok {
		t.Errorf("expected type Error")
		return
	}
	if in.Op != "bat" {
		t.Errorf("expected %q: got %q", "bat", in.Op)
	}
	in2, ok := in.Err.(*Error)
	if !ok {
		t.Errorf("expected type Error")
		return
	}
	if in2.Op != "foo" {
		t.Errorf("expected %q: got %q", "foo", in2.Op)
	}
}
