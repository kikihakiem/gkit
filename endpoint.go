package jetstream

import (
	"context"
)

type Endpoint[Req, Res any] func(ctx context.Context, request Req) (response Res, err error)
