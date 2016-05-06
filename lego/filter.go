package lego

import (
	"fmt"
)

// Sieve holds a function to filter data to implement the Pipe interface.
type Sieve struct {
	Filter func(interface{}) bool
}

// Transfer creates, starts and returns a PullPusher Brick that filters
// the data using its Filter function
func (s *Sieve) Transfer() PullPusher {
	b := filterBrick{
		source: legoBrickSource{newBrick()},
		sink:   legoBrickSink{newBrick()},
		filter: s.Filter,
	}
	go b.loop()
	return &b
}

type filterBrick struct {
	source legoBrickSource
	sink   legoBrickSink
	filter func(interface{}) bool
}

func (b *filterBrick) Pull() <-chan interface{} {
	return b.source.Pull()
}

func (b *filterBrick) Push() chan<- interface{} {
	return b.sink.Push()
}

func (b *filterBrick) Close() error {
	e1, e2 := b.sink.Close(), b.source.Close()
	if e1 != nil {
		if e2 != nil {
			return fmt.Errorf("error from upstream: %v; error from downstream: %v", e1, e2)
		}
		return e1
	}
	if e2 != nil {
		return e2
	}
	return nil
}

// user -> sink -> filter -> source -> user
// filterBrick internally pulls from its sink and pushes to its source
func (b *filterBrick) loop() {
	var in <-chan interface{} = b.sink.pipe
	var out chan<- interface{} = b.source.pipe
	for {
		select {
		case errc := <-b.source.closing:
			errc <- b.sink.err
			close(b.source.pipe)
		case errc := <-b.sink.closing:
			errc <- b.source.err
			close(b.sink.pipe)
		case d, ok := <-in:
			if !ok {
				close(out)
				in = nil
			} else if b.filter(d) {
				out <- d
			}
		}
	}
}
