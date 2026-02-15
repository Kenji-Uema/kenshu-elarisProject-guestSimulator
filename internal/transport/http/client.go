package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRestyClient(url string, port int) *resty.Client {
	return resty.New().
		SetTransport(otelhttp.NewTransport(http.DefaultTransport)).
		SetTimeout(5 * time.Second).
		SetBaseURL(fmt.Sprintf("http://%s:%d", url, port))
}
