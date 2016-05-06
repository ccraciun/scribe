package lego

import (
	"time"
)

// Brik types.

// These could(should?) be parameters per brick.
const maxPending = 10              // stop fetching while more than maxPending items in pending.
const fetchDelay = 1 * time.Minute // Default wait between fetches.
// maxConsecutiveRetries int
// retryDelay            time.Duration

// Internal representations of lego bricks.
type legoBrick struct {
	closing chan chan error  // closing concurrency pattern
	pipe    chan interface{} // delivers items to user
	err     error            // stored last error encountered
}

type legoBrickSource struct {
	legoBrick
}

type legoBrickSink struct {
	legoBrick
}

// NewBrick creates and returns a new brick with default channels.
func newBrick() legoBrick {
	return legoBrick{
		closing: make(chan chan error),
		pipe:    make(chan interface{}),
	}
}

// Source Bricks implement Puller
func (b *legoBrickSource) Pull() <-chan interface{} {
	return b.pipe
}

// Sink Bricks implement Pusher
func (b *legoBrickSink) Push() chan<- interface{} {
	return b.pipe
}

// All bricks implement close. But they shouldn't need to have close called.
// TODO: better define who is responsible for closing and create an option for brick that
// shuts down on channel close, after abstracting the brick.loop() out.
func (b *legoBrick) Close() error {
	errc := make(chan error)
	b.closing <- errc
	return <-errc
}
