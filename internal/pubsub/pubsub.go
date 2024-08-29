package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

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
	simpleQueueType int, // an enum to represent "durable" or "transient"
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
