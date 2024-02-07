package jetstream

import (
	"context"
)

type Endpoint[Req, Res any] func(ctx context.Context, request Req) (response Res, err error)

func NopEndpoint[Req, Res any](context.Context, Req) (Res, error) {
	var res Res
	return res, nil
}
