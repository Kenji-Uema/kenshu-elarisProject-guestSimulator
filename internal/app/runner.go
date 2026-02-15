package app

import (
	"context"
	"log/slog"
)

type Runner struct {
	machine            *Machine
	concurrencyLevel   int
	finishNotification chan bool
}

func NewRunner(machine *Machine, concurrencyLevel int) *Runner {
	return &Runner{machine: machine, concurrencyLevel: concurrencyLevel, finishNotification: make(chan bool, concurrencyLevel)}
}

func (r *Runner) Run(ctx context.Context) {
	r.coldStart(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.finishNotification:
			r.startMachine(ctx)
		}
	}
}

func (r *Runner) coldStart(ctx context.Context) {
	for i := 0; i < r.concurrencyLevel; i++ {
		r.startMachine(ctx)
	}
}

func (r *Runner) startMachine(ctx context.Context) {
	go func() {
		if err := r.machine.Start(ctx); err != nil {
			slog.ErrorContext(ctx, "machine stopped with error", "err", err)
		}
		r.finishNotification <- true
	}()
}
