// Package broker consist main interface for message broker order checker
// @author Vrulin Sergey (aka Alex Versus)
package broker

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/gobroker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/rabbit"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
)

// QueueBroker define main methods for check order queue
type QueueBroker interface {
	// Run broker
	Run(ctx context.Context) error
	// Push order id for check in broker
	Push(ctx context.Context, order order.Order) error
}

// Init project message broker
func Init(log *zap.Logger, stg storage.Storage, ent *env.Env) QueueBroker {
	if ent.BrokerType == env.BrokerTypeRabbitMQ {
		return rabbit.New(log, ent, stg)
	}

	return gobroker.New(log, ent, stg)
}
