// Package stream defines a structure to communication between inputs and outputs
package stream

import "sync"

// Stream makes packages able to subscribe to an incoming stream
type Stream struct {
	// Incoming data come from this channel
	Broadcast chan<- interface{}

	// Use a map to be able to delete an item
	outputs map[chan<- interface{}]struct{}

	// Mutex to lock this ressource
	lock sync.Mutex
}

// New creates a new stream.
func New() *Stream {
	s := &Stream{}
	broadcast := make(chan interface{}, 64)
	s.Broadcast = broadcast
	s.outputs = make(map[chan<- interface{}]struct{})
	go s.run(broadcast)
	return s
}

func (s *Stream) run(broadcast <-chan interface{}) {
	for msg := range broadcast {
		func() {
			s.lock.Lock()
			defer s.lock.Unlock()
			for output := range s.outputs {
				select {
				case output <- msg:
				default:
					// Remove output if failed
					delete(s.outputs, output)
					close(output)
				}
			}
		}()
	}

	// Incoming chan has been closed, close all outputs
	s.lock.Lock()
	defer s.lock.Unlock()
	for ch := range s.outputs {
		delete(s.outputs, ch)
		close(ch)
	}
}

// Close the incoming chan, this will also delete all outputs
func (s *Stream) Close() {
	close(s.Broadcast)
}

// Register a new output on a stream
func (s *Stream) Register(output chan<- interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.outputs[output] = struct{}{}
}

// Unregister removes an output
func (s *Stream) Unregister(output chan<- interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Make sure we did not already close this output
	_, ok := s.outputs[output]
	if ok {
		delete(s.outputs, output)
		close(output)
	}
}
