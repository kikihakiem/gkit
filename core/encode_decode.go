package gkit

import (
	"context"
)

// EncodeDecodeFunc encodes or decodes a user-domain object from a payload.
type EncodeDecodeFunc[In, Out any] func(ctx context.Context, request In) (Out, error)

// ResponseEncoder encodes a user-domain object as a response to a request.
type ResponseEncoder[RespWriter, Res any] func(ctx context.Context, respWriter RespWriter, response Res) error

// ErrorEncoder is responsible for encoding an error.
type ErrorEncoder[RespWriter any] func(ctx context.Context, respWriter RespWriter, err error)

// NopEncoderDecoder is a EncodeDecodeFunc that returns zero value of the output and nil error.
func NopEncoderDecoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// PassThroughEncoderDecoder is a EncodeDecodeFunc that simply returns the input and nil error.
func PassThroughEncoderDecoder[In any](_ context.Context, request In) (In, error) {
	return request, nil
}

// NopResponseEncoder does nothing.
func NopResponseEncoder[Res, RespWriter any](context.Context, Res, RespWriter) error {
	return nil
}

// NopErrorEncoder does nothing.
func NopErrorEncoder[RespWriter any](context.Context, RespWriter, error) {
}
