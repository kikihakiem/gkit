package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	httptransport "github.com/kikihakiem/gkit/transport/http"
)

func ExamplePopulateRequestContext() {
	handler := httptransport.NewServer(
		func(ctx context.Context, request interface{}) (response interface{}, err error) {
			fmt.Println("Method", ctx.Value(httptransport.ContextKeyRequestMethod).(string))
			fmt.Println("RequestPath", ctx.Value(httptransport.ContextKeyRequestPath).(string))
			fmt.Println("RequestURI", ctx.Value(httptransport.ContextKeyRequestURI).(string))
			fmt.Println("X-Request-ID", ctx.Value(httptransport.ContextKeyRequestXRequestID).(string))
			return struct{}{}, nil
		},
		func(context.Context, *http.Request) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, http.ResponseWriter, interface{}) error { return nil },
		httptransport.ServerBefore(httptransport.PopulateRequestContext),
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/search?q=sympatico", server.URL), nil)
	req.Header.Set("X-Request-Id", "a1b2c3d4e5")
	http.DefaultClient.Do(req)

	// Output:
	// Method PATCH
	// RequestPath /search
	// RequestURI /search?q=sympatico
	// X-Request-ID a1b2c3d4e5
}
