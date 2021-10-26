// Package registration create new registration in storage
package registration

import (
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
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

// Register new user
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	usr := &user.User{}
	if err := ht.ParseJSONReq(r, usr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Validate
	if len(usr.Login) == 0 || len(usr.Password) == 0 {
		http.Error(w, ht.ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}
	// Register new user
	if err := h.s.Register(*usr); err != nil {
		if errors.Is(err, pg.ErrLoginAlreadyExist) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.l.Info("Internal error", zap.Error(err))
		return
	}
	// HasAuth user
	token := ht.AuthUser(w)
	if err := h.s.SetToken(*usr, token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.l.Info("Internal error", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
