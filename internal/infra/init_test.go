package infra

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRunCleanupRunsInReverseOrderAndJoinsErrors(t *testing.T) {
	errOne := errors.New("one")
	errTwo := errors.New("two")
	var order []string

	err := runCleanup(context.Background(), []func(context.Context) error{
		func(context.Context) error {
			order = append(order, "first")
			return errOne
		},
		func(context.Context) error {
			order = append(order, "second")
			return errTwo
		},
	})

	if !reflect.DeepEqual(order, []string{"second", "first"}) {
		t.Fatalf("unexpected cleanup order: %#v", order)
	}
	if !errors.Is(err, errOne) || !errors.Is(err, errTwo) {
		t.Fatalf("unexpected cleanup error: %v", err)
	}
}
