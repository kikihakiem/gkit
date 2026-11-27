package echo

import (
	"context"

	"github.com/labstack/echo/v4"
)

// RequestFunc may take information from an HTTP request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint.
type RequestFunc func(context.Context, echo.Context) context.Context

// ServerResponseFunc may take information from a request context and use it to
// manipulate a ResponseWriter. ServerResponseFuncs are only executed in
// servers, after invoking the endpoint but prior to writing a response.
type ServerResponseFunc func(context.Context, echo.Context) context.Context

// ClientResponseFunc may take information from an HTTP request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type ClientResponseFunc func(context.Context, echo.Context) context.Context

// SetContentType returns a ServerResponseFunc that sets the Content-Type header
// to the provided value.
func SetContentType(contentType string) ServerResponseFunc {
	return SetResponseHeader("Content-Type", contentType)
}

// SetResponseHeader returns a ServerResponseFunc that sets the given header.
func SetResponseHeader(key, val string) ServerResponseFunc {
	return func(ctx context.Context, c echo.Context) context.Context {
		c.Response().Header().Set(key, val)
		return ctx
	}
}

// SetRequestHeader returns a RequestFunc that sets the given header.
func SetRequestHeader(key, val string) RequestFunc {
	return func(ctx context.Context, c echo.Context) context.Context {
		c.Request().Header.Set(key, val)
		return ctx
	}
}

// PopulateRequestContext is a RequestFunc that populates several values into
// the context from the HTTP request. Those values may be extracted using the
// corresponding ContextKey type in this package.
func PopulateRequestContext(ctx context.Context, c echo.Context) context.Context {
	for k, v := range map[contextKey]string{
		ContextKeyRequestMethod:          c.Request().Method,
		ContextKeyRequestURI:             c.Request().RequestURI,
		ContextKeyRequestPath:            c.Request().URL.Path,
		ContextKeyRequestProto:           c.Request().Proto,
		ContextKeyRequestHost:            c.Request().Host,
		ContextKeyRequestRemoteAddr:      c.Request().RemoteAddr,
		ContextKeyRequestXForwardedFor:   c.Request().Header.Get("X-Forwarded-For"),
		ContextKeyRequestXForwardedProto: c.Request().Header.Get("X-Forwarded-Proto"),
		ContextKeyRequestReferer:         c.Request().Header.Get("Referer"),
		ContextKeyRequestUserAgent:       c.Request().Header.Get("User-Agent"),
		ContextKeyRequestXRequestID:      c.Request().Header.Get("X-Request-Id"),
		ContextKeyRequestAccept:          c.Request().Header.Get("Accept"),
	} {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

type contextKey int

const (
	// ContextKeyRequestMethod is populated in the context by
	// PopulateRequestContext. Its value is r.Method.
	ContextKeyRequestMethod contextKey = iota

	// ContextKeyRequestURI is populated in the context by
	// PopulateRequestContext. Its value is r.RequestURI.
	ContextKeyRequestURI

	// ContextKeyRequestPath is populated in the context by
	// PopulateRequestContext. Its value is r.URL.Path.
	ContextKeyRequestPath

	// ContextKeyRequestProto is populated in the context by
	// PopulateRequestContext. Its value is r.Proto.
	ContextKeyRequestProto

	// ContextKeyRequestHost is populated in the context by
	// PopulateRequestContext. Its value is r.Host.
	ContextKeyRequestHost

	// ContextKeyRequestRemoteAddr is populated in the context by
	// PopulateRequestContext. Its value is r.RemoteAddr.
	ContextKeyRequestRemoteAddr

	// ContextKeyRequestXForwardedFor is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-For").
	ContextKeyRequestXForwardedFor

	// ContextKeyRequestXForwardedProto is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-Proto").
	ContextKeyRequestXForwardedProto

	// ContextKeyRequestReferer is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Referer").
	ContextKeyRequestReferer

	// ContextKeyRequestUserAgent is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("User-Agent").
	ContextKeyRequestUserAgent

	// ContextKeyRequestXRequestID is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Request-Id").
	ContextKeyRequestXRequestID

	// ContextKeyRequestAccept is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Accept").
	ContextKeyRequestAccept

	// ContextKeyResponseHeaders is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type http.Header, and
	// is captured only once the entire response has been written.
	ContextKeyResponseHeaders

	// ContextKeyResponseSize is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type int64.
	ContextKeyResponseSize
)
