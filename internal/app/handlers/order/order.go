package order

import (
	"context"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/checker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	ctx context.Context
	l   *zap.Logger
	s   pg.Storage
	c   checker.Executor
}

// New constructor
func New(l *zap.Logger, s pg.Storage, c checker.Executor) *Handler {
	return &Handler{l: l, s: s, c: c}
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
	if h.s.HasOrder(userID.(int), orderCode) {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.s.PutOrder(userID.(int), orderCode); err != nil {
		if errors.Is(err, pg.ErrOrderAlreadyExist) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		h.l.Info("Put order error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}
	// push task to check
	err = h.c.Push(userID.(int), orderCode)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}
