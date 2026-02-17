package steps

import (
	"context"
)

type Step interface {
	Validate() error
	Execute(ctx context.Context) error
	Name() string
}
