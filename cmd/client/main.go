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
	logger.Info("Starting Peril client...")

	run(logger)
}

func run(logger *slog.Logger) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Error("run", "error", err.Error())
		return
	}
	defer conn.Close()
	logger.Info("connected to rabbitmq", "conn", conn.RemoteAddr().String())

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		logger.Error("error reading username", "err", err.Error())
		return
	}

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.TransientQueue,
	)
	if err != nil {
		logger.Error("error creating queue", "err", err.Error())
		return
	}
	channel, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		pubsub.TransientQueue,
	)
	if err != nil {
		logger.Error("error creating queue", "err", err.Error())
		return
	}

	game := gamelogic.NewGameState(username)
	if err := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.TransientQueue,
		handlerPause(game),
	); err != nil {
		logger.Error("error subscribing to queue", "err", err.Error())
		return
	}
	if err := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.TransientQueue,
		handlerMove(game),
	); err != nil {
		logger.Error("error subscribing to queue", "err", err.Error())
		return
	}
	for {
		gamelogic.PrintClientHelp()
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			game.CommandSpawn(words)
		case "move":
			move, err := game.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				break
			}

			pubsub.PublishJSON(
				ctx,
				channel,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
				move,
			)
		case "status":
			game.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("spamming not allowed")
		case "quit":
			gamelogic.PrintQuit()
			cancel()
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Printf("> ")
		gs.HandlePause(ps)

		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(ps gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Printf("> ")
		outcome := gs.HandleMove(ps)
		if outcome == gamelogic.MoveOutComeSafe || outcome == gamelogic.MoveOutcomeMakeWar {
			return pubsub.Ack
		}

		return pubsub.NackDiscard
	}
}
