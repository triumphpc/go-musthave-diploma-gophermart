// Package broker consist main interface for message broker order checker
// @author Vrulin Sergey (aka Alex Versus)
package broker

import (
	"context"
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

// Publisher define main methods for check order queue
type Publisher interface {
	// Run producer and subscribe
	Run(ctx context.Context, sub Subscriber) error
	// Push order id for check in producer
	Push(order models.Order) error
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
				c.lgr.Info("Get order from chan", zap.Reflect("worker id", workID))
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
