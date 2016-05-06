package lego

import ()

// A Source represents a blocktype that produces data through a puller.
type Source interface {
	Intake() Puller
}

// A Sink represents a blocktype that consumes data through a pusher.
type Sink interface {
	Output() Pusher
}

// A Pipe represents a blocktype that consumes and produces data through a PullPusher.
type Pipe interface {
	Transfer() PullPusher
}

// Puller represents a block source that can be pulled from.
type Puller interface {
	Pull() <-chan interface{} // stream of items
	Close() error
}

// Pusher represents a block sink that can be pushed to.
type Pusher interface {
	Push() chan<- interface{}
	Close() error
}

// PullPusher represents a block pipe that can be pushed and pulled from.
type PullPusher interface {
	Push() chan<- interface{}
	Pull() <-chan interface{}
	Close() error
}

type Closer interface {
	Close() error
}
