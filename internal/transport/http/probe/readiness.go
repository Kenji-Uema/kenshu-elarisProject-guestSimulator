package probe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	"github.com/go-resty/resty/v2"
)

func ReadinessHandler(rabbitMqClient *mq.RabbitMqConnection, cottageClient *resty.Client, guestClient *resty.Client) http.HandlerFunc {
	ctx := context.Background()

	return func(w http.ResponseWriter, r *http.Request) {
		if !rabbitMqClient.IsConnectionOpen() {
			http.Error(w, "RabbitMQ connection is not open", http.StatusServiceUnavailable)
			return
		}

		if err := pingHealthz(cottageClient, ctx, "CottageManager"); err != nil {
			http.Error(w,
				err.Error(),
				http.StatusServiceUnavailable)
			return
		}

		if err := pingHealthz(guestClient, ctx, "GuestManager"); err != nil {
			http.Error(w,
				err.Error(),
				http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func pingHealthz(client *resty.Client, ctx context.Context, clientName string) error {
	resp, err := client.R().
		SetContext(ctx).
		Get("/healthz")
	if err != nil {
		return fmt.Errorf("%s healthz ping failed; err=%s", clientName, err.Error())
	}
	if resp.IsError() {
		return fmt.Errorf("%s healthz ping failed; httpCode=%d; err=%s", clientName, resp.StatusCode(), resp.Error())
	}

	return nil
}
