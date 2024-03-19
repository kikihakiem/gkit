package http

import (
	"context"
	"encoding/json"
	"net/http"

	gkit "github.com/kikihakiem/gkit/core"
)

// Server wraps an endpoint and implements http.Handler.
type Server[Req, Res any] struct {
	e            gkit.Endpoint[Req, Res]
	dec          gkit.EncodeDecodeFunc[*http.Request, Req]
	enc          EncodeResponseFunc[Res]
	before       []gkit.BeforeRequestFunc[*http.Request]
	after        []gkit.AfterResponseFunc[http.ResponseWriter]
	errorEncoder gkit.ErrorEncoder[http.ResponseWriter]
	finalizer    []ServerFinalizerFunc
	errorHandler gkit.ErrorHandler
}

// NewServer constructs a new HTTP server, which implements http.Handler and wraps
// the provided endpoint.
func NewServer[Req, Res any](
	e gkit.Endpoint[Req, Res],
	dec gkit.EncodeDecodeFunc[*http.Request, Req],
	enc EncodeResponseFunc[Res],
	options ...ServerOption[Req, Res],
) *Server[Req, Res] {
	s := &Server[Req, Res]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: gkit.LogErrorHandler(nil),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// NewHandlerFunc constructs a new http.HandlerFunc and wraps the provided endpoint.
func NewHandlerFunc[Req, Res any](
	e gkit.Endpoint[Req, Res],
	dec gkit.EncodeDecodeFunc[*http.Request, Req],
	enc EncodeResponseFunc[Res],
	options ...ServerOption[Req, Res],
) func(http.ResponseWriter, *http.Request) {
	return NewServer(e, dec, enc, options...).ServeHTTP
}

// ServerOption sets an optional parameter for servers.
type ServerOption[Req, Res any] gkit.Option[*Server[Req, Res]]

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore[Req, Res any](before ...gkit.BeforeRequestFunc[*http.Request]) ServerOption[Req, Res] {
	return func(s *Server[Req, Res]) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter[Req, Res any](after ...gkit.AfterResponseFunc[http.ResponseWriter]) ServerOption[Req, Res] {
	return func(s *Server[Req, Res]) { s.after = append(s.after, after...) }
}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder[Req, Res any](ee gkit.ErrorEncoder[http.ResponseWriter]) ServerOption[Req, Res] {
	return func(s *Server[Req, Res]) { s.errorEncoder = ee }
}

// ServerErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom ServerErrorEncoder or ServerFinalizer, both of which have access to
// the context.
func ServerErrorHandler[Req, Res any](errorHandler gkit.ErrorHandler) ServerOption[Req, Res] {
	return func(s *Server[Req, Res]) { s.errorHandler = errorHandler }
}

// ServerFinalizer is executed at the end of every HTTP request.
// By default, no finalizer is registered.
func ServerFinalizer[Req, Res any](f ...ServerFinalizerFunc) ServerOption[Req, Res] {
	return func(s *Server[Req, Res]) { s.finalizer = append(s.finalizer, f...) }
}

// ServeHTTP implements http.Handler.
func (s Server[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if len(s.finalizer) > 0 {
		iw := &interceptingWriter{w, http.StatusOK, 0}
		defer func() {
			ctx = context.WithValue(ctx, ContextKeyResponseHeaders, iw.Header())
			ctx = context.WithValue(ctx, ContextKeyResponseSize, iw.written)
			for _, f := range s.finalizer {
				f(ctx, iw.code, r)
			}
		}()
		w = iw.reimplementInterfaces()
	}

	for _, f := range s.before {
		ctx = f(ctx, r)
	}

	request, err := s.dec(ctx, r)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		s.errorEncoder(ctx, w, err)
		return
	}

	response, err := s.e(ctx, request)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		s.errorEncoder(ctx, w, err)
		return
	}

	for _, f := range s.after {
		ctx = f(ctx, w, err)
	}

	if err := s.enc(ctx, w, response); err != nil {
		s.errorHandler.Handle(ctx, err)
		s.errorEncoder(ctx, w, err)
		return
	}
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter)

// ServerFinalizerFunc can be used to perform work at the end of an HTTP
// request, after the response has been written to the client. The principal
// intended use is for request logging. In addition to the response code
// provided in the function signature, additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
type ServerFinalizerFunc func(ctx context.Context, code int, r *http.Request)

// DecodeJSONRequest is a DecodeRequestFunc that deserialize JSON to domain object.
func DecodeJSONRequest[Req any](_ context.Context, r *http.Request) (Req, error) {
	var req Req
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return req, err
	}

	return req, nil
}

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the ResponseWriter. Many JSON-over-HTTP services can use it as
// a sensible default. TODO: If the response implements Headerer, the provided headers
// will be applied to the response. If the response implements StatusCoder, the
// provided StatusCode will be used instead of 200.
func EncodeJSONResponse[Res any](_ context.Context, w http.ResponseWriter, response Res) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if headerer, ok := any(response).(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	code := http.StatusOK
	if sc, ok := any(response).(StatusCoder); ok {
		code = sc.StatusCode()
	}

	w.WriteHeader(code)

	if code == http.StatusNoContent {
		return nil
	}

	return json.NewEncoder(w).Encode(response)
}

// DefaultErrorEncoder writes the error to the ResponseWriter, by default a
// content type of text/plain, a body of the plain text of the error, and a
// status code of 500. If the error implements Headerer, the provided headers
// will be applied to the response. If the error implements json.Marshaler, and
// the marshaling succeeds, a content type of application/json and the JSON
// encoded form of the error will be used. If the error implements StatusCoder,
// the provided StatusCode will be used instead of 500.
func DefaultErrorEncoder(_ context.Context, w http.ResponseWriter, err error) {
	contentType, body := "text/plain; charset=utf-8", []byte(err.Error())

	if marshaler, ok := err.(json.Marshaler); ok {
		if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
			contentType, body = "application/json; charset=utf-8", jsonBody
		}
	}

	w.Header().Set("Content-Type", contentType)

	if headerer, ok := err.(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	code := http.StatusInternalServerError
	if sc, ok := err.(StatusCoder); ok {
		code = sc.StatusCode()
	}

	w.WriteHeader(code)
	w.Write(body) //nolint:errcheck
}

// StatusCoder is checked by DefaultErrorEncoder. If an error value implements
// StatusCoder, the StatusCode will be used when encoding the error. By default,
// StatusInternalServerError (500) is used.
type StatusCoder interface {
	StatusCode() int
}

// Headerer is checked by DefaultErrorEncoder. If an error value implements
// Headerer, the provided headers will be applied to the response writer, after
// the Content-Type is set.
type Headerer interface {
	Headers() http.Header
}
