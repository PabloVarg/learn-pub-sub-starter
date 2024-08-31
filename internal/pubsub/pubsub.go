package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType = int

const (
	DurableQueue = iota
	TransientQueue
)

type AckType = int

const (
	Ack = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](ctx context.Context, ch *amqp.Channel, exchange, key string, val T) error {
	message, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        message,
	})
	if err != nil {
		return err
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	queue, err := channel.QueueDeclare(
		queueName,
		simpleQueueType == DurableQueue,
		simpleQueueType == TransientQueue,
		simpleQueueType == TransientQueue,
		false,
		amqp.Table{
			"x-dead-letter-exchange": routing.ExchangeDeadLetter,
		},
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	if err := channel.QueueBind(
		queue.Name,
		key,
		exchange,
		false,
		amqp.Table{
			"x-dead-letter-exchange": routing.ExchangeDeadLetter,
		},
	); err != nil {
		return nil, amqp.Queue{}, err
	}

	return channel, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
) error {
	channel, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
	)
	if err != nil {
		return err
	}

	delivery, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for message := range delivery {
			var decodedMessage T

			if err := json.Unmarshal(message.Body, &decodedMessage); err != nil {
				continue
			}

			ackType := handler(decodedMessage)
			switch ackType {
			case Ack:
				fmt.Printf("Acknowledging message %+v\n", decodedMessage)
				message.Ack(false)
			case NackRequeue:
				fmt.Printf("Not acknowledging (requeue) message %+v\n", decodedMessage)
				message.Nack(false, true)
			case NackDiscard:
				fmt.Printf("Not acknowledging (discard) message %+v\n", decodedMessage)
				message.Nack(false, false)
			}
		}
	}()

	return nil
}
