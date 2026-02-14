package grpc

import (
	"log/slog"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
