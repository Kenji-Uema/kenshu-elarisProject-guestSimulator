package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestEmulator/internal/config"
	clockEmuProto "github.com/Kenji-Uema/guestEmulator/internal/transport/grpc/pb/clockEmu"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type Emu struct {
	conn   *grpc.ClientConn
	client clockEmuProto.ClockServiceClient
}

func NewClockEmu(cfg config.ServicesConfig) (*Emu, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockEmuGrpcUrl, cfg.ClockEmuGrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Emu{conn: conn, client: clockEmuProto.NewClockServiceClient(conn)}, nil
}

func (e *Emu) Close() error {
	return e.conn.Close()
}

func (e *Emu) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
