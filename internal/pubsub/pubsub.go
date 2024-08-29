package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType = int

const (
	DurableQueue = iota
	TransientQueue
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
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	if err := channel.QueueBind(
		queue.Name,
		key,
		exchange,
		false,
		nil,
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
	handler func(T),
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

			handler(decodedMessage)
			message.Ack(false)
		}
	}()

	return nil
}
