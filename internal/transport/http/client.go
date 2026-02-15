package http

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

func NewRestyClient(url string, port int) *resty.Client {
	return resty.New().
		SetTimeout(5 * time.Second).
		SetBaseURL(fmt.Sprintf("http://%s:%d", url, port))
}
