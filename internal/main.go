package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/flows"
	"github.com/Kenji-Uema/guestSimulator/internal/app/journey"
	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/log"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

const shutdownTimeout = 5 * time.Second

type guestJourneyFactoryDeps struct {
	flowConfig              config.FlowsConfig
	servicesConfig          config.ServicesConfig
	rabbitConnection        *mq.RabbitMqConnection
	communicationConsumer   config.RabbitMqConsumerConfig
	redisCache              *redisc.Cache
	clock                   port.Clock
	hourNotificationService services.HourNotificationService
}

func exitOnError(ctx context.Context, errMsg string, err error) {
	if err != nil {
		slog.ErrorContext(ctx, errMsg, "err", err)
		os.Exit(1)
	}
}

func runCleanup(ctx context.Context, cleanup []func(context.Context) error) error {
	var shutdownErr error
	for i := len(cleanup) - 1; i >= 0; i-- {
		if err := cleanup[i](ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	return shutdownErr
}

func buildGuestJourney(deps guestJourneyFactoryDeps) (*journey.GuestJourney, error) {
	state := &domain.State{}

	guestRegisterFlow, err := flows.NewGuestRegisterFlowWithState(state, deps.servicesConfig, deps.redisCache)
	if err != nil {
		return nil, err
	}

	bookingFlow, err := flows.NewBookingFlow(state, deps.servicesConfig, deps.redisCache, deps.clock)
	if err != nil {
		return nil, err
	}

	paymentFlow, err := flows.NewPaymentFlowWithState(state, deps.servicesConfig, deps.redisCache)
	if err != nil {
		return nil, err
	}

	lodgingFlow, err := flows.NewLodgingFlowWithState(state, deps.servicesConfig, deps.redisCache, deps.hourNotificationService)
	if err != nil {
		return nil, err
	}

	guestConsumer, err := mq.NewRabbitmqConsumer(deps.rabbitConnection, deps.communicationConsumer.Consume)
	if err != nil {
		return nil, err
	}

	return journey.NewGuestJourney(
		state,
		guestRegisterFlow,
		bookingFlow,
		paymentFlow,
		lodgingFlow,
		guestConsumer,
		deps.redisCache,
		deps.communicationConsumer,
	)
}

func runJourneys(ctx context.Context, concurrencyLevel int, factory func() (*journey.GuestJourney, error)) {
	finishNotification := make(chan struct{}, concurrencyLevel)

	startJourney := func() {
		go func() {
			guestJourney, err := factory()
			if err != nil {
				slog.ErrorContext(ctx, "failed to create guest guestJourney flow", "err", err)
				finishNotification <- struct{}{}
				return
			}

			if err := guestJourney.Start(ctx); err != nil {
				slog.ErrorContext(ctx, "guest guestJourney stopped with error", "err", err)
			}

			finishNotification <- struct{}{}
		}()
	}

	for i := 0; i < concurrencyLevel; i++ {
		startJourney()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-finishNotification:
			startJourney()
		}
	}
}

func main() {
	slog.SetDefault(log.NewLogger())
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cleanup := make([]func(context.Context) error, 0, 4)
	started := false
	defer func() {
		if started {
			return
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := runCleanup(shutdownCtx, cleanup); err != nil {
			slog.ErrorContext(ctx, "startup cleanup", "err", err)
		}
	}()

	configs, err := config.LoadConfigs()
	exitOnError(ctx, "failed to load configs", err)

	shutdownTelemetry, err := telemetry.Init(ctx, telemetry.Config{
		Endpoint: configs.AppConfig.Telemetry.OTLPEndpoint,
		GrpcPort: configs.AppConfig.Telemetry.OTLPGrpcPort,
		Insecure: configs.AppConfig.Telemetry.OTLPInsecure,
	}, telemetry.AppInfo{
		ServiceName: configs.AppConfig.Name.ServiceName,
		Version:     configs.AppConfig.Name.Version,
	})
	exitOnError(ctx, "failed to init telemetry", err)
	cleanup = append(cleanup, shutdownTelemetry)

	rabbitmqInfra, err := infra.NewRabbitmq(ctx, configs.RabbitMqConnConfig, configs.RabbitMqConsumersConfig.HourChange)
	exitOnError(ctx, "failed to init rabbitmq", err)
	cleanup = append(cleanup, rabbitmqInfra.ConnectionClose)

	redisInfra, err := infra.NewRedisClient(ctx, configs.RedisConfig)
	exitOnError(ctx, "failed to init redis", err)
	cleanup = append(cleanup, func(context.Context) error {
		return redisInfra.Close()
	})

	clockInfra, err := clock.NewClock(configs.ServicesConfig)
	exitOnError(ctx, "failed to init clock", err)
	cleanup = append(cleanup, func(context.Context) error {
		return clockInfra.Close()
	})

	probeServer := http.StartHTTPServer(configs.ProbeConfig, configs.ServicesConfig, rabbitmqInfra.Connection, redisInfra.Raw)
	cleanup = append(cleanup, func(context.Context) error {
		http.ShutDownHTTPServer(probeServer)
		return nil
	})

	timeEventService, err := services.NewTimeEventService(rabbitmqInfra.HourEventConsumer)
	exitOnError(ctx, "failed to create time event service", err)
	go timeEventService.Start(ctx)
	hourNotificationService := services.NewHourNotificationService(timeEventService)

	started = true
	runJourneys(ctx, configs.FlowsConfig.GuestJourney.ConcurrencyLevel, func() (*journey.GuestJourney, error) {
		return buildGuestJourney(guestJourneyFactoryDeps{
			flowConfig:              configs.FlowsConfig,
			servicesConfig:          configs.ServicesConfig,
			rabbitConnection:        rabbitmqInfra.Connection,
			communicationConsumer:   configs.RabbitMqConsumersConfig.Communication,
			redisCache:              redisInfra.Client,
			clock:                   clockInfra,
			hourNotificationService: hourNotificationService,
		})
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := runCleanup(shutdownCtx, cleanup); err != nil {
		slog.ErrorContext(ctx, "shutdown resources", "err", err)
	}
}
