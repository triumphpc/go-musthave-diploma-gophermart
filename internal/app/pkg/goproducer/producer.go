package goproducer

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// Producer describe publisher model
type Producer struct {
	pub broker.Producer
}

// NewProducer constructor
func NewProducer(lgr *zap.Logger, ent *env.Env, stg storage.Storage) *Producer {
	return &Producer{
		pub: broker.Producer{
			Lgr:     lgr,
			Tasks:   make(chan broker.Task, 1000),
			Workers: make([]chan int, broker.Size),
			Ent:     ent,
			Stg:     stg,
		},
	}
}

// Push task in producer
func (p *Producer) Push(task broker.Task) error {
	return p.pub.Push(task)
}

// Run checker for orders
func (p *Producer) Run(ctx context.Context, sub broker.Subscriber) error {
	return p.pub.Run(ctx, sub)
}
