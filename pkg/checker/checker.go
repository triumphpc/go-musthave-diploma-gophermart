// Package checker implement check order status in loyal machine
// @author Vrulin Sergey (aka Alex Versus)
package checker

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"time"
)

// Repeater run get orders for check iteratively
func Repeater(ctx context.Context, input chan<- models.Order, lgr *zap.Logger, stg storage.Storage) func() error {
	return func() error {
		lgr.Info("Run Repeater")
		defer lgr.Info("Out Repeater")

		for {
			select {
			// How ofter chek in storage
			case <-time.After(5 * time.Second):
				orders, err := stg.OrdersForCheck(ctx)
				if err != nil {
					lgr.Error("Get order error", zap.Error(err))
					continue
				}

				if len(orders) == 0 {
					continue
				}

				for _, ord := range orders {
					lgr.Info("Set order to chan", zap.Reflect("order", ord))
					input <- ord
				}

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
