package mocks

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
)

type MockAuth struct {
	l   *zap.Logger
	s   storage.Storage
	usr user.User
}

func NewMock(l *zap.Logger, s storage.Storage, usr user.User) *MockAuth {
	return &MockAuth{l, s, usr}
}

// CheckAuth check cookie token and set ctx user id
func (h MockAuth) CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ht.CtxUser, h.usr)))
	})
}
