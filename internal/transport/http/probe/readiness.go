package probe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/go-resty/resty/v2"
)

func ReadinessHandler(rabbitMqClient *mq.RabbitMqConnection, redisClient *redisc.Redis, cottageClient *resty.Client, guestClient *resty.Client) http.HandlerFunc {
	ctx := context.Background()

	return func(w http.ResponseWriter, r *http.Request) {
		if !rabbitMqClient.IsConnectionOpen() {
			http.Error(w, "RabbitMQ connection is not open", http.StatusServiceUnavailable)
			return
		}

		if err := redisClient.Ping(r.Context()); err != nil {
			http.Error(w, fmt.Sprintf("Redis ping failed; err=%s", err.Error()), http.StatusServiceUnavailable)
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
