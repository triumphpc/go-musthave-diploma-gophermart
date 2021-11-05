//
// Package broker consist main interface for message broker order checker
//
// Vrulin Sergey (aka Alex Versus) 2021
//
package broker

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
)

// Task for producer
type Task = func() error

// Publisher define main methods for check order queue
type Publisher interface {
	// Publish task
	Publish(task Task) error
	// Channel return active publisher chan
	Channel() <-chan Task
}

// Producer describe publisher model
type Producer struct {
	lgr   *zap.Logger
	tasks chan Task
	ent   *env.Env
	stg   storage.Storage
}

// NewProducer create new producer
func NewProducer(lgr *zap.Logger, ent *env.Env, stg storage.Storage) *Producer {
	return &Producer{
		lgr:   lgr,
		ent:   ent,
		stg:   stg,
		tasks: make(chan Task, 1000),
	}
}

// Publish task by producer
func (p *Producer) Publish(task Task) error {
	p.tasks <- task
	return nil
}

// Channel return active publisher chan
func (p *Producer) Channel() <-chan Task {
	return p.tasks
}
