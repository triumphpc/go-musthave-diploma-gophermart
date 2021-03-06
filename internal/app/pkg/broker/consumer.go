//
// Package broker consist main interface for message broker order checker
//
// Vrulin Sergey (aka Alex Versus) 2021
//
package broker

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
)

// Subscriber define methods for consumers
type Subscriber interface {
	// Subscribe on publisher chan
	Subscribe(ctx context.Context, input <-chan Task, workID int) error
}

// SubscriberImpl describe subscriber model
type SubscriberImpl struct {
	lgr *zap.Logger
	ent *env.Env
	stg storage.Storage
}

// NewSubscriber create new consumer
func NewSubscriber(lgr *zap.Logger, env *env.Env, stg storage.Storage) *SubscriberImpl {
	return &SubscriberImpl{
		lgr: lgr,
		ent: env,
		stg: stg,
	}
}

// Subscribe on channel
func (c *SubscriberImpl) Subscribe(ctx context.Context, input <-chan Task, workID int) error {
	// Pusher run check orders in goroutines
	c.lgr.Info("Subscribe", zap.Int("work id", workID))
	defer c.lgr.Info("Unsubscribe", zap.Int("work id", workID))

	for {
		select {
		case task := <-input:
			c.lgr.Info("Subscriber get task", zap.Int("worker id", workID))
			if err := task(ctx); err != nil {
				c.lgr.Error("Task executed with error", zap.Error(err))
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
