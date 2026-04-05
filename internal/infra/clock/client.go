package clock

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type Clock struct {
	conn   *grpc.ClientConn
	client ClockServiceClient
}

func NewClock(cfg config.ServicesConfig) (port.Clock, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockEmuGrpcUrl, cfg.ClockEmuGrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Clock{conn: conn, client: NewClockServiceClient(conn)}, nil
}

func (c *Clock) Close() error {
	return c.conn.Close()
}

func (c *Clock) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := c.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
