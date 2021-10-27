// Package balance get balance user
// @author Vrulin Sergey (aka Alex Versus)
package balance

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
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

}
