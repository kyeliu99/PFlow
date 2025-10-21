package mq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/rabbitmq/amqp091-go"
)

// Publisher defines a minimal interface for publishing events.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, payload any) error
}

// Consumer defines a minimal interface for subscribing to queue messages.
type Consumer interface {
	Consume(handler func(amqp091.Delivery)) error
	Close() error
}

// RabbitPublisher publishes JSON events to a RabbitMQ exchange.
type RabbitPublisher struct {
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	exchange string
}

// NewRabbitPublisher creates a publisher connecting to RabbitMQ.
func NewRabbitPublisher(url, exchange string) (*RabbitPublisher, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return &RabbitPublisher{conn: conn, channel: ch, exchange: exchange}, nil
}

// Publish serializes the payload to JSON and sends it to the exchange.
func (p *RabbitPublisher) Publish(ctx context.Context, routingKey string, payload any) error {
	if p == nil {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Close terminates the connection.
func (p *RabbitPublisher) Close() error {
	if p == nil {
		return nil
	}
	if err := p.channel.Close(); err != nil {
		log.Printf("close channel: %v", err)
	}
	return p.conn.Close()
}

// RabbitConsumer consumes messages from queue and acknowledges on success.
type RabbitConsumer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   string
}

// NewRabbitConsumer sets up queue bindings and returns a consumer.
func NewRabbitConsumer(url, exchange, queue string) (*RabbitConsumer, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	q, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.QueueBind(q.Name, "ticket.*", exchange, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return &RabbitConsumer{conn: conn, channel: ch, queue: q.Name}, nil
}

// Consume begins delivering messages to handler.
func (c *RabbitConsumer) Consume(handler func(amqp091.Delivery)) error {
	deliveries, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for msg := range deliveries {
			handler(msg)
		}
	}()
	return nil
}

// Close closes the consumer resources.
func (c *RabbitConsumer) Close() error {
	if c == nil {
		return nil
	}
	if err := c.channel.Close(); err != nil {
		log.Printf("close channel: %v", err)
	}
	return c.conn.Close()
}
