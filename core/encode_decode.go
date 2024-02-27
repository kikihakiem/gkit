package gkit

import (
	"context"
)

// EncodeDecodeFunc encodes or decodes a user-domain object from a payload and vice-versa.
type EncodeDecodeFunc[In, Out any] func(ctx context.Context, request In) (Out, error)

// ErrorEncoder is responsible for encoding an error.
type ErrorEncoder[Out any] func(ctx context.Context, err error) Out

// NopEncoderDecoder is a EncodeDecodeFunc that returns zero value of the input and nil error.
func NopEncoderDecoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// PassThroughEncoderDecoder is a EncodeDecodeFunc that simply returns the input and nil error.
func PassThroughEncoderDecoder[In any](_ context.Context, request In) (In, error) {
	return request, nil
}
