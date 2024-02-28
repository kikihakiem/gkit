//go:build unit

package http

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
)

type versatileWriter struct {
	http.ResponseWriter
	hijackCalled   bool
	readFromCalled bool
	pushCalled     bool
	flushCalled    bool
}

func (v *versatileWriter) Flush() { v.flushCalled = true }
func (v *versatileWriter) Push(target string, opts *http.PushOptions) error {
	v.pushCalled = true
	return nil
}

func (v *versatileWriter) ReadFrom(r io.Reader) (n int64, err error) {
	v.readFromCalled = true
	return 0, nil
}

func (v *versatileWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	v.hijackCalled = true
	return nil, nil, nil
}

func TestInterceptingWriter_passthroughs(t *testing.T) {
	w := &versatileWriter{}
	iw := (&interceptingWriter{ResponseWriter: w}).reimplementInterfaces()
	iw.(http.Flusher).Flush()
	iw.(http.Pusher).Push("", nil)
	iw.(http.Hijacker).Hijack()
	iw.(io.ReaderFrom).ReadFrom(nil)

	if !w.flushCalled {
		t.Error("Flush not called")
	}
	if !w.pushCalled {
		t.Error("Push not called")
	}
	if !w.hijackCalled {
		t.Error("Hijack not called")
	}
	if !w.readFromCalled {
		t.Error("ReadFrom not called")
	}
}

// TestInterceptingWriter_reimplementInterfaces is also derived from
// https://github.com/felixge/httpsnoop, like interceptingWriter.
func TestInterceptingWriter_reimplementInterfaces(t *testing.T) {
	doTest := func(t *testing.T, name string, inner http.ResponseWriter, flusher, hijacker, readerFrom, pusher bool) {
		t.Run(name, func(t *testing.T) {
			w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
			if _, ok := w.(http.ResponseWriter); ok != true {
				t.Error("unexpected interface")
			}
			if _, ok := w.(http.Flusher); ok != flusher {
				t.Error("unexpected interface")
			}
			if _, ok := w.(http.Hijacker); ok != hijacker {
				t.Error("unexpected interface")
			}
			if _, ok := w.(io.ReaderFrom); ok != readerFrom {
				t.Error("unexpected interface")
			}
			if _, ok := w.(http.Pusher); ok != pusher {
				t.Error("unexpected interface")
			}
		})
	}

	// combination 1/16
	doTest(t, "http.ResponseWriter", struct {
		http.ResponseWriter
	}{}, false, false, false, false)

	// combination 2/16
	doTest(t, "http.ResponseWriter, http.Pusher", struct {
		http.ResponseWriter
		http.Pusher
	}{}, false, false, false, true)

	// combination 3/16
	doTest(t, "http.ResponseWriter, io.ReaderFrom", struct {
		http.ResponseWriter
		io.ReaderFrom
	}{}, false, false, true, false)

	// combination 4/16
	doTest(t, "http.ResponseWriter, io.ReaderFrom, http.Pusher", struct {
		http.ResponseWriter
		io.ReaderFrom
		http.Pusher
	}{}, false, false, true, true)

	// combination 5/16
	doTest(t, "http.ResponseWriter, http.Hijacker", struct {
		http.ResponseWriter
		http.Hijacker
	}{}, false, true, false, false)

	// combination 6/16
	doTest(t, "http.ResponseWriter, http.Hijacker, http.Pusher", struct {
		http.ResponseWriter
		http.Hijacker
		http.Pusher
	}{}, false, true, false, true)

	// combination 7/16
	doTest(t, "http.ResponseWriter, http.Hijacker, io.ReaderFrom", struct {
		http.ResponseWriter
		http.Hijacker
		io.ReaderFrom
	}{}, false, true, true, false)

	// combination 8/16
	doTest(t, "http.ResponseWriter, http.Hijacker, io.ReaderFrom, http.Pusher", struct {
		http.ResponseWriter
		http.Hijacker
		io.ReaderFrom
		http.Pusher
	}{}, false, true, true, true)

	// combination 9/16
	doTest(t, "http.ResponseWriter, http.Flusher", struct {
		http.ResponseWriter
		http.Flusher
	}{}, true, false, false, false)

	// combination 10/16
	doTest(t, "http.ResponseWriter, http.Flusher, http.Pusher", struct {
		http.ResponseWriter
		http.Flusher
		http.Pusher
	}{}, true, false, false, true)

	// combination 11/16
	doTest(t, "http.ResponseWriter, http.Flusher, io.ReaderFrom", struct {
		http.ResponseWriter
		http.Flusher
		io.ReaderFrom
	}{}, true, false, true, false)

	// combination 12/16
	doTest(t, "http.ResponseWriter, http.Flusher, io.ReaderFrom, http.Pusher", struct {
		http.ResponseWriter
		http.Flusher
		io.ReaderFrom
		http.Pusher
	}{}, true, false, true, true)

	// combination 13/16
	doTest(t, "http.ResponseWriter, http.Flusher, http.Hijacker", struct {
		http.ResponseWriter
		http.Flusher
		http.Hijacker
	}{}, true, true, false, false)

	// combination 14/16
	doTest(t, "http.ResponseWriter, http.Flusher, http.Hijacker, http.Pusher", struct {
		http.ResponseWriter
		http.Flusher
		http.Hijacker
		http.Pusher
	}{}, true, true, false, true)

	// combination 15/16
	doTest(t, "http.ResponseWriter, http.Flusher, http.Hijacker, io.ReaderFrom", struct {
		http.ResponseWriter
		http.Flusher
		http.Hijacker
		io.ReaderFrom
	}{}, true, true, true, false)

	// combination 16/16
	doTest(t, "http.ResponseWriter, http.Flusher, http.Hijacker, io.ReaderFrom, http.Pusher", struct {
		http.ResponseWriter
		http.Flusher
		http.Hijacker
		io.ReaderFrom
		http.Pusher
	}{}, true, true, true, true)
}
