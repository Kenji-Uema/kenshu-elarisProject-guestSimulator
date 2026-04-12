package fakes

type Acknowledger struct {
	AckCalls  int
	NackCalls int
	RejectErr error
}

func (a *Acknowledger) Ack(uint64, bool) error {
	a.AckCalls++
	return nil
}

func (a *Acknowledger) Nack(uint64, bool, bool) error {
	a.NackCalls++
	return nil
}

func (a *Acknowledger) Reject(uint64, bool) error {
	return a.RejectErr
}
