package gkit

import (
	"context"
)

// BeforeRequestFunc may take information from the request and put it into a
// request context.
type BeforeRequestFunc[Req any] func(ctx context.Context, req Req) context.Context

// AfterResponseFunc may take information from response and error and put it into a
// response context.
type AfterResponseFunc[Res any] func(ctx context.Context, resp Res, err error) context.Context
