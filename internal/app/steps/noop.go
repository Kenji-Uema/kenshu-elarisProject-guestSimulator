package steps

import "context"

type noopStep struct{}

func NewNoopStep() Step {
	return noopStep{}
}

func (noopStep) Name() string {
	return "GuestJourneyNoopStep"
}

func (noopStep) Validate() error {
	return nil
}

func (noopStep) Execute(context.Context) error {
	return nil
}
