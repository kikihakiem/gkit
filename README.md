# GKit

Stands for Generic Kit. Heavily inspired by [Go kit](https://github.com/go-kit/kit).

## Why?

Because this is better:

```go
func CreateUserEndpoint(ctx context.Context, request dto.CreateUserRequest) (dto.CreateUserResponse, error) {
	response, err := doSomething(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to do something: %w", err)
	}

	return response, nil
}

```

than this:

```go
func CreateUserEndpoint(ctx context.Context, req interface{}) (interface{}, error) {
	request, ok := req.(dto.CreateUserRequest)
	if !ok {
		return nil, fmt.Errorf("expected CreateUserRequest struct, got %+v", req)
	}

	response, err := doSomething(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to do something: %w", err)
	}

	return response, nil
}
```

Please check the [example](/example) for more examples.
