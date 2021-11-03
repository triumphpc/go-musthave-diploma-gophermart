package rabbit

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Producer run and check task for order
type Producer struct {
	pub broker.Producer
	// main chan
	channel *amqp.Channel
	// queue for publisher
	queue amqp.Queue
}

// queueName for order check
const queueName = "orders"

// NewProducer construct
func NewProducer(lgr *zap.Logger, ent *env.Env, stg storage.Storage) *Producer {
	return &Producer{
		pub: broker.Producer{
			Lgr:     lgr,
			Tasks:   make(chan broker.Task, 1000),
			Workers: make([]chan int, broker.Size),
			Ent:     ent,
			Stg:     stg,
		},
	}
}

// Run broker
func (p *Producer) Run(ctx context.Context, sub broker.Subscriber) error {
	conn, err := amqp.Dial(p.pub.Ent.BrokerHost)
	if err != nil {
		return err
	}

	errChan := conn.NotifyClose(make(chan *amqp.Error, 1))
	p.channel, err = conn.Channel()
	if err != nil {
		return err
	}

	defer func() {
		p.pub.Lgr.Info("Close rabbit connection")
		conn.Close()
		p.channel.Close()
	}()

	// Bag for errors from rabbit
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err = <-errChan
		p.pub.Lgr.Error("Error from connection", zap.Error(err))
		cancel()
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

	// Run readers for rabbit
	group, currentCtx := errgroup.WithContext(ctx)

	for i := range p.pub.Workers {
		workID := i
		f := func() error {
			p.pub.Lgr.Info("Rabbit worker start", zap.Int("id", workID))
			for {
				select {
				case msg := <-msgChan:
					var ord models.Order
					if err := json.Unmarshal(msg.Body, &ord); err != nil {
						return err
					}
					// Push in broker for check
					task := func(inCh chan<- models.Order) error {
						inCh <- ord
						return nil
					}
					if err := p.pub.Push(task); err != nil {
						return err
					}
				case <-currentCtx.Done():
					p.pub.Lgr.Info("Rabbit worker out by context", zap.Int("id", workID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
	}

	// Run general broker
	if err := p.pub.Run(ctx, sub); err != nil {
		return err
	}

	return group.Wait()
}

// Push task in producer
// Custom implementation for rabbit format
func (p *Producer) Push(task broker.Task) error {
	inCh := make(chan models.Order)

	go func() {
		err := func() error {
			ord := <-inCh
			p.pub.Lgr.Info("Push order in rabbit", zap.Reflect("order", ord))
			body, err := json.Marshal(ord)
			if err != nil {
				return err
			}

			// to rabbit
			return p.channel.Publish(
				"",
				p.queue.Name,
				false,
				false,
				amqp.Publishing{
					Body: body,
				})

		}()
		if err != nil {
			p.pub.Lgr.Error("in push rabbit", zap.Error(err))
		}
	}()

	err := task(inCh)
	if err != nil {
		return err
	}

	return nil
}
