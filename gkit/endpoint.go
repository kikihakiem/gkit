package gkit

import (
	"context"
)

// Endpoint takes a request object and returns a response object. It is intended
// to hold the business logic.
type Endpoint[Req, Res any] func(ctx context.Context, request Req) (response Res, err error)

// NopEndpoint simply returns the response object's zero value and nil error.
func NopEndpoint[Req, Res any](context.Context, Req) (Res, error) {
	var res Res
	return res, nil
}
