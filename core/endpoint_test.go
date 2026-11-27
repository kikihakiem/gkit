package gkit_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	gkit "github.com/kikihakiem/gkit/core"
)

type getOrderRequest struct{ ID string }
type getOrderResponse struct{ Status string }

func getOrder(_ context.Context, req getOrderRequest) (getOrderResponse, error) {
	return getOrderResponse{Status: "order " + req.ID}, nil
}

// trace is a generic middleware. Chain infers its type parameters from the endpoint.
func trace[Req, Res any](next gkit.Endpoint[Req, Res]) gkit.Endpoint[Req, Res] {
	return func(ctx context.Context, req Req) (Res, error) {
		fmt.Println("trace: before")
		res, err := next(ctx, req)
		fmt.Println("trace: after")
		return res, err
	}
}

// requireID reads the typed request, so it needs no type assertion.
func requireID(next gkit.Endpoint[getOrderRequest, getOrderResponse]) gkit.Endpoint[getOrderRequest, getOrderResponse] {
	return func(ctx context.Context, req getOrderRequest) (getOrderResponse, error) {
		fmt.Println("requireID")
		if req.ID == "" {
			return getOrderResponse{}, errors.New("missing id")
		}
		return next(ctx, req)
	}
}

func ExampleChain() {
	endpoint := gkit.Chain(getOrder, trace, requireID)

	res, err := endpoint(context.Background(), getOrderRequest{ID: "42"})
	fmt.Println(res.Status, err)

	// Output:
	// trace: before
	// requireID
	// trace: after
	// order 42 <nil>
}

func TestChainStopsAtFailingMiddleware(t *testing.T) {
	called := false
	endpoint := gkit.Chain(func(context.Context, getOrderRequest) (getOrderResponse, error) {
		called = true
		return getOrderResponse{}, nil
	}, requireID)

	if _, err := endpoint(context.Background(), getOrderRequest{}); err == nil {
		t.Fatal("want an error for a request without an ID")
	}
	if called {
		t.Error("the endpoint ran although the middleware returned an error")
	}
}

func TestChainWithoutMiddleware(t *testing.T) {
	res, err := gkit.Chain(getOrder)(context.Background(), getOrderRequest{ID: "7"})
	if err != nil || res.Status != "order 7" {
		t.Errorf("want %q and nil, have %q and %v", "order 7", res.Status, err)
	}
}
