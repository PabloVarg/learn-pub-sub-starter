package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Peril server...")

	run(logger)
}

func run(logger *slog.Logger) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		logger.Error("run", "err", err.Error())
		return
	}
	defer conn.Close()
	logger.Info("connected to rabbitmq", "conn", conn.RemoteAddr().String())

	channel, err := conn.Channel()
	if err != nil {
		logger.Error("error creating channel", "err", err.Error())
		return
	}

	pubsub.PublishJSON(
		ctx,
		channel,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: true,
		},
	)
	logger.Info("sent message")

	select {
	case <-ctx.Done():
	}
}
