package lego

import ()

// Transformer holds a function to transform data to implement the Pipe interface.
type Transformer struct {
	Transform func(interface{}) interface{}
}

// Transfer creates, starts and returns a PullPusher Brick that transforms
// the data using its Transform function
func (s *Transformer) Transfer() PullPusher {
	b := transformerBrick{
		source: legoBrickSource{legoBrick{
			closing: make(chan chan error),
			pipe:    make(chan interface{}),
		}},
		sink: legoBrickSink{legoBrick{
			closing: make(chan chan error),
			pipe:    make(chan interface{}),
		}},
		transform: s.Transform,
	}
	go b.loop()
	return &b
}

// TODO: any advantage to holding sink and source as bricks?
// find out a way to simplify.
type transformerBrick struct {
	source    legoBrickSource
	sink      legoBrickSink
	transform func(interface{}) interface{}
}

func (b *transformerBrick) Pull() <-chan interface{} {
	return b.source.Pull()
}

func (b *transformerBrick) Push() chan<- interface{} {
	return b.sink.Push()
}

func (b *transformerBrick) Close() error {
	errc := make(chan error)
	b.source.closing <- errc
	return <-errc
}

// user -> sink -> transform -> source -> user
// transformerBrick internally pulls from its sink and pushes to its source
func (b *transformerBrick) loop() {
	var in <-chan interface{} = b.sink.pipe
	var out chan<- interface{} = b.source.pipe
	for {
		select {
		case d, ok := <-in:
			if !ok {
				close(out)
				in = nil
				continue
			}
			out <- b.transform(d)
		case errc := <-b.source.closing:
			errc <- nil
			return
		}
	}
}
