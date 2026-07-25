package z

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// WorkerPanic reports a panic raised inside a validation worker goroutine —
// almost always from a user closure in Refine, Transform or a custom Check.
//
// A panic cannot cross a goroutine boundary, so a panicking closure inside
// ParseParallelSlice, ConcurrentBatch or ConcurrentParseAny would otherwise take
// the process down with it, no matter what the caller wrapped the call in.
// Instead the panic is captured and re-raised on the calling goroutine, so
// recovering works exactly as it does for a sequential Parse:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        if wp, ok := r.(*WorkerPanic); ok {
//	            log.Printf("element %d panicked: %v", wp.Index, wp.Value)
//	        }
//	    }
//	}()
type WorkerPanic struct {
	// Op names the call the panic happened under, e.g. "ParseParallelSlice".
	Op string
	// Index is the input element being validated, or -1 if unknown.
	Index int
	// Value is whatever was passed to panic.
	Value any
	// Stack is the stack of the worker goroutine at the moment of the panic,
	// which the re-raised panic would otherwise lose.
	Stack []byte
}

func (e *WorkerPanic) Error() string {
	return fmt.Sprintf("z: %s: panic while validating element %d: %v\n\nworker goroutine stack:\n%s",
		e.Op, e.Index, e.Value, e.Stack)
}

// Unwrap exposes the panic value when it was already an error.
func (e *WorkerPanic) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

// panicRecord collects the first panic raised by any worker in a group.
type panicRecord struct {
	once    sync.Once
	stopped atomic.Bool
	value   any
	stack   []byte
	index   int
}

// capture is deferred inside a worker's per-element body. It records the first
// panic and asks the remaining workers to stop.
func (p *panicRecord) capture(index int) {
	if r := recover(); r != nil {
		p.once.Do(func() {
			p.value, p.stack, p.index = r, debug.Stack(), index
		})
		p.stopped.Store(true)
	}
}

// stop reports whether a worker has already panicked, so the rest can wind down
// instead of doing work whose result is about to be discarded.
func (p *panicRecord) stop() bool { return p.stopped.Load() }

// rethrow re-raises a captured panic on the calling goroutine. It must be called
// after the worker group has been waited on.
func (p *panicRecord) rethrow(op string) {
	if !p.stopped.Load() {
		return
	}
	panic(&WorkerPanic{Op: op, Index: p.index, Value: p.value, Stack: p.stack})
}
