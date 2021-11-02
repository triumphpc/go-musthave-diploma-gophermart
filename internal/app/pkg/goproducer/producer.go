package goproducer

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"runtime"
)

// Task for check order id
type Task = func() error

// Producer describe publisher model
type Producer struct {
	lgr     *zap.Logger
	tasks   chan Task
	workers []chan int
	ent     *env.Env
	stg     storage.Storage
	sub     broker.Subscriber
	ctx     context.Context
}

// NewProducer constructor
func NewProducer(logger *zap.Logger, env *env.Env, stg storage.Storage) *Producer {
	return &Producer{
		lgr:     logger,
		tasks:   make(chan Task, 1000),
		workers: make([]chan int, runtime.NumCPU()),
		ent:     env,
		stg:     stg,
	}
}

// add task in pool
func (p *Producer) add(task Task) error {
	p.tasks <- task
	return nil
}

// Run checker for orders
func (p *Producer) Run(ctx context.Context, sub broker.Subscriber) error {
	group, currentCtx := errgroup.WithContext(ctx)
	// chan for check order
	inputCh := make(chan models.Order, 1000)
	defer close(inputCh)

	// Set active subscriber
	p.sub = sub
	p.ctx = ctx

	for i := range p.workers {
		p.workers[i] = make(chan int, 1)
		workID := i
		f := func() error {
			p.lgr.Info("Worker start", zap.Int("id", workID))
			for {
				select {
				case task := <-p.tasks:
					p.lgr.Info("Worker take task", zap.Int("id", workID))

					if err := task(); err != nil {
						p.lgr.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					p.lgr.Info("Worker out by context", zap.Int("id", workID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
		// Create subscriptions
		group.Go(p.sub.Subscribe(currentCtx, inputCh, workID))
	}
	p.lgr.Info("GoBroker pool ran with", zap.Int(" thread of number", len(p.workers)))
	// Run getter list for check
	group.Go(checker.Repeater(currentCtx, inputCh, p.lgr, p.stg))

	return group.Wait()
}

// Push new check for order
func (p *Producer) Push(order models.Order) error {
	p.lgr.Info("Push order id", zap.Int("id", order.Code))
	// Add to check pool
	err := p.add(func() error {
		return p.sub.Check(p.ctx, order)
	})

	return err
}
