package domain

type WeightedTuple[T any] struct {
	Value  T
	Weight float64
}
