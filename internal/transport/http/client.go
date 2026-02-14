package http

import (
	"time"

	"github.com/go-resty/resty/v2"
)

func NewRestyClient(url string) *resty.Client {
	return resty.New().
		SetTimeout(5 * time.Second).
		SetBaseURL(url)
}
