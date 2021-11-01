// Package storage describe methods for project storage
// @author Sergey Vrulin (aka Alex Versus)
package storage

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/withdraw"
)

type Storage interface {
	// Register put new user in storage
	Register(ctx context.Context, user user.User) error
	// HasAuth check user auth in storage
	HasAuth(ctx context.Context, user user.User) (bool, error)
	// SetToken set token to user
	SetToken(ctx context.Context, user user.User, token string) error
	// UserByToken check token in storage and get user id
	UserByToken(ctx context.Context, token string) (user.User, error)
	// PutOrder put order in process for check status
	PutOrder(ctx context.Context, ord order.Order) error
	// SetStatus update status for order
	SetStatus(ctx context.Context, orderCode int, status int, timeout int, points int) error
	// AddPoints add points to user
	AddPoints(ctx context.Context, userID int, points int, orderCode int) error
	// Orders get all orders by user
	Orders(ctx context.Context, userID int) ([]order.Order, error)
	// OrderByCode get order by code
	OrderByCode(ctx context.Context, code int) (order.Order, error)
	// OrdersForCheck get all orders for check in loyalty machine
	OrdersForCheck(ctx context.Context) ([]order.Order, error)
	// Withdraw points from user account
	Withdraw(ctx context.Context, ord order.Order, points float64) error
	// AddWithdraw to queue
	AddWithdraw(ctx context.Context, ord order.Order, points float64) error
	// ActiveWithdrawals get list of new withdrawals
	ActiveWithdrawals(ctx context.Context) ([]withdraw.Withdraw, error)
	// WithdrawsByUserID get list of user withdrawals
	WithdrawsByUserID(ctx context.Context, userID int) ([]withdraw.Withdraw, error)
	// Close storage connect
	Close()
}
