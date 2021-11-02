package rabbit

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"runtime"
)

// Producer run and check task for order
type Producer struct {
	lgr     *zap.Logger
	ent     *env.Env
	stg     storage.Storage
	channel *amqp.Channel
	queue   amqp.Queue
	sub     broker.Subscriber
}

// queueName for order check
const queueName = "orders"

// NewProducer construct
func NewProducer(logger *zap.Logger, ent *env.Env, stg storage.Storage) *Producer {
	return &Producer{
		lgr: logger,
		ent: ent,
		stg: stg,
	}
}

// Run broker
func (p *Producer) Run(ctx context.Context, sub broker.Subscriber) error {
	conn, err := amqp.Dial(p.ent.BrokerHost)
	if err != nil {
		return err
	}

	// Set active subscriber
	p.sub = sub

	errChan := conn.NotifyClose(make(chan *amqp.Error, 1))
	p.channel, err = conn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		p.lgr.Info("Close rabbit connection")
		conn.Close()
		p.channel.Close()

	}()

	if p.queue, err = p.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	msgChan, err := p.channel.Consume(
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
	// chan for check order
	inputCh := make(chan models.Order, 1000)
	defer close(inputCh)

	for i := 0; i < runtime.NumCPU(); i++ {
		workID := i
		f := func() error {
			for {
				select {
				case msg := <-msgChan:
					if err := p.task(ctx, msg); err != nil {
						p.lgr.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					return ctx.Err()
				case err := <-errChan:
					p.lgr.Error("Error from connection", zap.Error(err))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
		// Run workers for check
		group.Go(p.sub.Subscribe(currentCtx, inputCh, workID))
	}
	// Run getter list for check
	group.Go(checker.Repeater(currentCtx, inputCh, p.lgr, p.stg))

	return group.Wait()
}

// task execute task from chan
func (p *Producer) task(ctx context.Context, msg amqp.Delivery) error {
	var userOrder models.Order
	if err := json.Unmarshal(msg.Body, &userOrder); err != nil {
		return err
	}

	if err := p.sub.Check(ctx, userOrder); err != nil {
		return err
	}
	return nil
}

// Push order id in queue
func (p *Producer) Push(order models.Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		"",
		p.queue.Name,
		false,
		false,
		amqp.Publishing{
			Body: body,
		})
}
