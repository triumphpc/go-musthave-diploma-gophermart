// Package balance get balance user
// @author Vrulin Sergey (aka Alex Versus)
package balance

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
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
	var currentUser models.User
	if token, err := r.Cookie(ht.CookieUserIDName); err == nil {
		currentUser, _ = h.s.UserByToken(r.Context(), token.Value)
	}

	if currentUser.UserID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	var response struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	response.Withdrawn = currentUser.Withdrawn
	response.Current = currentUser.Points

	body, err := json.Marshal(response)
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
