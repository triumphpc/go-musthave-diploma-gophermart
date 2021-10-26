package rabbit

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"runtime"
)

// RMQBroker run and check task for order
type RMQBroker struct {
	lgr     *zap.Logger
	ent     *env.Env
	stg     storage.Storage
	channel *amqp.Channel
	queue   amqp.Queue
}

// queueName for order check
const queueName = "orders"

// New construct
func New(logger *zap.Logger, ent *env.Env, stg storage.Storage) *RMQBroker {
	return &RMQBroker{
		lgr: logger,
		ent: ent,
		stg: stg,
	}
}

// Run broker
func (b *RMQBroker) Run(ctx context.Context) error {
	conn, err := amqp.Dial(b.ent.BrokerHost)
	if err != nil {
		return err
	}

	errChan := conn.NotifyClose(make(chan *amqp.Error, 1))
	b.channel, err = conn.Channel()
	if err != nil {
		return err
	}

	defer func() {
		b.lgr.Info("Close rabbit connection")
		defer conn.Close()
		b.channel.Close()

	}()

	if b.queue, err = b.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	msgChan, err := b.channel.Consume(
		queueName,
		"consumer",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	group, currentCtx := errgroup.WithContext(ctx)
	for i := 0; i < runtime.NumCPU(); i++ {
		bID := i

		f := func() error {
			b.lgr.Info("Start worker", zap.Int("id", bID))
			defer b.lgr.Info("Stop worker", zap.Int("id", bID))

			for {
				select {
				case msg := <-msgChan:
					if err := b.handler(msg.Body); err != nil {
						b.lgr.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					return ctx.Err()
				case err := <-errChan:
					b.lgr.Error("Error from connection", zap.Error(err))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
	}

	return group.Wait()
}

// handler task from queue
func (b *RMQBroker) handler(body []byte) error {
	var userOrder order.Order
	if err := json.Unmarshal(body, &userOrder); err != nil {
		return err
	}
	return checker.Check(b.lgr, b.ent, b.stg, userOrder)
}

// Push order id in queue
func (b *RMQBroker) Push(order order.Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return b.channel.Publish(
		"",
		b.queue.Name,
		false,
		false,
		amqp.Publishing{
			Body: body,
		})
}
