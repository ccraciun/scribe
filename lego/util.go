package lego

import (
	log "github.com/Sirupsen/logrus"
	"sync"
)

// CloseAndLog closes a Closer and logs the error, if any
func CloseAndLog(c Closer) {
	log.Debugf("Closing %v", c)
	err := c.Close()
	if err != nil {
		log.WithError(err).Errorf("Error from %v", c)
	}
}

// Merge merges multiple channels together.
// Copied wholesale from https://blog.golang.org/pipelines
func Merge(cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan int) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Chain ties together a Puller -> Pusher, and starts a goroutine to drive the chain.
// TODO: Chain shouldn't need to return a closer, or atleast should end the goroutine without
// waiting for a Close call
func Chain(intake Puller, output Pusher) Closer {
	s := &segment{
		intake:  intake,
		output:  output,
		closing: make(chan chan error),
	}
	go s.loop()
	return s
}

type segment struct {
	closing chan chan error
	intake  Puller
	output  Pusher
}

func (s *segment) loop() {
	in := s.intake.Pull()
	out := s.output.Push()
	for {
		select {
		case v, ok := <-in:
			if !ok {
				in = nil
				continue
			}
			out <- v
		case errc := <-s.closing:
			// TODO: close s.closing to catch double closes.
			errc <- nil
			close(out)
			return
		}
	}
}

func (s *segment) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}
