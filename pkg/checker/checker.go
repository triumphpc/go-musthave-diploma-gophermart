//
// Package checker implement check order status in loyal machine
//
// author Vrulin Sergey (aka Alex Versus) 2021
//
package checker

import (
	"context"
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/mq"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// Controller implement logic for check order statuses
type Controller interface {
	// Repeater describe logic for repeat check orders status
	Repeater(ctx context.Context, pub broker.Publisher) error
	// Check describe logic for check order status
	Check(ctx context.Context, usrOrd models.Order) error
	// PrepareTask describe mapping task for publisher
	PrepareTask(ctx context.Context, ord models.Order) broker.Task
	// RunListeners run listeners for producer
	RunListeners(ctx context.Context, pub broker.Publisher) error
}

// Checker implement controller interface
type Checker struct {
	lgr *zap.Logger
	ent *env.Env
	stg storage.Storage
	mq  mq.Handler
}

// New constructor for checker struct
func New(lgr *zap.Logger, ent *env.Env, stg storage.Storage) *Checker {
	chr := &Checker{
		lgr: lgr,
		ent: ent,
		stg: stg,
	}

	if ent.BrokerType == env.BrokerTypeRabbitMQ {
		rab, err := mq.New(lgr, ent)
		if err != nil {
			lgr.Fatal("Can't init rabbit handler")
		}
		chr.mq = *rab
	}

	return chr
}

// RunListeners run listeners for publisher
func (c *Checker) RunListeners(ctx context.Context, pub broker.Publisher) error {
	defer func() {
		if c.ent.BrokerType == env.BrokerTypeRabbitMQ {
			// Close rabbit connections
			c.mq.Close()
		}
	}()

	// only for rabbit run listeners
	if c.ent.BrokerType == env.BrokerTypeRabbitMQ {
		group, currentCtx := errgroup.WithContext(ctx)

		for i := 0; i < runtime.NumCPU(); i++ {
			workID := i
			f := func() error {
				c.lgr.Info("MQ listener run", zap.Int("work id", workID))
				defer c.lgr.Info("MQ listener stop", zap.Int("work id", workID))

				for {
					select {
					case msg := <-c.mq.Get():
						// Wrapper for task subscribers
						task := func() error {
							var ord models.Order
							if err := json.Unmarshal(msg.Body, &ord); err != nil {
								return err
							}
							if err := c.Check(currentCtx, ord); err != nil {
								return err
							}
							return nil
						}
						// Publish task
						c.lgr.Info("MQ listener push task", zap.Int("work id", workID))
						if err := pub.Publish(task); err != nil {
							return err
						}
					case <-currentCtx.Done():
						return ctx.Err()
					}
				}
			}
			group.Go(f)
		}

		return group.Wait()
	}

	return nil
}

// PrepareTask prepare task for current publisher
func (c *Checker) PrepareTask(ctx context.Context, ord models.Order) broker.Task {
	return func() error {
		// if implement default broker
		if c.ent.BrokerType == env.BrokerTypeGO {
			return c.Check(ctx, ord)
		}

		// if it's rabbit mq
		if c.ent.BrokerType == env.BrokerTypeRabbitMQ {
			c.lgr.Info("Publish order in rabbit", zap.Reflect("order", ord))
			body, err := json.Marshal(ord)
			if err != nil {
				return err
			}
			// only put in rabbit
			if err = c.mq.Put(body); err != nil {
				return err
			}
		}

		return nil
	}
}

// Repeater run get orders for check iteratively
// Repeated checks
func (c *Checker) Repeater(ctx context.Context, pub broker.Publisher) error {
	c.lgr.Info("Repeater started")
	defer c.lgr.Info("Repeater stopped")

	for {
		select {
		// How ofter chek in storage
		case <-time.After(5 * time.Second):
			orders, err := c.stg.OrdersForCheck(ctx)
			if err != nil {
				c.lgr.Error("Get order error", zap.Error(err))
				continue
			}

			if len(orders) == 0 {
				continue
			}

			for _, ord := range orders {
				c.lgr.Info("Push order rto task", zap.Reflect("order", ord))
				if err = pub.Publish(c.PrepareTask(ctx, ord)); err != nil {
					return err
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Check order status from loyal machine
// Implement business logic for order status check
func (c *Checker) Check(ctx context.Context, usrOrd models.Order) error {
	c.lgr.Info("Check order status", zap.Reflect("order", usrOrd))

	url := c.ent.AccrualSystemAddress + "/api/orders/" + usrOrd.Code
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.lgr.Info("LM code", zap.Int("code", resp.StatusCode))

	switch resp.StatusCode {
	// to many connects strategy
	case http.StatusTooManyRequests:
		secTimeout := resp.Header.Get("Retry-After")
		c.lgr.Info("To many connections. Set pause", zap.String("sec", secTimeout))

		timeout, err := strconv.Atoi(secTimeout)
		if err != nil {
			return err
		}
		orderID, err := strconv.Atoi(usrOrd.Code)
		if err != nil {
			return err
		}
		return c.stg.SetStatus(ctx, orderID, models.PROCESSING, timeout, 0)

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

		orderID, err := strconv.Atoi(usrOrd.Code)
		if err != nil {
			return err
		}
		// Check current status for order
		switch ord.Status {
		case models.LoyalRegistered:
			if err := c.stg.SetStatus(ctx, orderID, models.NEW, 1, 0); err != nil {
				return err
			}
			c.lgr.Info("Order registered", zap.Int("order code", orderID))

		case models.LoyalInvalid:
			if err := c.stg.SetStatus(ctx, orderID, models.INVALID, 0, 0); err != nil {
				return err
			}
			c.lgr.Info("Order invalid status", zap.Int("order code", orderID))

		case models.LoyalProcessing:
			if err := c.stg.SetStatus(ctx, orderID, models.PROCESSING, 1, 0); err != nil {
				return err
			}
			c.lgr.Info("Order is processing", zap.Int("order code", orderID))

		case models.LoyalProcessed:
			if err := c.stg.AddPoints(ctx, usrOrd.UserID, ord.Accrual, orderID); err != nil {
				return err
			}
			c.lgr.Info("Order is processed", zap.Reflect("order", ord))

		default:
			return c.badResponseCheck(ctx, usrOrd)
		}
	default:
		return c.badResponseCheck(ctx, usrOrd)
	}
	return nil
}

// badResponseCheck work with bad response from loyal machine
func (c *Checker) badResponseCheck(ctx context.Context, userOrder models.Order) error {
	orderID, err := strconv.Atoi(userOrder.Code)
	if err != nil {
		return err
	}
	if userOrder.Attempts > 5 {
		if err := c.stg.SetStatus(ctx, orderID, models.INVALID, 0, 0); err != nil {
			return err
		}
		c.lgr.Info("Order invalid status", zap.Int("order code", orderID))
		return nil

	}
	currentTimeout := userOrder.Attempts * 60
	if err := c.stg.SetStatus(ctx, orderID, models.PROCESSING, currentTimeout, 0); err != nil {
		return err
	}
	return nil
}
