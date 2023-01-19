package main

import (
	"context"
	"time"
)

// cancellationSource implements the context.Context interface and allows a
// caller to distinguish between one of two possible contexts for which one was
// responsible for cancellation, by testing for identity against the `final'
// struct member.
type cancellationSource struct {
	a     contextSource
	b     contextSource
	final contextSource
	done  chan struct{}
	err   error
}

type contextSource struct {
	context.Context
	source cancelSourceID
}

type cancelSourceID int

const (
	cancelSourceFileChange cancelSourceID = iota
	cancelSourceSignal
)

func newCancellationSource(a contextSource, b contextSource) *cancellationSource {
	s := new(cancellationSource)
	s.a = a
	s.b = b
	s.done = make(chan struct{})
	go s.run()
	return s
}

func (s *cancellationSource) run() {
	select {
	case <-s.a.Done():
		s.final = s.a
		s.err = s.final.Err()
	case <-s.b.Done():
		s.final = s.b
		s.err = s.final.Err()
	case <-s.done:
		return
	}
	close(s.done)
}

func (s *cancellationSource) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (s *cancellationSource) Done() <-chan struct{} {
	return s.done
}

func (s *cancellationSource) Err() error {
	return s.err
}

var _ context.Context = (*cancellationSource)(nil)

func (s *cancellationSource) Value(key any) any {
	panic("not implemented") // TODO: Implement
}
