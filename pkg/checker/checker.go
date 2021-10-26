// Package checker implement check order status in loyal machine
// @author Vrulin Sergey (aka Alex Versus)
package checker

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

// Check order status from loyal machine
func Check(lgr *zap.Logger, ent *env.Env, stg storage.Storage, userOrder order.Order) error {
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

		return stg.SetStatus(userOrder.Code, order.PROCESSING, timeout, 0)

	case http.StatusOK:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		ord := order.LoyalOrder{}
		if err := json.Unmarshal(body, &ord); err != nil {
			return err
		}

		lgr.Info("Response from loyal machine", zap.Reflect("order", ord))

		// Check current status for order
		switch ord.Status {
		case order.LoyalRegistered:
			if err := stg.SetStatus(userOrder.Code, order.NEW, 1, 0); err != nil {
				return err
			}
			lgr.Info("Order registered", zap.Int("order code", userOrder.Code))

		case order.LoyalInvalid:
			if err := stg.SetStatus(userOrder.Code, order.INVALID, 0, 0); err != nil {
				return err
			}
			lgr.Info("Order invalid status", zap.Int("order code", userOrder.Code))

		case order.LoyalProcessing:
			if err := stg.SetStatus(userOrder.Code, order.PROCESSING, 1, 0); err != nil {
				return err
			}
			lgr.Info("Order is processing", zap.Int("order code", userOrder.Code))

		case order.LoyalProcessed:
			if err := stg.AddPoints(userOrder.UserID, ord.Accrual, userOrder.Code); err != nil {
				return err
			}
			lgr.Info("Order is processed", zap.Reflect("order", ord))

		default:
			lgr.Info("Unknown status from loyal machine")
			if err := stg.SetStatus(userOrder.Code, order.PROCESSING, 60, 0); err != nil {
				return err
			}
		}
	default:
		lgr.Info("Bad code loyal machine")
		if err := stg.SetStatus(userOrder.Code, order.PROCESSING, 60, 0); err != nil {
			return err
		}
	}
	return nil
}
