package gobroker

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"runtime"
	"time"
)

// Task for check order id
type Task = func() error

// GoBroker run and check task for order
type GoBroker struct {
	l       *zap.Logger
	tasks   chan Task
	workers []chan int
	ent     *env.Env
	stg     storage.Storage
}

// New constructor
func New(logger *zap.Logger, env *env.Env, stg storage.Storage) *GoBroker {
	return &GoBroker{
		l:       logger,
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

func (b *GoBroker) Pause(sec int) {
	for i := range b.workers {
		b.workers[i] <- sec
	}
}

// Run checker for orders
func (b *GoBroker) Run(ctx context.Context) error {
	group, currentCtx := errgroup.WithContext(ctx)

	for i := range b.workers {
		b.workers[i] = make(chan int, 1)
		wID := i
		f := func() error {
			b.l.Info("Worker start", zap.Int("id", wID))
			for {
				select {
				case timout := <-b.workers[wID]:
					b.l.Info("Worker pause", zap.Int("id", wID))
					time.Sleep(time.Duration(timout) * time.Second)

				case task := <-b.tasks:
					b.l.Info("Worker take task", zap.Int("id", wID))

					if err := task(); err != nil {
						b.l.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					b.l.Info("Worker out by context", zap.Int("id", wID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
	}
	b.l.Info("GoBroker pool ran with", zap.Int(" thread of number", len(b.workers)))

	return group.Wait()
}

// Push new check for order
func (b *GoBroker) Push(order order.Order) error {
	b.l.Info("Push order id", zap.Int("id", order.Code))
	// Add to check pool
	err := b.add(func() error {
		return checker.Check(b.l, b.ent, b.stg, order)
	})

	return err
}
