// Package broker consist main interface for message broker order checker
// @author Vrulin Sergey (aka Alex Versus)
package broker

import (
	"context"
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
)

// Task for producer
type Task = func(chan<- models.Order) error

// Publisher define main methods for check order queue
type Publisher interface {
	// Run producer and subscribe
	Run(ctx context.Context, sub Subscriber) error
	// Push order task
	Push(task Task) error
}

// Subscriber define methods for consumers
type Subscriber interface {
	// Subscribe on channel
	Subscribe(ctx context.Context, input <-chan models.Order, workID int) func() error
	// Check current status from loyal machine
	Check(ctx context.Context, userOrder models.Order) error
}

// Consumer describe subscriber model
type Consumer struct {
	lgr *zap.Logger
	ent *env.Env
	stg storage.Storage
}

// Producer describe publisher model
type Producer struct {
	Lgr     *zap.Logger
	Tasks   chan Task
	Workers []chan int
	Ent     *env.Env
	Stg     storage.Storage
}

// Push task in producer
func (p *Producer) Push(task Task) error {
	p.Tasks <- task
	return nil
}

// Size num of workers
var Size int = runtime.NumCPU()

// Run checker for orders
func (p *Producer) Run(ctx context.Context, sub Subscriber) error {
	group, currentCtx := errgroup.WithContext(ctx)
	// chan for check order
	inCh := make(chan models.Order, 1000)
	defer close(inCh)

	for i := range p.Workers {
		p.Workers[i] = make(chan int, 1)
		workID := i
		f := func() error {
			p.Lgr.Info("Worker start", zap.Int("id", workID))
			for {
				select {
				case task := <-p.Tasks:
					p.Lgr.Info("Worker take task", zap.Int("id", workID))

					if err := task(inCh); err != nil {
						p.Lgr.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					p.Lgr.Info("Worker out by context", zap.Int("id", workID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
		// Create subscriptions
		group.Go(sub.Subscribe(currentCtx, inCh, workID))
	}
	p.Lgr.Info("GoBroker pool ran with", zap.Int(" thread of number", len(p.Workers)))
	// Run getter list for check
	group.Go(checker.Repeater(currentCtx, inCh, p.Lgr, p.Stg))

	return group.Wait()
}

// NewConsumer create new consumer
func NewConsumer(logger *zap.Logger, env *env.Env, stg storage.Storage) *Consumer {
	return &Consumer{
		lgr: logger,
		ent: env,
		stg: stg,
	}
}

// Subscribe on channel
func (c *Consumer) Subscribe(ctx context.Context, input <-chan models.Order, workID int) func() error {
	// Pusher run check orders in goroutines
	return func() error {
		c.lgr.Info("Run subscribe", zap.Int("work id", workID))
		defer c.lgr.Info("Out subscribe", zap.Int("work id", workID))
		for {
			select {
			// How ofter check in storage
			case ord := <-input:
				c.lgr.Info("Get order from chan", zap.Reflect("order", ord), zap.Reflect("worker id", workID))
				if err := c.Check(ctx, ord); err != nil {
					return err
				}

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Check order status from loyal machine
func (c *Consumer) Check(ctx context.Context, userOrder models.Order) error {
	c.lgr.Info("Check order", zap.Reflect("order", userOrder))

	url := c.ent.AccrualSystemAddress + "/api/orders/" + strconv.Itoa(userOrder.Code)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.lgr.Info("Response code", zap.Int("code", resp.StatusCode))

	switch resp.StatusCode {
	// to many connects strategy
	case http.StatusTooManyRequests:
		secTimeout := resp.Header.Get("Retry-After")
		c.lgr.Info("To many connections. Set pause", zap.String("sec", secTimeout))

		timeout, err := strconv.Atoi(secTimeout)
		if err != nil {
			return err
		}
		return c.stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, timeout, 0)

	case http.StatusOK:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		ord := models.LoyalOrder{}
		if err := json.Unmarshal(body, &ord); err != nil {
			return err
		}

		c.lgr.Info("Response from loyal machine", zap.Reflect("order", ord))

		// Check current status for order
		switch ord.Status {
		case models.LoyalRegistered:
			if err := c.stg.SetStatus(ctx, userOrder.Code, models.NEW, 1, 0); err != nil {
				return err
			}
			c.lgr.Info("Order registered", zap.Int("order code", userOrder.Code))

		case models.LoyalInvalid:
			if err := c.stg.SetStatus(ctx, userOrder.Code, models.INVALID, 0, 0); err != nil {
				return err
			}
			c.lgr.Info("Order invalid status", zap.Int("order code", userOrder.Code))

		case models.LoyalProcessing:
			if err := c.stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, 1, 0); err != nil {
				return err
			}
			c.lgr.Info("Order is processing", zap.Int("order code", userOrder.Code))

		case models.LoyalProcessed:
			if err := c.stg.AddPoints(ctx, userOrder.UserID, ord.Accrual, userOrder.Code); err != nil {
				return err
			}
			c.lgr.Info("Order is processed", zap.Reflect("order", ord))

		default:
			return c.badResponseCheck(ctx, userOrder)
		}
	default:
		return c.badResponseCheck(ctx, userOrder)
	}
	return nil
}

// badResponseCheck work with bad response from loyal machine
func (c *Consumer) badResponseCheck(ctx context.Context, userOrder models.Order) error {
	if userOrder.Attempts > 5 {
		if err := c.stg.SetStatus(ctx, userOrder.Code, models.INVALID, 0, 0); err != nil {
			return err
		}
		c.lgr.Info("Order invalid status", zap.Int("order code", userOrder.Code))
		return nil

	}
	currentTimeout := userOrder.Attempts * 60
	if err := c.stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, currentTimeout, 0); err != nil {
		return err
	}
	return nil
}
