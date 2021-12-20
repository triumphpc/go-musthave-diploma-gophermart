//
// Package mq implement handler for transfer order by rabbit mq channels
//
// Vrulin Sergey (aka Alex Versus) 2021
//
package mq

import (
	"context"
	"errors"
	"github.com/streadway/amqp"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"go.uber.org/zap"
)

// Handler describe struct for broker
type Handler struct {
	// main chan
	channel *amqp.Channel
	// queue for publisher
	queue amqp.Queue
	// chan for subscribers
	msgChan <-chan amqp.Delivery
	// Logger
	lgr *zap.Logger
	// Connection
	conn *amqp.Connection
}

// queueName for order check
const queueName = "orders"

// New constructor
func New(lgr *zap.Logger, ent *env.Env) (*Handler, error) {
	p := &Handler{lgr: lgr}

	conn, err := amqp.Dial(ent.BrokerHost)
	if err != nil {
		return p, err
	}
	p.conn = conn

	errChan := conn.NotifyClose(make(chan *amqp.Error, 1))
	p.channel, err = conn.Channel()
	if err != nil {
		return p, err
	}

	// Bag for errors from rabbit
	_, cancel := context.WithCancel(context.Background())
	go func() {
		err = <-errChan
		if !errors.Is(err, context.Canceled) {
			lgr.Error("Error from connection", zap.Error(err))
			cancel()
		}
	}()

	// Make chains
	if p.queue, err = p.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return p, err
	}

	p.msgChan, err = p.channel.Consume(
		queueName,
		"consumer",
		true,
		false,
		false,
		false,
		nil,
	)

	return p, err
}

// Put message to broker
func (h *Handler) Put(body []byte) error {
	// to rabbit
	return h.channel.Publish(
		"",
		h.queue.Name,
		false,
		false,
		amqp.Publishing{
			Body: body,
		})
}

// Get chan from rabbit
func (h *Handler) Get() <-chan amqp.Delivery {
	return h.msgChan
}

// Close rabbit connections
func (h *Handler) Close() {
	h.lgr.Info("Close rabbit connection")
	h.conn.Close()
	h.channel.Close()
}
