// Package storage describe methods for project storage
// @author Sergey Vrulin (aka Alex Versus)
package storage

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
)

type Storage interface {
	// Register put new user in storage
	Register(ctx context.Context, user models.User) error
	// HasAuth check user auth in storage
	HasAuth(ctx context.Context, user models.User) (bool, error)
	// SetToken set token to user
	SetToken(ctx context.Context, user models.User, token string) error
	// UserByToken check token in storage and get user id
	UserByToken(ctx context.Context, token string) (models.User, error)
	// PutOrder put order in process for check status
	PutOrder(ctx context.Context, ord models.Order) error
	// SetStatus update status for order
	SetStatus(ctx context.Context, orderCode int, status int, timeout int, points float64) error
	// AddPoints add points to user
	AddPoints(ctx context.Context, userID int, points float64, orderCode int) error
	// Orders get all orders by user
	Orders(ctx context.Context, userID int) ([]models.Order, error)
	// OrderByCode get order by code
	OrderByCode(ctx context.Context, code int) (models.Order, error)
	// OrdersForCheck get all orders for check in loyalty machine
	OrdersForCheck(ctx context.Context) ([]models.Order, error)
	// Withdraw points from user account
	Withdraw(ctx context.Context, ord models.Order, points float64) error
	// AddWithdraw to queue
	AddWithdraw(ctx context.Context, ord models.Order, points float64) error
	// ActiveWithdrawals get list of new withdrawals
	ActiveWithdrawals(ctx context.Context) ([]models.Withdraw, error)
	// WithdrawsByUserID get list of user withdrawals
	WithdrawsByUserID(ctx context.Context, userID int) ([]models.Withdraw, error)
	// Close storage connect
	Close()
}
