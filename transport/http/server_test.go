package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gkit "github.com/kikihakiem/gkit/core"
	httptransport "github.com/kikihakiem/gkit/transport/http"
)

type emptyStruct struct{}

func TestServerBadDecode(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return emptyStruct{}, nil },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, errors.New("dang") },
		func(context.Context, http.ResponseWriter, any) error { return nil },
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	resp, _ := http.Get(server.URL)
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEndpoint(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return emptyStruct{}, errors.New("dang") },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		func(context.Context, http.ResponseWriter, any) error { return nil },
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	resp, _ := http.Get(server.URL)
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEncode(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return emptyStruct{}, nil },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		func(context.Context, http.ResponseWriter, any) error { return errors.New("dang") },
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	resp, _ := http.Get(server.URL)
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerErrorEncoder(t *testing.T) {
	errTeapot := errors.New("teapot")
	code := func(err error) int {
		if errors.Is(err, errTeapot) {
			return http.StatusTeapot
		}
		return http.StatusInternalServerError
	}
	handler := httptransport.NewServer(
		func(context.Context, emptyStruct) (emptyStruct, error) { return emptyStruct{}, errTeapot },
		func(context.Context, *http.Request) (emptyStruct, error) { return emptyStruct{}, nil },
		func(context.Context, http.ResponseWriter, emptyStruct) error { return nil },
		httptransport.ServerErrorEncoder[emptyStruct, emptyStruct](func(_ context.Context, w http.ResponseWriter, err error) { w.WriteHeader(code(err)) }),
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	resp, _ := http.Get(server.URL)
	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerErrorHandler(t *testing.T) {
	errTeapot := errors.New("teapot")
	msgChan := make(chan string, 1)
	handler := httptransport.NewServer(
		func(context.Context, emptyStruct) (emptyStruct, error) { return emptyStruct{}, errTeapot },
		func(context.Context, *http.Request) (emptyStruct, error) { return emptyStruct{}, nil },
		func(context.Context, http.ResponseWriter, emptyStruct) error { return nil },
		httptransport.ServerErrorHandler[emptyStruct, emptyStruct](gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
			msgChan <- err.Error()
		})),
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	http.Get(server.URL)
	if want, have := errTeapot.Error(), <-msgChan; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestServerHappyPath(t *testing.T) {
	step, response := testServer(t)
	step()
	resp := <-response
	defer resp.Body.Close()
	buf, _ := io.ReadAll(resp.Body)
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d (%s)", want, have, buf)
	}
}

func TestMultipleServerBefore(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan emptyStruct)
	)
	handler := httptransport.NewServer(
		gkit.NopEndpoint,
		func(context.Context, *http.Request) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ emptyStruct) error {
			w.Header().Set(headerKey, headerVal)
			w.WriteHeader(statusCode)
			w.Write([]byte(responseBody))
			return nil
		},
		httptransport.ServerBefore[emptyStruct, emptyStruct](func(ctx context.Context, r *http.Request) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		httptransport.ServerBefore[emptyStruct, emptyStruct](func(ctx context.Context, r *http.Request) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerBefores are used")
			}

			close(done)
			return ctx
		}),
	)

	server := httptest.NewServer(handler)
	defer server.Close()
	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

func TestMultipleServerAfter(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan emptyStruct)
	)
	handler := httptransport.NewServer(
		gkit.NopEndpoint,
		func(context.Context, *http.Request) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ emptyStruct) error {
			w.Header().Set(headerKey, headerVal)
			w.WriteHeader(statusCode)
			w.Write([]byte(responseBody))
			return nil
		},
		httptransport.ServerAfter[emptyStruct, emptyStruct](func(ctx context.Context, _ http.ResponseWriter, _ error) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		httptransport.ServerAfter[emptyStruct, emptyStruct](func(ctx context.Context, _ http.ResponseWriter, _ error) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerAfters are used")
			}

			close(done)
			return ctx
		}),
	)

	server := httptest.NewServer(handler)
	defer server.Close()
	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

func TestServerFinalizer(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan emptyStruct)
	)
	handler := httptransport.NewServer(
		gkit.NopEndpoint,
		func(context.Context, *http.Request) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ emptyStruct) error {
			w.Header().Set(headerKey, headerVal)
			w.WriteHeader(statusCode)
			w.Write([]byte(responseBody))
			return nil
		},
		httptransport.ServerFinalizer[emptyStruct, emptyStruct](func(ctx context.Context, code int, _ *http.Request) {
			if want, have := statusCode, code; want != have {
				t.Errorf("StatusCode: want %d, have %d", want, have)
			}

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

	server := httptest.NewServer(handler)
	defer server.Close()
	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

type enhancedResponse struct {
	Foo string `json:"foo"`
}

func (e enhancedResponse) StatusCode() int      { return http.StatusPaymentRequired }
func (e enhancedResponse) Headers() http.Header { return http.Header{"X-Edward": []string{"Snowden"}} }

func TestEncodeJSONResponse(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return enhancedResponse{Foo: "bar"}, nil },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		httptransport.EncodeJSONResponse,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusPaymentRequired, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := `{"foo":"bar"}`, strings.TrimSpace(string(buf)); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

type multiHeaderResponse emptyStruct

func (_ multiHeaderResponse) Headers() http.Header {
	return http.Header{"Vary": []string{"Origin", "User-Agent"}}
}

func TestAddMultipleHeaders(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return multiHeaderResponse{}, nil },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		httptransport.EncodeJSONResponse,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]map[string]emptyStruct{"Vary": {"Origin": emptyStruct{}, "User-Agent": emptyStruct{}}}
	for k, vls := range resp.Header {
		for _, v := range vls {
			delete((expect[k]), v)
		}
		if len(expect[k]) != 0 {
			t.Errorf("Header: unexpected header %s: %v", k, expect[k])
		}
	}
}

type multiHeaderResponseError struct {
	multiHeaderResponse
	msg string
}

func (m multiHeaderResponseError) Error() string {
	return m.msg
}

func TestAddMultipleHeadersErrorEncoder(t *testing.T) {
	errStr := "oh no"
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) {
			return nil, multiHeaderResponseError{msg: errStr}
		},
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		httptransport.EncodeJSONResponse,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]map[string]emptyStruct{"Vary": {"Origin": emptyStruct{}, "User-Agent": emptyStruct{}}}
	for k, vls := range resp.Header {
		for _, v := range vls {
			delete((expect[k]), v)
		}
		if len(expect[k]) != 0 {
			t.Errorf("Header: unexpected header %s: %v", k, expect[k])
		}
	}
	if b, _ := io.ReadAll(resp.Body); errStr != string(b) {
		t.Errorf("ErrorEncoder: got: %q, expected: %q", b, errStr)
	}
}

type noContentResponse emptyStruct

func (e noContentResponse) StatusCode() int { return http.StatusNoContent }

func TestEncodeNoContent(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return noContentResponse{}, nil },
		func(context.Context, *http.Request) (interface{}, error) { return emptyStruct{}, nil },
		httptransport.EncodeJSONResponse,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusNoContent, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := 0, len(buf); want != have {
		t.Errorf("Body: want no content, have %d bytes", have)
	}
}

type enhancedError emptyStruct

func (e enhancedError) Error() string                { return "enhanced error" }
func (e enhancedError) StatusCode() int              { return http.StatusTeapot }
func (e enhancedError) MarshalJSON() ([]byte, error) { return []byte(`{"err":"enhanced"}`), nil }
func (e enhancedError) Headers() http.Header         { return http.Header{"X-Enhanced": []string{"1"}} }

func TestEnhancedError(t *testing.T) {
	handler := httptransport.NewServer(
		func(context.Context, any) (any, error) { return nil, enhancedError{} },
		func(context.Context, *http.Request) (any, error) { return emptyStruct{}, nil },
		func(_ context.Context, w http.ResponseWriter, _ any) error { return nil },
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	if want, have := "1", resp.Header.Get("X-Enhanced"); want != have {
		t.Errorf("X-Enhanced: want %q, have %q", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := `{"err":"enhanced"}`, strings.TrimSpace(string(buf)); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

type fooRequest struct {
	Foo string `json:"foo"`
}

func TestDecodeJSONRequest(t *testing.T) {
	resw := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", strings.NewReader(`{"foo": "bar"}`))
	if err != nil {
		t.Error("Failed to create request")
	}
	handler := httptransport.NewServer(
		func(ctx context.Context, request fooRequest) (any, error) {
			if want, have := "bar", request.Foo; want != have {
				t.Errorf("Expected %s got %s", want, have)
			}
			return nil, nil
		},
		httptransport.DecodeJSONRequest,
		httptransport.EncodeJSONResponse,
	)
	handler.ServeHTTP(resw, req)
}

func testServer(t *testing.T) (step func(), resp <-chan *http.Response) {
	var (
		stepch   = make(chan bool)
		endpoint = func(context.Context, emptyStruct) (emptyStruct, error) { <-stepch; return emptyStruct{}, nil }
		response = make(chan *http.Response)
		handler  = httptransport.NewServer(
			endpoint,
			func(context.Context, *http.Request) (emptyStruct, error) { return emptyStruct{}, nil },
			func(context.Context, http.ResponseWriter, emptyStruct) error { return nil },
			httptransport.ServerBefore[emptyStruct, emptyStruct](func(ctx context.Context, _ *http.Request) context.Context { return ctx }),
			httptransport.ServerAfter[emptyStruct, emptyStruct](func(ctx context.Context, _ http.ResponseWriter, _ error) context.Context { return ctx }),
		)
	)
	go func() {
		server := httptest.NewServer(handler)
		defer server.Close()
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Error(err)
			return
		}
		response <- resp
	}()
	return func() { stepch <- true }, response
}
