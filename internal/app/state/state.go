package state

import (
	"context"

	"github.com/Kenji-Uema/guestEmulator/internal/app/errors"
)

type State interface {
	Execute(ctx context.Context, in any) (any, error)
}

type ZeroState interface {
	Execute(ctx context.Context) (context.Context, error)
}

type stateTyped[IN any, OUT any] interface {
	Execute(ctx context.Context, in IN) (OUT, error)
}

type Adapter[IN any, OUT any] struct {
	State stateTyped[IN, OUT]
}

func (a Adapter[IN, OUT]) Execute(ctx context.Context, in any) (any, error) {
	typed, ok := in.(IN)
	if !ok {
		return nil, errors.ErrInvalidInputType
	}
	return a.State.Execute(ctx, typed)
}
