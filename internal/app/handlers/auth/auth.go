// Package auth implement handler logic for user auth handler
// @author Sergey Vrulin (aka Alex Versus) 2021
package auth

import (
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

// ErrAuthIncorrect incorrect user or pass
var ErrAuthIncorrect = errors.New("auth incorrect")

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
	// Check auth
	if !h.s.HasAuth(*usr) {
		http.Error(w, ErrAuthIncorrect.Error(), http.StatusUnauthorized)
		return
	}
	// HasAuth user
	token := ht.AuthUser(w)
	err := h.s.SetToken(*usr, token)

	if err != nil {
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		h.l.Info("Internal error", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
