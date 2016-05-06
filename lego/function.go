package lego

import ()

// FunctionSink implements the Sink interface by writing to the given function
type FunctionSink struct {
	Func func(interface{})
}

// FunctionSource implements the Source interface by reading from the given function
type FunctionSource struct {
	Func func() interface{}
}

// Intake creates, starts and returns a Puller Brick from a function.
func (s *FunctionSource) Intake() Puller {
	p := &funcSource{
		legoBrickSource: legoBrickSource{newBrick()},
		Func:            s.Func,
	}
	go p.loop()
	return p
}

// Output creates, starts and returns a Pusher Brick to a function.
func (s *FunctionSink) Output() Pusher {
	p := &funcSink{
		legoBrickSink: legoBrickSink{newBrick()},
		Func:          s.Func,
	}
	go p.loop()
	return p
}

type funcSource struct {
	legoBrickSource

	Func func() interface{}
}

func (p *funcSource) loop() {
	for {
		it := p.Func()
		select {
		case p.pipe <- it:
		case errc := <-p.closing:
			close(p.pipe)
			errc <- nil
			return
		}
	}
}

type funcSink struct {
	legoBrickSink

	Func func(interface{})
}

func (p *funcSink) loop() {
	for {
		in := p.pipe
		select {
		case d, ok := <-in:
			if !ok {
				in = nil
				continue
			}
			p.Func(d)
		case errc := <-p.closing:
			errc <- nil
			return
		}
	}
}
