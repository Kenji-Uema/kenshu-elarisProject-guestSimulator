package utils

import (
	"math/rand"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
)

func PickRandom[T any](list []T) T {
	return list[rand.Intn(len(list))]
}

func PickRandomWeighted[T any](list []domain.WeightedTuple[T]) T {
	total := 0.0
	for _, item := range list {
		if item.Weight > 0 {
			total += item.Weight
		}
	}

	target := rand.Float64() * total
	for _, item := range list {
		if item.Weight <= 0 {
			continue
		}
		target -= item.Weight
		if target <= 0 {
			return item.Value
		}
	}

	return list[len(list)-1].Value
}
