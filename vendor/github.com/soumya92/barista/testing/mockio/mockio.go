// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mockio provides infinite streams that can be used for testing stdin/stdout.
package mockio

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// Writable is an infinite stream that satisfies io.Writer,
// and adds methods to get portions of the output written to it.
type Writable struct {
	// The buffer that holds all non-consumed output.
	buffer bytes.Buffer
	// A channel that signals any time new output is available.
	signal chan *interface{}
	// Mutex to prevent data races in buffer.
	mutex sync.Mutex
}

// Write satisfies the io.Writer interface.
func (w *Writable) Write(out []byte) (n int, e error) {
	w.mutex.Lock()
	n, e = w.buffer.Write(out)
	w.mutex.Unlock()
	nonBlockingSignal(w.signal)
	return
}

var _ io.Writer = (*Writable)(nil)

// ReadNow clears the buffer and returns its previous contents.
func (w *Writable) ReadNow() string {
	w.mutex.Lock()
	val := w.buffer.String()
	w.mutex.Unlock()
	w.buffer = bytes.Buffer{}
	return val
}

// ReadUntil reads up to the first occurrence of the given character,
// or until the timeout expires, whichever comes first.
func (w *Writable) ReadUntil(delim byte, timeout time.Duration) (string, error) {
	w.mutex.Lock()
	val, err := w.buffer.ReadString(delim)
	w.mutex.Unlock()
	if err == nil {
		return val, nil
	}
	timeoutChan := time.After(timeout)
	// EOF means we ran out of bytes, so we need to wait until more are written.
	for err == io.EOF {
		select {
		case <-timeoutChan:
			return val, err
		case <-w.signal:
			var v string
			w.mutex.Lock()
			v, err = w.buffer.ReadString(delim)
			w.mutex.Unlock()
			val += v
		}
	}
	return val, err
}

// Stdout returns a Writable that can be used for making assertions
// against what was written to stdout.
func Stdout() *Writable {
	return &Writable{
		buffer: bytes.Buffer{},
		signal: make(chan *interface{}),
	}
}

// Readable is an infinite stream that satisfies io.Reader and io.Writer
// Reads block until something is written to the stream, which mimics stdin.
type Readable struct {
	// The buffer that holds all non-consumed output.
	buffer bytes.Buffer
	// A channel that signals any time new output is available.
	available chan *interface{}
	// A channel that signals any time output is consumed.
	consumed chan *interface{}
	// Mutex to prevent data races in buffer.
	mutex sync.Mutex
}

// Read satisfies the io.Reader interface.
func (r *Readable) Read(out []byte) (n int, e error) {
	r.mutex.Lock()
	len := r.buffer.Len()
	r.mutex.Unlock()
	if len == 0 {
		<-r.available
	}
	r.mutex.Lock()
	n, e = r.buffer.Read(out)
	r.mutex.Unlock()
	nonBlockingSignal(r.consumed)
	if e == io.EOF {
		e = nil
	}
	return
}

// Write satisfies the io.Writer interface.
func (r *Readable) Write(out []byte) (n int, e error) {
	r.mutex.Lock()
	n, e = r.buffer.Write(out)
	r.mutex.Unlock()
	r.signalWrite()
	return
}

var _ io.Reader = (*Readable)(nil)
var _ io.Writer = (*Readable)(nil)

// WriteString proxies directly to the byte buffer but adds a signal.
func (r *Readable) WriteString(s string) (n int, e error) {
	r.mutex.Lock()
	n, e = r.buffer.WriteString(s)
	r.mutex.Unlock()
	r.signalWrite()
	return
}

// signalWrite signals that data was written, and waits for it to be consumed.
func (r *Readable) signalWrite() {
	if nonBlockingSignal(r.available) {
		<-r.consumed
	}
}

// Stdin returns a Readable that can be used in place of stdin for testing.
func Stdin() *Readable {
	return &Readable{
		buffer:    bytes.Buffer{},
		available: make(chan *interface{}),
		consumed:  make(chan *interface{}),
	}
}

// nonBlockingSignal sends a signal, but only if there are listeners.
func nonBlockingSignal(ch chan<- *interface{}) bool {
	select {
	case ch <- nil:
		return true
	default:
		return false
	}
}
