package orderslist

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	l *zap.Logger
	s storage.Storage
}

// New constructor
func New(l *zap.Logger, s storage.Storage) *Handler {
	return &Handler{l, s}
}

// HasAuth user
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ht.CtxUserIsAuth)

	if userID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	orders, err := h.s.Orders(userID.(int))
	if err != nil {
		h.l.Info("Internal error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(orders)
	if err != nil {
		h.l.Info("Internal error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(body)
	if err != nil {
		h.l.Info("Internal error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}

}
