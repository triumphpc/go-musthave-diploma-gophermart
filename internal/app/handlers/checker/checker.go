package checker

import (
	"encoding/json"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// Task for check order id
type Task = func(ctx context.Context) error

// Executor describe pool methods
type Executor interface {
	// Run checker pool
	Run(ctx context.Context) error
	// Push order id for check
	Push(userID int, orderCode int) error
	// Pause set on pause all workers
	Pause(sec int)
}

// Checker run and check task for order
type Checker struct {
	l       *zap.Logger
	tasks   chan Task
	workers []chan int
	env     *env.Env
	stg     pg.Storage
}

// ErrUnknownLoyalStatus if status unknown from loyal machine
var ErrUnknownLoyalStatus = errors.New("unknown loyal status")

// New constructor
func New(logger *zap.Logger, env *env.Env, stg pg.Storage) *Checker {
	return &Checker{
		l:       logger,
		tasks:   make(chan Task, 1000),
		workers: make([]chan int, runtime.NumCPU()),
		env:     env,
		stg:     stg,
	}
}

// add task in pool
func (c *Checker) add(task Task) error {
	c.tasks <- task
	return nil
}

func (c *Checker) Pause(sec int) {
	for i := range c.workers {
		c.workers[i] <- sec
	}
}

// Run checker for orders
func (c *Checker) Run(ctx context.Context) error {
	group, currentCtx := errgroup.WithContext(ctx)

	for i := range c.workers {
		c.workers[i] = make(chan int, 1)
		wID := i
		f := func() error {
			c.l.Info("Worker start", zap.Int("id", wID))
			for {
				select {
				case timout := <-c.workers[wID]:
					c.l.Info("Worker pause", zap.Int("id", wID))
					time.Sleep(time.Duration(timout) * time.Second)

				case task := <-c.tasks:
					c.l.Info("Worker take task", zap.Int("id", wID))

					if err := task(currentCtx); err != nil {
						c.l.Error("Task executed with error", zap.Error(err))
						return err
					}

				case <-currentCtx.Done():
					c.l.Info("Worker out by context", zap.Int("id", wID))
					return ctx.Err()
				}
			}
		}
		group.Go(f)
	}
	c.l.Info("Checker pool ran with", zap.Int(" thread of number", len(c.workers)))

	return group.Wait()
}

// Push new check for order
func (c *Checker) Push(userID int, orderCode int) error {
	c.l.Info("Push order id", zap.Int("id", orderCode))
	// Add to check pool
	err := c.add(func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)

		url := c.env.AccrualSystemAddress + "/api/orders/" + strconv.Itoa(orderCode)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		// to many connects strategy
		case http.StatusTooManyRequests:
			secTimeout := resp.Header.Get("Retry-After")
			c.l.Info("To many connections. Set pause", zap.String("sec", secTimeout))
			sec, err := strconv.Atoi(secTimeout)
			if err != nil {
				return err
			}
			c.Pause(sec)

			// Run again after timeout
			c.repeatCheck(ctx, orderCode, sec, userID)

			return nil

		case http.StatusOK:
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			ord := order.LoyalOrder{}
			err = json.Unmarshal(body, &ord)
			if err != nil {
				return err
			}

			c.l.Info("Response from loyal machine", zap.Reflect("order", ord))

			// Check current status for order
			switch ord.Status {
			case order.LoyalRegistered:
				if err := c.stg.SetStatus(orderCode, order.NEW); err != nil {
					cancel()
					return err
				}
				c.l.Info("Order registered", zap.Int("order code", orderCode))
				// repeat check after 1 sec
				c.repeatCheck(ctx, orderCode, 1, userID)

			case order.LoyalInvalid:
				if err := c.stg.SetStatus(orderCode, order.INVALID); err != nil {
					cancel()
					return err
				}
				c.l.Info("Order invalid status", zap.Int("order code", orderCode))

			case order.LoyalProcessing:
				if err := c.stg.SetStatus(orderCode, order.PROCESSING); err != nil {
					cancel()
					return err
				}
				c.l.Info("Order is processing", zap.Int("order code", orderCode))
				// repeat check after 1 sec
				c.repeatCheck(ctx, orderCode, 1, userID)

			case order.LoyalProcessed:
				if err := c.stg.AddPoints(userID, ord.Accrual, orderCode); err != nil {
					cancel()
					return err
				}
				c.l.Info("Order is processed", zap.Int("order code", orderCode))
			default:
				c.l.Info("Unknown status from loyal machine")
				return ErrUnknownLoyalStatus
			}
		default:
			// Internal error
			cancel()
			return ht.ErrInternalError
		}

		return nil
	})

	return err
}

// repeatCheck repeat check in sec second
func (c *Checker) repeatCheck(ctx context.Context, orderCode int, sec int, userID int) {
	ctx, cancel := context.WithCancel(ctx)
	c.l.Info("Create repeat task", zap.Int("order code", orderCode))
	time.AfterFunc(time.Duration(sec)*time.Second, func() {
		c.l.Info("Run repeat task", zap.Int("order code", orderCode))
		err := c.Push(userID, orderCode)
		if err != nil {
			c.l.Info("Error repeat task", zap.Error(err))
			cancel()
		}
		<-ctx.Done()
	})
}
