//go:build unit

package http_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	httptransport "github.com/kikihakiem/gkit/transport/http"
)

type TestResponse struct {
	Body   io.ReadCloser
	String string
}

func TestHTTPClient(t *testing.T) {
	var (
		testbody = "testbody"
		encode   = func(context.Context, *http.Request, struct{}) error { return nil }
		decode   = func(_ context.Context, r *http.Response) (TestResponse, error) {
			buffer := make([]byte, len(testbody))
			r.Body.Read(buffer)
			return TestResponse{r.Body, string(buffer)}, nil
		}
		headers        = make(chan string, 1)
		headerKey      = "X-Foo"
		headerVal      = "abcde"
		afterHeaderKey = "X-The-Dude"
		afterHeaderVal = "Abides"
		afterVal       = ""
		afterFunc      = func(ctx context.Context, r *http.Response) context.Context {
			afterVal = r.Header.Get(afterHeaderKey)
			return ctx
		}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Get(headerKey)
		w.Header().Set(afterHeaderKey, afterHeaderVal)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testbody))
	}))

	client := httptransport.NewClient[struct{}, TestResponse](
		"GET",
		mustParse(server.URL),
		encode,
		decode,
		httptransport.ClientBefore[struct{}, TestResponse](httptransport.SetRequestHeader(headerKey, headerVal)),
		httptransport.ClientAfter[struct{}, TestResponse](afterFunc),
	)

	res, err := client.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	var have string
	select {
	case have = <-headers:
	case <-time.After(time.Millisecond):
		t.Fatalf("timeout waiting for %s", headerKey)
	}
	// Check that Request Header was successfully received
	if want := headerVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that Response header set from server was received in SetClientAfter
	if want, have := afterVal, afterHeaderVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	if want, have := testbody, res.String; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that response body was closed
	b := make([]byte, 1)
	_, err = res.Body.Read(b)
	if err == nil {
		t.Fatal("wanted error, got none")
	}
	if doNotWant, have := io.EOF, err; doNotWant == have {
		t.Errorf("do not want %q, have %q", doNotWant, have)
	}
}

func TestHTTPClientBufferedStream(t *testing.T) {
	// bodysize has a size big enought to make the resopnse.Body not an instant read
	// so if the response is cancelled it wount be all readed and the test would fail
	// The 6000 has not a particular meaning, it big enough to fulfill the usecase.
	const bodysize = 6000
	var (
		testbody = string(make([]byte, bodysize))
		encode   = func(context.Context, *http.Request, struct{}) error { return nil }
		decode   = func(_ context.Context, r *http.Response) (TestResponse, error) {
			return TestResponse{r.Body, ""}, nil
		}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testbody))
	}))

	client := httptransport.NewClient[struct{}, TestResponse](
		"GET",
		mustParse(server.URL),
		encode,
		decode,
		httptransport.BufferedStream[struct{}, TestResponse](true),
	)

	res, err := client.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	// Faking work
	time.Sleep(time.Second * 1)

	// Check that response body was NOT closed
	b := make([]byte, len(testbody))
	_, err = res.Body.Read(b)
	if want, have := io.EOF, err; have != want {
		t.Fatalf("want %q, have %q", want, have)
	}
	if want, have := testbody, string(b); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestClientFinalizer(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		responseBody = "go eat a fly ugly\n"
		done         = make(chan struct{})
		encode       = func(context.Context, *http.Request, struct{}) error { return nil }
		decode       = func(_ context.Context, r *http.Response) (TestResponse, error) {
			return TestResponse{r.Body, ""}, nil
		}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerKey, headerVal)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()

	client := httptransport.NewClient[struct{}, TestResponse](
		"GET",
		mustParse(server.URL),
		encode,
		decode,
		httptransport.ClientFinalizer[struct{}, TestResponse](func(ctx context.Context, err error) {
			responseHeader := ctx.Value(httptransport.ContextKeyResponseHeaders).(http.Header)
			if want, have := headerVal, responseHeader.Get(headerKey); want != have {
				t.Errorf("%s: want %q, have %q", headerKey, want, have)
			}

			responseSize := ctx.Value(httptransport.ContextKeyResponseSize).(int64)
			if want, have := int64(len(responseBody)), responseSize; want != have {
				t.Errorf("response size: want %d, have %d", want, have)
			}

			close(done)
		}),
	)

	_, err := client.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

func TestEncodeJSONRequest(t *testing.T) {
	var (
		header http.Header
		body   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		header = r.Header

		body = string(b)
	}))

	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := httptransport.NewClient[any](
		"POST",
		serverURL,
		httptransport.EncodeJSONRequest,
		func(context.Context, *http.Response) (any, error) { return nil, nil },
	).Endpoint()

	for _, test := range []struct {
		value any
		body  string
	}{
		{nil, "null\n"},
		{12, "12\n"},
		{1.2, "1.2\n"},
		{true, "true\n"},
		{"test", "\"test\"\n"},
		{enhancedRequest{Foo: "foo"}, "{\"foo\":\"foo\"}\n"},
	} {
		if _, err := client(context.Background(), test.value); err != nil {
			t.Error(err)
			continue
		}

		if body != test.body {
			t.Errorf("%v: actual %#v, expected %#v", test.value, body, test.body)
		}

		if _, err := client(context.Background(), enhancedRequest{Foo: "foo"}); err != nil {
			t.Fatal(err)
		}

		if _, ok := header["X-Edward"]; !ok {
			t.Fatalf("X-Edward value: actual %v, expected %v", nil, []string{"Snowden"})
		}

		if v := header.Get("X-Edward"); v != "Snowden" {
			t.Errorf("X-Edward string: actual %v, expected %v", v, "Snowden")
		}
	}
}

func TestSetClient(t *testing.T) {
	var (
		encode = func(context.Context, *http.Request, any) error { return nil }
		decode = func(_ context.Context, r *http.Response) (string, error) {
			t, err := io.ReadAll(r.Body)
			if err != nil {
				return "", err
			}
			return string(t), nil
		}
	)

	testHttpClient := httpClientFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
			Body:       io.NopCloser(bytes.NewBufferString("hello, world!")),
		}, nil
	})

	client := httptransport.NewClient[any, string](
		"GET",
		&url.URL{},
		encode,
		decode,
		httptransport.SetClient[any, string](testHttpClient),
	).Endpoint()

	resp, err := client(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp != "hello, world!" {
		t.Fatal("Expected response to be 'hello, world!' string")
	}
}

func TestNewExplicitClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%d", r.ContentLength)
	}))
	defer srv.Close()

	req := func(ctx context.Context, request any) (*http.Request, error) {
		req, _ := http.NewRequest("POST", srv.URL, strings.NewReader(request.(string)))
		return req, nil
	}

	dec := func(_ context.Context, resp *http.Response) (response any, err error) {
		buf, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		return string(buf), err
	}

	client := httptransport.NewExplicitClient(req, dec)

	request := "hello world"
	response, err := client.Endpoint()(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	if want, have := "11", response.(string); want != have {
		t.Fatalf("want %q, have %q", want, have)
	}
}

func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

type enhancedRequest struct {
	Foo string `json:"foo"`
}

func (e enhancedRequest) Headers() http.Header { return http.Header{"X-Edward": []string{"Snowden"}} }

type httpClientFunc func(req *http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}
