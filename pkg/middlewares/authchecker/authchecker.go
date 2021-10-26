// Package authchecker check user auth, search in storage and set in context value
package authchecker

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	l *zap.Logger
	s storage.Storage
}

func New(l *zap.Logger, s storage.Storage) *Handler {
	return &Handler{l, s}
}

// CheckAuth check cookie token and set ctx user id
func (h Handler) CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID int
		if token, err := r.Cookie(ht.CookieUserIDName); err == nil {
			userID, _ = h.s.UserByToken(token.Value)
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ht.CtxUserIsAuth, userID)))
	})
}
