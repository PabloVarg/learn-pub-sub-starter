package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.DurableQueue,
	)
	if err != nil {
		logger.Error("error creating queue", "err", err.Error())
		return
	}

	for {
		gamelogic.PrintServerHelp()
		words := gamelogic.GetInput()

		switch {
		case len(words) == 0:
			continue
		case words[0] == "pause":
			pubsub.PublishJSON(
				ctx,
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			logger.Info("sent pause message")
		case words[0] == "resume":
			pubsub.PublishJSON(
				ctx,
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			logger.Info("sent resume message")
		case words[0] == "quit":
			cancel()
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
