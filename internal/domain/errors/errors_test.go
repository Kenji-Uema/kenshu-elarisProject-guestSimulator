package errors

import "testing"

func TestErrInvalidInputTypeIsDefined(t *testing.T) {
	if ErrInvalidInputType == nil {
		t.Fatal("expected error to be defined")
	}
}
