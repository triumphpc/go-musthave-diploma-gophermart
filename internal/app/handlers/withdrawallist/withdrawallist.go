// Package withdrawallist return list of current user withdrawals
package withdrawallist

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
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

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	usr := r.Context().Value(ht.CtxUser)
	currentUser, _ := usr.(user.User)

	if currentUser.UserID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	wds, err := h.s.WithdrawsByUserID(r.Context(), currentUser.UserID)

	if err != nil {
		h.l.Info("Internal error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(wds)
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
