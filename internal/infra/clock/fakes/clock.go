package fakes

import (
	"context"
	"time"
)

type Clock struct {
	NowValue time.Time
	NowErr   error
	CloseErr error
}

func (c Clock) Now(context.Context) (*time.Time, error) {
	if c.NowErr != nil {
		return nil, c.NowErr
	}
	now := c.NowValue
	return &now, nil
}

func (c Clock) Close() error {
	return c.CloseErr
}
