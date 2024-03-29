package http

import (
	"io"
	"net/http"
)

type interceptingWriter struct {
	http.ResponseWriter
	code    int
	written int64
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *interceptingWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *interceptingWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.written += int64(n)
	return n, err
}

// reimplementInterfaces returns a wrapped version of the embedded ResponseWriter
// and selectively implements the same combination of additional interfaces as
// the wrapped one. The interfaces it may implement are: http.Hijacker,
// http.Pusher, http.Flusher and io.ReaderFrom. The standard
// library is known to assert the existence of these interfaces and behaves
// differently. This implementation is derived from
// https://github.com/felixge/httpsnoop.
func (w *interceptingWriter) reimplementInterfaces() http.ResponseWriter {
	var (
		hj, i1 = w.ResponseWriter.(http.Hijacker)
		pu, i2 = w.ResponseWriter.(http.Pusher)
		fl, i3 = w.ResponseWriter.(http.Flusher)
		rf, i4 = w.ResponseWriter.(io.ReaderFrom)
	)

	switch {
	case !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
		}{w}
	case !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			io.ReaderFrom
		}{w, rf}
	case !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Flusher
		}{w, fl}
	case !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{w, fl, rf}
	case !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
		}{w, pu}
	case !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			io.ReaderFrom
		}{w, pu, rf}
	case !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
		}{w, pu, fl}
	case !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{w, pu, fl, rf}
	case i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
		}{w, hj}
	case i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{w, hj, rf}
	case i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
		}{w, hj, fl}
	case i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
			io.ReaderFrom
		}{w, hj, fl, rf}
	case i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{w, hj, pu}
	case i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{w, hj, pu, rf}
	case i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
		}{w, hj, pu, fl}
	case i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{w, hj, pu, fl, rf}
	default:
		return struct {
			http.ResponseWriter
		}{w}
	}
}
