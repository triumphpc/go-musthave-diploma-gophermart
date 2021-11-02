package gobroker

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"runtime"
)

// Task for check order id
type Task = func() error

// GoBroker run and check task for order
type GoBroker struct {
	lgr     *zap.Logger
	tasks   chan Task
	workers []chan int
	ent     *env.Env
	stg     storage.Storage
}

// New constructor
func New(logger *zap.Logger, env *env.Env, stg storage.Storage) *GoBroker {
	return &GoBroker{
		lgr:     logger,
		tasks:   make(chan Task, 1000),
		workers: make([]chan int, runtime.NumCPU()),
		ent:     env,
		stg:     stg,
	}
}

// add task in pool
func (b *GoBroker) add(task Task) error {
	b.tasks <- task
	return nil
}

// Run checker for orders
func (b *GoBroker) Run(ctx context.Context) error {
	group, currentCtx := errgroup.WithContext(ctx)
	// chan for check order
	inputCh := make(chan models.Order, 1000)
	defer close(inputCh)

	for i := range b.workers {
		b.workers[i] = make(chan int, 1)
		workID := i
		f := func() error {
			b.lgr.Info("Worker start", zap.Int("id", workID))
			for {
				select {
				case task := <-b.tasks:
					b.lgr.Info("Worker take task", zap.Int("id", workID))

					if err := task(); err != nil {
						b.lgr.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					b.lgr.Info("Worker out by context", zap.Int("id", workID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
		// Run workers for check
		group.Go(checker.Pusher(currentCtx, inputCh, b.lgr, b.ent, b.stg, workID))
	}
	b.lgr.Info("GoBroker pool ran with", zap.Int(" thread of number", len(b.workers)))
	// Run getter list for check
	group.Go(checker.Repeater(currentCtx, inputCh, b.lgr, b.stg))

	return group.Wait()
}

// Push new check for order
func (b *GoBroker) Push(ctx context.Context, order models.Order) error {
	b.lgr.Info("Push order id", zap.Int("id", order.Code))
	// Add to check pool
	err := b.add(func() error {
		return checker.Check(ctx, b.lgr, b.ent, b.stg, order)
	})

	return err
}
