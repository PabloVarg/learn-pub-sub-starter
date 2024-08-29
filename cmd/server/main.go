package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signal.NotifyContext(ctx, os.Interrupt)

	run(ctx, logger)
}

func run(ctx context.Context, logger *slog.Logger) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		logger.Error("run", "err", err.Error())
		return
	}
	defer conn.Close()

	logger.Info("connected to rabbitmq", "conn", conn.RemoteAddr().String())

	select {
	case <-ctx.Done():
	}
}
