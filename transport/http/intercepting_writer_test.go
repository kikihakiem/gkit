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
	// combination 1/16
	{
		t.Log("http.ResponseWriter")
		inner := struct {
			http.ResponseWriter
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 2/16
	{
		t.Log("http.ResponseWriter, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 3/16
	{
		t.Log("http.ResponseWriter, io.ReaderFrom")
		inner := struct {
			http.ResponseWriter
			io.ReaderFrom
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 4/16
	{
		t.Log("http.ResponseWriter, io.ReaderFrom, http.Pusher")
		inner := struct {
			http.ResponseWriter
			io.ReaderFrom
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 5/16
	{
		t.Log("http.ResponseWriter, http.Hijacker")
		inner := struct {
			http.ResponseWriter
			http.Hijacker
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 6/16
	{
		t.Log("http.ResponseWriter, http.Hijacker, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 7/16
	{
		t.Log("http.ResponseWriter, http.Hijacker, io.ReaderFrom")
		inner := struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 8/16
	{
		t.Log("http.ResponseWriter, http.Hijacker, io.ReaderFrom, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 9/16
	{
		t.Log("http.ResponseWriter, http.Flusher")
		inner := struct {
			http.ResponseWriter
			http.Flusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 10/16
	{
		t.Log("http.ResponseWriter, http.Flusher, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 11/16
	{
		t.Log("http.ResponseWriter, http.Flusher, io.ReaderFrom")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 12/16
	{
		t.Log("http.ResponseWriter, http.Flusher, io.ReaderFrom, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 13/16
	{
		t.Log("http.ResponseWriter, http.Flusher, http.Hijacker")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 14/16
	{
		t.Log("http.ResponseWriter, http.Flusher, http.Hijacker, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != false {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}

	// combination 15/16
	{
		t.Log("http.ResponseWriter, http.Flusher, http.Hijacker, io.ReaderFrom")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != false {
			t.Error("unexpected interface")
		}

	}

	// combination 16/16
	{
		t.Log("http.ResponseWriter, http.Flusher, http.Hijacker, io.ReaderFrom, http.Pusher")
		inner := struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
			http.Pusher
		}{}
		w := (&interceptingWriter{ResponseWriter: inner}).reimplementInterfaces()
		if _, ok := w.(http.ResponseWriter); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Flusher); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Hijacker); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(io.ReaderFrom); ok != true {
			t.Error("unexpected interface")
		}
		if _, ok := w.(http.Pusher); ok != true {
			t.Error("unexpected interface")
		}

	}
}
