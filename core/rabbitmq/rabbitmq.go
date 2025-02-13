package rabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   *amqp.Queue
}

func New(uri string, queueName string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// creating a durable queue to ensure data persistence.
	q, err := ch.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		// on error declare the queue
		q, err = ch.QueueDeclare(queueName, false, true, false, false, nil)
		if err != nil {
			return nil, err
		}
	}

	return &RabbitMQ{conn: conn, channel: ch, queue: &q}, nil
}

func (r *RabbitMQ) Close() error {
	err := r.conn.Close()
	if err != nil {
		return err
	}
	err = r.channel.Close()
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Publish(ctx context.Context, data []byte) error {
	// using default exchange as we only have one queue.
	err := r.channel.PublishWithContext(ctx, "", r.queue.Name, true, false, amqp.Publishing{
		Body:        data,
		ContentType: "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Subscribe() (<-chan amqp.Delivery, error) {
	consume, err := r.channel.Consume(r.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return consume, nil
}
