package port

import "context"

type Redis interface {
	Ping(ctx context.Context) error
	Close() error
}
