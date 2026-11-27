//go:build unit

package echo_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gkit "github.com/kikihakiem/gkit/core"
	echotransport "github.com/kikihakiem/gkit/transport/echo"
	"github.com/labstack/echo/v4"
)

type emptyStruct struct{}

func TestServerBadDecode(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, errors.New("dang") },
		func(context.Context, echo.Context, any) error { return nil },
	)
	rec, _ := handleWith[any, any](handlerFunc)
	if want, have := http.StatusInternalServerError, rec.Result().StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEndpoint(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return emptyStruct{}, errors.New("dang") },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context, any) error { return nil },
	)
	rec, _ := handleWith[any, any](handlerFunc)
	if want, have := http.StatusInternalServerError, rec.Result().StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEncode(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context, any) error { return errors.New("dang") },
	)
	rec, _ := handleWith[any, any](handlerFunc)
	if want, have := http.StatusInternalServerError, rec.Result().StatusCode; want != have {
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
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, emptyStruct) (emptyStruct, error) { return emptyStruct{}, errTeapot },
		func(context.Context, echo.Context) (emptyStruct, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context, emptyStruct) error { return nil },
		echotransport.ServerErrorEncoder(func(_ context.Context, c echo.Context, err error) { c.Response().WriteHeader(code(err)) }),
	)
	rec, _ := handleWith[emptyStruct, emptyStruct](handlerFunc)
	if want, have := http.StatusTeapot, rec.Result().StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerErrorHandler(t *testing.T) {
	errTeapot := errors.New("teapot")
	msgChan := make(chan string, 1)
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, emptyStruct) (emptyStruct, error) { return emptyStruct{}, errTeapot },
		func(context.Context, echo.Context) (emptyStruct, error) { return emptyStruct{}, nil },
		func(context.Context, echo.Context, emptyStruct) error { return nil },
		echotransport.ServerErrorHandler(gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
			msgChan <- err.Error()
		})),
	)
	handleWith[emptyStruct, emptyStruct](handlerFunc)
	if want, have := errTeapot.Error(), <-msgChan; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestServerHappyPath(t *testing.T) {
	step, response := testServer(t)
	step()
	resp := <-response

	if want, have := http.StatusOK, resp.Result().StatusCode; want != have {
		t.Errorf("want %d, have %d (%s)", want, have, resp.Body.String())
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
	handlerFunc := echotransport.NewHandlerFunc(
		gkit.NopEndpoint,
		func(context.Context, echo.Context) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, c echo.Context, _ emptyStruct) error {
			c.Response().Header().Set(headerKey, headerVal)
			c.Response().WriteHeader(statusCode)
			c.Response().Write([]byte(responseBody))
			return nil
		},
		echotransport.ServerBefore(func(ctx context.Context, r echo.Context) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		echotransport.ServerBefore(func(ctx context.Context, r echo.Context) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerBefores are used")
			}

			close(done)
			return ctx
		}),
	)

	handleWith[emptyStruct, emptyStruct](handlerFunc)

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
	handlerFunc := echotransport.NewHandlerFunc(
		gkit.NopEndpoint,
		func(context.Context, echo.Context) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, c echo.Context, _ emptyStruct) error {
			c.Response().Header().Set(headerKey, headerVal)
			c.Response().WriteHeader(statusCode)
			c.Response().Write([]byte(responseBody))
			return nil
		},
		echotransport.ServerAfter(func(ctx context.Context, _ echo.Context) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		echotransport.ServerAfter(func(ctx context.Context, _ echo.Context) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerAfters are used")
			}

			close(done)
			return ctx
		}),
	)

	handleWith[emptyStruct, emptyStruct](handlerFunc)

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
	handlerFunc := echotransport.NewHandlerFunc(
		gkit.NopEndpoint,
		func(context.Context, echo.Context) (emptyStruct, error) {
			return emptyStruct{}, nil
		},
		func(_ context.Context, c echo.Context, _ emptyStruct) error {
			c.Response().Header().Set(headerKey, headerVal)
			c.Response().WriteHeader(statusCode)
			c.Response().Write([]byte(responseBody))
			return nil
		},
		echotransport.ServerFinalizer(func(ctx context.Context, code int, _ echo.Context) {
			if want, have := statusCode, code; want != have {
				t.Errorf("StatusCode: want %d, have %d", want, have)
			}

			responseHeader := ctx.Value(echotransport.ContextKeyResponseHeaders).(http.Header)
			if want, have := headerVal, responseHeader.Get(headerKey); want != have {
				t.Errorf("%s: want %q, have %q", headerKey, want, have)
			}

			responseSize := ctx.Value(echotransport.ContextKeyResponseSize).(int64)
			if want, have := int64(len(responseBody)), responseSize; want != have {
				t.Errorf("response size: want %d, have %d", want, have)
			}

			close(done)
		}),
	)

	handleWith[emptyStruct, emptyStruct](handlerFunc)

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
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return enhancedResponse{Foo: "bar"}, nil },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		echotransport.EncodeJSONResponse,
	)

	rec, err := handleWith[any, any](handlerFunc)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusPaymentRequired, rec.Result().StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}

	if want, have := `{"foo":"bar"}`, strings.TrimSpace(rec.Body.String()); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

type multiHeaderResponse emptyStruct

func (_ multiHeaderResponse) Headers() http.Header {
	return http.Header{"Vary": []string{"Origin", "User-Agent"}}
}

func TestAddMultipleHeaders(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return multiHeaderResponse{}, nil },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		echotransport.EncodeJSONResponse,
	)

	rec, err := handleWith[any, any](handlerFunc)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]map[string]emptyStruct{"Vary": {"Origin": emptyStruct{}, "User-Agent": emptyStruct{}}}
	for k, vls := range rec.Header() {
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
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) {
			return nil, multiHeaderResponseError{msg: errStr}
		},
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		echotransport.EncodeJSONResponse,
	)

	rec, _ := handleWith[any, any](handlerFunc)

	expect := map[string]map[string]emptyStruct{"Vary": {"Origin": emptyStruct{}, "User-Agent": emptyStruct{}}}
	for k, vls := range rec.Header() {
		for _, v := range vls {
			delete((expect[k]), v)
		}
		if len(expect[k]) != 0 {
			t.Errorf("Header: unexpected header %s: %v", k, expect[k])
		}
	}

	if b := rec.Body.String(); errStr != string(b) {
		t.Errorf("ErrorEncoder: got: %q, expected: %q", b, errStr)
	}
}

type noContentResponse emptyStruct

func (e noContentResponse) StatusCode() int { return http.StatusNoContent }

func TestEncodeNoContent(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return noContentResponse{}, nil },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		echotransport.EncodeJSONResponse,
	)

	rec, err := handleWith[any, any](handlerFunc)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusNoContent, rec.Result().StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}

	if want, have := 0, len(rec.Body.String()); want != have {
		t.Errorf("Body: want no content, have %d bytes", have)
	}
}

type enhancedError emptyStruct

func (e enhancedError) Error() string                { return "enhanced error" }
func (e enhancedError) StatusCode() int              { return http.StatusTeapot }
func (e enhancedError) MarshalJSON() ([]byte, error) { return []byte(`{"err":"enhanced"}`), nil }
func (e enhancedError) Headers() http.Header         { return http.Header{"X-Enhanced": []string{"1"}} }

func TestEnhancedError(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(context.Context, any) (any, error) { return nil, enhancedError{} },
		func(context.Context, echo.Context) (any, error) { return emptyStruct{}, nil },
		func(_ context.Context, c echo.Context, _ any) error { return nil },
	)

	rec, _ := handleWith[any, any](handlerFunc)

	if want, have := http.StatusTeapot, rec.Result().StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	if want, have := "1", rec.Header().Get("X-Enhanced"); want != have {
		t.Errorf("X-Enhanced: want %q, have %q", want, have)
	}

	if want, have := `{"err":"enhanced"}`, strings.TrimSpace(rec.Body.String()); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

type fooRequest struct {
	FromJSONBody  string `json:"foo"`
	FromPathParam int    `param:"id"`
}

func TestDecodeJSONRequest(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(ctx context.Context, request fooRequest) (any, error) {
			if want, have := "bar", request.FromJSONBody; want != have {
				t.Errorf("Expected %s got %s", want, have)
			}
			if want, have := 123, request.FromPathParam; want != have {
				t.Errorf("Expected %d got %d", want, have)
			}
			return nil, nil
		},
		echotransport.DecodeJSONRequest,
		echotransport.EncodeJSONResponse,
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/entities/123", strings.NewReader(`{"foo": "bar"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")
	handlerFunc(c)
}

func TestDecodeJSONRequestError(t *testing.T) {
	handlerFunc := echotransport.NewHandlerFunc(
		func(ctx context.Context, request fooRequest) (any, error) { return nil, nil },
		echotransport.DecodeJSONRequest,
		echotransport.EncodeJSONResponse,
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/entities/123", strings.NewReader(`{"foo": "bar"}`))
	// intentionally leave the content-type header unset
	// req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handlerFunc(c)
	if err == nil {
		t.Fatal("expecting an error, got nil")
	}
}

func testServer(t *testing.T) (step func(), resp <-chan *httptest.ResponseRecorder) {
	var (
		stepch      = make(chan bool)
		endpoint    = func(context.Context, emptyStruct) (emptyStruct, error) { <-stepch; return emptyStruct{}, nil }
		response    = make(chan *httptest.ResponseRecorder)
		handlerFunc = echotransport.NewHandlerFunc(
			endpoint,
			func(context.Context, echo.Context) (emptyStruct, error) { return emptyStruct{}, nil },
			func(context.Context, echo.Context, emptyStruct) error { return nil },
			echotransport.ServerBefore(func(ctx context.Context, _ echo.Context) context.Context { return ctx }),
			echotransport.ServerAfter(func(ctx context.Context, _ echo.Context) context.Context { return ctx }),
		)
	)
	go func() {
		rec, err := handleWith[emptyStruct, emptyStruct](handlerFunc)
		if err != nil {
			t.Error(err)
			return
		}
		response <- rec
	}()
	return func() { stepch <- true }, response
}

func handleWith[Req, Res any](handlerFunc echo.HandlerFunc) (*httptest.ResponseRecorder, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/dummy", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return rec, handlerFunc(c)
}

func TestDefaultErrorEncoderUnwrapsErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	echotransport.DefaultErrorEncoder(context.Background(), c, fmt.Errorf("get order: %w", enhancedError{}))

	if want, have := http.StatusTeapot, rec.Code; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	if want, have := "1", rec.Header().Get("X-Enhanced"); want != have {
		t.Errorf("X-Enhanced: want %q, have %q", want, have)
	}
	if want, have := `{"err":"enhanced"}`, rec.Body.String(); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}
