// Package checker implement check order status in loyal machine
// @author Vrulin Sergey (aka Alex Versus)
package checker

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
	"time"
)

// Check order status from loyal machine
func Check(ctx context.Context, lgr *zap.Logger, ent *env.Env, stg storage.Storage, userOrder models.Order) error {
	lgr.Info("Check order", zap.Reflect("order", userOrder))

	url := ent.AccrualSystemAddress + "/api/orders/" + strconv.Itoa(userOrder.Code)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	lgr.Info("Response code", zap.Int("code", resp.StatusCode))

	switch resp.StatusCode {
	// to many connects strategy
	case http.StatusTooManyRequests:
		secTimeout := resp.Header.Get("Retry-After")
		lgr.Info("To many connections. Set pause", zap.String("sec", secTimeout))

		timeout, err := strconv.Atoi(secTimeout)
		if err != nil {
			return err
		}
		return stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, timeout, 0)

	case http.StatusOK:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		ord := models.LoyalOrder{}
		if err := json.Unmarshal(body, &ord); err != nil {
			return err
		}

		lgr.Info("Response from loyal machine", zap.Reflect("order", ord))

		// Check current status for order
		switch ord.Status {
		case models.LoyalRegistered:
			if err := stg.SetStatus(ctx, userOrder.Code, models.NEW, 1, 0); err != nil {
				return err
			}
			lgr.Info("Order registered", zap.Int("order code", userOrder.Code))

		case models.LoyalInvalid:
			if err := stg.SetStatus(ctx, userOrder.Code, models.INVALID, 0, 0); err != nil {
				return err
			}
			lgr.Info("Order invalid status", zap.Int("order code", userOrder.Code))

		case models.LoyalProcessing:
			if err := stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, 1, 0); err != nil {
				return err
			}
			lgr.Info("Order is processing", zap.Int("order code", userOrder.Code))

		case models.LoyalProcessed:
			if err := stg.AddPoints(ctx, userOrder.UserID, ord.Accrual, userOrder.Code); err != nil {
				return err
			}
			lgr.Info("Order is processed", zap.Reflect("order", ord))

		default:
			return badResponseCheck(ctx, userOrder, stg, lgr)
		}
	default:
		return badResponseCheck(ctx, userOrder, stg, lgr)
	}
	return nil
}

// badResponseCheck work with bad response from loyal machine
func badResponseCheck(ctx context.Context, userOrder models.Order, stg storage.Storage, lgr *zap.Logger) error {
	if userOrder.Attempts > 5 {
		if err := stg.SetStatus(ctx, userOrder.Code, models.INVALID, 0, 0); err != nil {
			return err
		}
		lgr.Info("Order invalid status", zap.Int("order code", userOrder.Code))
		return nil

	}
	currentTimeout := userOrder.Attempts * 60
	if err := stg.SetStatus(ctx, userOrder.Code, models.PROCESSING, currentTimeout, 0); err != nil {
		return err
	}
	return nil
}

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

// Pusher run check orders in goroutines
func Pusher(ctx context.Context, input <-chan models.Order, lgr *zap.Logger, ent *env.Env, stg storage.Storage, workID int) func() error {
	return func() error {
		lgr.Info("Run pusher", zap.Int("work id", workID))
		defer lgr.Info("Out pushed", zap.Int("work id", workID))
		for {
			select {
			// How ofter check in storage
			case ord := <-input:
				lgr.Info("Get order from chan", zap.Reflect("worker id", workID))
				if err := Check(ctx, lgr, ent, stg, ord); err != nil {
					return err
				}

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Handle task from queue
func Handle(ctx context.Context, body []byte, lgr *zap.Logger, ent *env.Env, stg storage.Storage) error {
	var userOrder models.Order
	if err := json.Unmarshal(body, &userOrder); err != nil {
		return err
	}
	return Check(ctx, lgr, ent, stg, userOrder)
}
