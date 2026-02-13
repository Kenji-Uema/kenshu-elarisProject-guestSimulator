package utils

import (
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"

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
		slog.Error("did not connect", "err", err)
		os.Exit(1)
	}

	return conn
}

func CloseGrpcConnection(conn *grpc.ClientConn) {
	err := conn.Close()
	if err != nil {
		slog.Error("did not close the connection", "err", err)
		os.Exit(1)
	}
}

func NewRestyClient(url string) *resty.Client {
	return resty.New().
		SetTimeout(5 * time.Second).
		SetBaseURL(url)
}
