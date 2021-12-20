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

// Task for producer
type Task = func(ctx context.Context) error

// Publisher define main methods for check order queue
type Publisher interface {
	// Publish task
	Publish(task Task) error
	// Channel return active publisher chan
	Channel() <-chan Task
}

// PublisherImpl describe publisher model
type PublisherImpl struct {
	lgr   *zap.Logger
	tasks chan Task
	ent   *env.Env
	stg   storage.Storage
}

// NewPublisher create new producer
func NewPublisher(lgr *zap.Logger, ent *env.Env, stg storage.Storage) *PublisherImpl {
	return &PublisherImpl{
		lgr:   lgr,
		ent:   ent,
		stg:   stg,
		tasks: make(chan Task, 1000),
	}
}

// Publish task by producer
func (p *PublisherImpl) Publish(task Task) error {
	p.tasks <- task
	return nil
}

// Channel return active publisher chan
func (p *PublisherImpl) Channel() <-chan Task {
	return p.tasks
}
