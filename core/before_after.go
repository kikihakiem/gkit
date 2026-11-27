package gkit

import (
	"context"
)

// BeforeRequestFunc may take information from the request and put it into a
// request context.
type BeforeRequestFunc[Req any] func(ctx context.Context, req Req) context.Context

// AfterResponseFunc may take information from a response and put it into the
// context. It runs only after the endpoint succeeds, so it receives no error.
type AfterResponseFunc[Res any] func(ctx context.Context, resp Res) context.Context
