package gkit

import (
	"context"
)

// DecodeRequestFunc decodes a user-domain request object from a request payload.
type DecodeRequestFunc[In, Out any] func(ctx context.Context, request In) (Out, error)

// EncodeRequestFunc encodes the passed request object into the request payload.
type EncodeRequestFunc[In, Out any] func(ctx context.Context, request In) (Out, error)

// EncodeResponseFunc encodes the passed response object to the response body.
type EncodeResponseFunc[In, Out any] func(ctx context.Context, response In) (Out, error)

// DecodeResponseFunc extracts a user-domain response object from the response body.
type DecodeResponseFunc[In, Out any] func(ctx context.Context, response In) (Out, error)

// ErrorEncoder is responsible for encoding an error.
type ErrorEncoder[Out any] func(ctx context.Context, err error) Out

// NopRequestEncoder is a EncodeRequestFunc that can be used for requests that do not
// need to be decoded, and returns zero, nil.
func NopRequestEncoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// NopResponseDecoder is a DecodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns zero, nil.
func NopResponseDecoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// NopRequestDecoder is a DecodeRequestFunc that can be used for requests that do not
// need to be decoded, and simply returns zero, nil.
func NopRequestDecoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// NopResponseEncoder is a EncodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns zero, nil.
func NopResponseEncoder[In, Out any](context.Context, In) (Out, error) {
	var out Out
	return out, nil
}

// PassThroughRequestEncoder is a EncodeRequestFunc that simply returns request, nil.
func PassThroughRequestEncoder[In any](_ context.Context, request In) (In, error) {
	return request, nil
}

// PassThroughResponseDecoder is a DecodeResponseFunc that simply returns response, nil.
func PassThroughResponseDecoder[In any](_ context.Context, response In) (In, error) {
	return response, nil
}

// PassThroughRequestDecoder is a DecodeRequestFunc that simply returns request, nil.
func PassThroughRequestDecoder[In any](_ context.Context, request In) (In, error) {
	return request, nil
}

// PassThroughResponseEncoder is a EncodeResponseFunc that simply returns response, nil.
func PassThroughResponseEncoder[In any](_ context.Context, response In) (In, error) {
	return response, nil
}
