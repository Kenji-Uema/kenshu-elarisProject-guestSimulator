package domain

import (
	"slices"
	"testing"
)

func TestSetAddRemoveValues(t *testing.T) {
	set := NewSet[string]()

	set.Add("alpha")
	set.Add("beta")
	set.Add("alpha")

	values := set.Values()
	slices.Sort(values)

	if !slices.Equal(values, []string{"alpha", "beta"}) {
		t.Fatalf("unexpected values: %v", values)
	}

	set.Remove("alpha")

	values = set.Values()
	if len(values) != 1 || values[0] != "beta" {
		t.Fatalf("unexpected values after remove: %v", values)
	}
}
