# gkit

[![Go Reference](https://pkg.go.dev/badge/github.com/kikihakiem/gkit/core.svg)](https://pkg.go.dev/github.com/kikihakiem/gkit/core)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go version](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)](https://go.dev/dl/)

gkit (Generic Kit) is a small toolkit for building services in Go. It follows the design of [Go kit](https://github.com/go-kit/kit): each route has a decoder, an endpoint and an encoder. gkit uses generics, so endpoints take and return your own types, and the compiler checks that a decoder, an endpoint and an encoder fit together.

## Why

With Go kit, an endpoint takes and returns `interface{}`, so it must check the request type at run time:

```go
func CreateUserEndpoint(ctx context.Context, req interface{}) (interface{}, error) {
	request, ok := req.(dto.CreateUserRequest)
	if !ok {
		return nil, fmt.Errorf("expected CreateUserRequest, got %T", req)
	}

	return doSomething(ctx, request)
}
```

With gkit, the endpoint is a normal typed function:

```go
func CreateUserEndpoint(ctx context.Context, request dto.CreateUserRequest) (dto.CreateUserResponse, error) {
	return doSomething(ctx, request)
}
```

If a route uses a decoder that returns a different type than the endpoint accepts, the code does not compile. With Go kit, the same mistake fails only when a request arrives.

## Modules

Each directory is a separate Go module, so a service downloads only the transports it uses.

| Module | Description | Docs | CI |
|---|---|---|---|
| [`core`](core) | Endpoints, endpoint middleware, and the shared function types | [![Go Reference](https://pkg.go.dev/badge/github.com/kikihakiem/gkit/core.svg)](https://pkg.go.dev/github.com/kikihakiem/gkit/core) | [![Core](https://github.com/kikihakiem/gkit/actions/workflows/core.yml/badge.svg)](https://github.com/kikihakiem/gkit/actions/workflows/core.yml) |
| [`transport/http`](transport/http) | `net/http` server and client | [![Go Reference](https://pkg.go.dev/badge/github.com/kikihakiem/gkit/transport/http.svg)](https://pkg.go.dev/github.com/kikihakiem/gkit/transport/http) | [![HTTP](https://github.com/kikihakiem/gkit/actions/workflows/transport-http.yml/badge.svg)](https://github.com/kikihakiem/gkit/actions/workflows/transport-http.yml) |
| [`transport/echo`](transport/echo) | [Echo](https://echo.labstack.com) handler | [![Go Reference](https://pkg.go.dev/badge/github.com/kikihakiem/gkit/transport/echo.svg)](https://pkg.go.dev/github.com/kikihakiem/gkit/transport/echo) | [![Echo](https://github.com/kikihakiem/gkit/actions/workflows/transport-echo.yml/badge.svg)](https://github.com/kikihakiem/gkit/actions/workflows/transport-echo.yml) |
| [`transport/jetstream`](transport/jetstream) | NATS JetStream publisher and subscriber | [![Go Reference](https://pkg.go.dev/badge/github.com/kikihakiem/gkit/transport/jetstream.svg)](https://pkg.go.dev/github.com/kikihakiem/gkit/transport/jetstream) | [![Jetstream](https://github.com/kikihakiem/gkit/actions/workflows/transport-jetstream.yml/badge.svg)](https://github.com/kikihakiem/gkit/actions/workflows/transport-jetstream.yml) |

## Installation

gkit requires Go 1.22 or later.

```sh
go get github.com/kikihakiem/gkit/core github.com/kikihakiem/gkit/transport/http
```

## Quick start

This program serves `GET /orders/{id}`. It returns the order as JSON, or a 404 for an unknown order.

```go
package main

import (
	"context"
	"log"
	"net/http"

	gkit "github.com/kikihakiem/gkit/core"
	httptransport "github.com/kikihakiem/gkit/transport/http"
)

type GetOrderRequest struct{ ID string }

type GetOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// orderNotFound implements httptransport.StatusCoder, so the default error
// encoder answers with 404 instead of 500.
type orderNotFound struct{ id string }

func (e orderNotFound) Error() string   { return "order " + e.id + " not found" }
func (e orderNotFound) StatusCode() int { return http.StatusNotFound }

// getOrder holds the business logic. It knows nothing about HTTP.
func getOrder(_ context.Context, req GetOrderRequest) (GetOrderResponse, error) {
	if req.ID != "42" {
		return GetOrderResponse{}, orderNotFound{id: req.ID}
	}
	return GetOrderResponse{ID: req.ID, Status: "shipped"}, nil
}

func decodeGetOrder(_ context.Context, r *http.Request) (GetOrderRequest, error) {
	return GetOrderRequest{ID: r.PathValue("id")}, nil
}

// logErrors is endpoint middleware. It works for any request and response type.
func logErrors[Req, Res any](next gkit.Endpoint[Req, Res]) gkit.Endpoint[Req, Res] {
	return func(ctx context.Context, req Req) (Res, error) {
		res, err := next(ctx, req)
		if err != nil {
			log.Printf("request %+v failed: %v", req, err)
		}
		return res, err
	}
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /orders/{id}", httptransport.NewServer(
		gkit.Chain(getOrder, logErrors),
		decodeGetOrder,
		httptransport.EncodeJSONResponse,
	))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

```sh
$ curl localhost:8080/orders/42
{"id":"42","status":"shipped"}
$ curl -i localhost:8080/orders/7
HTTP/1.1 404 Not Found
...
order 7 not found
```

The [example](example) directory has a longer service with an HTTP server and a JetStream subscriber.

## How a request is handled

A server runs these steps for each request:

1. **Before functions** (`ServerBefore`) copy values from the request into the context.
2. **Decoder** turns the request into your request type. An error here stops the request, for example with a 400.
3. **Endpoint middleware** (`gkit.Chain`) runs around the endpoint, for example a permission check that returns a 403.
4. **Endpoint** runs the business logic, for example validation that returns a 422.
5. **After functions** (`ServerAfter`) set response headers. They run only when the endpoint succeeds.
6. **Encoder** writes the response.

An error in steps 2 to 6 goes to the error encoder (`ServerErrorEncoder`), so one function writes every error body. Because the steps always run in this order, a bad request body gives a 400 before a permission check can give a 403.

## Middleware

Use endpoint middleware for checks that need the typed request, for example a permission check on the order ID. `gkit.Chain` takes the endpoint first, so Go infers the types, and generic middleware needs no type arguments:

```go
endpoint := gkit.Chain(svc.GetOrder, logErrors, requireLogin)
```

The first middleware runs first. A middleware that needs a parameter takes the endpoint as its first argument, for example `requirePermission(endpoint, "orders:read")`.

Use `net/http` middleware for concerns that do not need the payload, such as access logs, metrics and rate limits. A gkit `Server` is an `http.Handler`, so any `func(http.Handler) http.Handler` middleware works with it.

## Errors

`DefaultErrorEncoder` looks for these interfaces on the error, including on errors wrapped with `fmt.Errorf("...: %w", err)`:

| Interface | Effect |
|---|---|
| `StatusCoder` | Sets the status code. The default is 500. |
| `Headerer` | Adds response headers. |
| `json.Marshaler` | Writes the error as JSON. The default is the error text. |

To use your own format, pass an error encoder to every server. `gkit.ErrorEncoder` has the same argument order as Go kit's `ErrorEncoder`, so a function written for Go kit works without changes:

```go
func encodeError(ctx context.Context, err error, w http.ResponseWriter) { /* ... */ }

httptransport.NewServer(endpoint, decode, encode, httptransport.ServerErrorEncoder(encodeError))
```

## Coming from Go kit

The structure and most names are the same as in Go kit. These parts are different:

- Endpoints, decoders and encoders use your types instead of `interface{}`.
- Options have no type parameters, so one option value works for every route.
- `gkit.Chain(endpoint, middlewares...)` takes the endpoint first, so that Go can infer the types.
- gkit has transports for `net/http`, Echo and NATS JetStream. It has no gRPC transport, service discovery or circuit breaker.

## Versioning

Each module has its own version, tagged as `<module path>/vX.Y.Z`, for example `transport/http/v0.7.0`. The modules are at v0, so a minor version can contain breaking changes. The [releases](https://github.com/kikihakiem/gkit/releases) describe the changes in each version.

## License

gkit is released under the [MIT License](LICENSE).
