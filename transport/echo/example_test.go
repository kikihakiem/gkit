//go:build unit

package echo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	echotransport "github.com/kikihakiem/gkit/transport/echo"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestPopulateRequestContext(t *testing.T) {
	handler := echotransport.NewHandler(
		func(ctx context.Context, request struct{}) (response struct{}, err error) {
			assert.Equal(t, http.MethodPatch, ctx.Value(echotransport.ContextKeyRequestMethod).(string))
			assert.Equal(t, "/search", ctx.Value(echotransport.ContextKeyRequestPath).(string))
			assert.Equal(t, "/search?q=sympatico", ctx.Value(echotransport.ContextKeyRequestURI).(string))
			assert.Equal(t, "a1b2c3d4e5", ctx.Value(echotransport.ContextKeyRequestXRequestID).(string))
			return struct{}{}, nil
		},
		func(context.Context, echo.Context) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, echo.Context, struct{}) error { return nil },
		echotransport.ServerBefore(echotransport.PopulateRequestContext),
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/search?q=sympatico", nil)
	req.Header.Set("X-Request-Id", "a1b2c3d4e5")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler.Handle(c)
}
