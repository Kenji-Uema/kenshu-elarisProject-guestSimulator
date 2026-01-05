package utils

import (
	"guestEmulator/internal/domain"
	"log"
	"math/rand"
	"time"

	"github.com/go-resty/resty/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	if total <= 0 {
		return PickRandom(list).Value
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

func NewGrpcConnection(addr string) (conn *grpc.ClientConn) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	return conn
}

func CloseGrpcConnection(conn *grpc.ClientConn) {
	err := conn.Close()
	if err != nil {
		log.Fatalf("did not close the connection: %v", err)
	}
}

func NewRestyClient(url string) *resty.Client {
	return resty.New().
		SetTimeout(5 * time.Second).
		SetBaseURL(url)
}
