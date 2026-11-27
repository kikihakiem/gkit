package gkit

import (
	"context"
)

// Endpoint takes a request object and returns a response object. It is intended
// to hold the business logic.
type Endpoint[Req, Res any] func(ctx context.Context, request Req) (response Res, err error)

// Middleware wraps an endpoint to add behavior, such as logging or a
// permission check. It runs after the request is decoded, so it can read the
// typed request.
type Middleware[Req, Res any] func(Endpoint[Req, Res]) Endpoint[Req, Res]

// Chain wraps e with the given middlewares. The first middleware is the
// outermost one, so it runs first. Chain takes the endpoint first, so that Go
// infers the type parameters from it and accepts generic middleware functions
// without type arguments, for example Chain(svc.GetOrder, Logging, Auth).
func Chain[Req, Res any](e Endpoint[Req, Res], middlewares ...Middleware[Req, Res]) Endpoint[Req, Res] {
	for i := len(middlewares) - 1; i >= 0; i-- {
		e = middlewares[i](e)
	}
	return e
}

// NopEndpoint simply returns the response object's zero value and nil error.
func NopEndpoint[Req, Res any](context.Context, Req) (Res, error) {
	var res Res
	return res, nil
}
