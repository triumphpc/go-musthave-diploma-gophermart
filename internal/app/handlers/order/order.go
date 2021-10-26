package order

import (
	"context"
	"database/sql"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	ctx context.Context
	lgr *zap.Logger
	stg storage.Storage
	bkr broker.QueueBroker
}

// New constructor
func New(l *zap.Logger, s storage.Storage, c broker.QueueBroker) *Handler {
	return &Handler{lgr: l, stg: s, bkr: c}
}

// Register order
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ht.CtxUserIsAuth)

	if userID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	orderCode, err := ht.IsValidOrder(r)
	if err != nil {
		if errors.Is(err, ht.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	order, err := h.stg.OrderByCode(orderCode)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			h.lgr.Info("Error in get order", zap.Error(err))
			http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if order.UserID == userID.(int) {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	order.UserID = userID.(int)
	order.Code = orderCode

	// Create order
	if err := h.stg.PutOrder(order); err != nil {
		// If someone already added code
		if errors.Is(err, pg.ErrOrderAlreadyExist) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		h.lgr.Info("Put order error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}

	// Push in broker for check
	err = h.bkr.Push(order)
	if err != nil {
		h.lgr.Info("Error handler", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}
