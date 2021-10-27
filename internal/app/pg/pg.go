package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/migrations"
	"go.uber.org/zap"
	"time"
)

// Pg storage
type Pg struct {
	db *sql.DB
	l  *zap.Logger
}

// ErrLoginAlreadyExist if login already exist in storage
var ErrLoginAlreadyExist = errors.New("already exist")

// ErrUserNotFound if not found user by code
var ErrUserNotFound = errors.New("user not exist")

// ErrOrderAlreadyExist if found order code
var ErrOrderAlreadyExist = errors.New("order exists")

// sqlNewRecord for new record in db
const sqlNewUser = "INSERT INTO users (id, login, password) VALUES (default, $1, $2)"

// sqlGetUser check user
const sqlGetUser = "SELECT 1 FROM users WHERE login=$1 AND password=$2"

// sqlUpdateToken for set delete flag
const sqlUpdateToken = "UPDATE users SET auth_token=$1 WHERE login=$2"

// sqlCheckToken get user id by token
const sqlCheckToken = "SELECT id FROM users WHERE auth_token=$1"

// sqlNewOrder create new order
const sqlNewOrder = "INSERT INTO orders (id, user_id, code, check_status) VALUES (default, $1, $2, $3)"

// sqlUpdateStatus update status order
const sqlUpdateStatus = `
UPDATE orders SET check_status=$1, accrual=$3, repeat_at=$4, check_attempts = check_attempts + 1  
WHERE code=$2 AND is_check_done=false
`

// sqlUpdateDoneStatus set ended status
const sqlUpdateDoneStatus = "UPDATE orders SET check_status=$1, accrual=$2, is_check_done=true WHERE code=$3"

// sqlAddPoints update user points
const sqlAddPoints = "UPDATE users SET points=points+$1 WHERE id=$2"

// sqlGetOrders get all user orders
const sqlGetOrders = `
SELECT code AS number,
       CASE
           WHEN check_status = 1 THEN 'PROCESSING'
           WHEN check_status = 2 THEN 'INVALID'
           WHEN check_status = 3 THEN 'PROCESSED'
           ELSE 'NEW'
           END
            AS status,
       created_at,
       accrual
FROM orders
WHERE user_id = $1
ORDER BY id DESC
`

// sqlGetOrder get order by code
const sqlGetOrder = `
SELECT code, user_id, is_check_done, check_attempts
FROM orders
WHERE code=$1
`

// sqlGetOrdersForCheck get chunk orders for checking
const sqlGetOrdersForCheck = `
SELECT code, user_id, check_attempts
FROM orders WHERE is_check_done=false
AND repeat_at < NOW() at time zone 'utc' LIMIT 1000
`

// New New new Pg with not null fields
func New(ctx context.Context, l *zap.Logger, e *env.Env) (*Pg, error) {
	// Database init
	connect, err := sql.Open("postgres", e.DatabaseDsn)
	if err != nil {
		return nil, err
	}
	// Ping
	if err := connect.PingContext(ctx); err != nil {
		return nil, err
	}
	// Run migrations
	goose.SetBaseFS(migrations.EmbedMigrations)
	if err := goose.Up(connect, "."); err != nil {
		panic(err)
	}

	return &Pg{connect, l}, nil
}

// Close connection
func (s *Pg) Close() {
	err := s.db.Close()
	if err != nil {
		s.l.Info("Closing don't close")
	}
}

// Register register new user in storage
func (s *Pg) Register(u user.User) error {
	if _, err := s.db.Exec(sqlNewUser, u.Login, u.HexPassword()); err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == pgerrcode.UniqueViolation {
				return ErrLoginAlreadyExist
			}
			return err
		}
	}

	return nil
}

// HasAuth search user in storage
func (s *Pg) HasAuth(u user.User) bool {
	return s.rowExists(sqlGetUser, u.Login, u.HexPassword())
}

// SetToken update token to user
func (s *Pg) SetToken(u user.User, t string) error {
	_, err := s.db.Exec(sqlUpdateToken, t, u.Login)

	return err
}

// UserByToken check if token exist
func (s *Pg) UserByToken(t string) (int, error) {
	var userID int
	err := s.db.QueryRow(sqlCheckToken, t).Scan(&userID)
	if err != nil {
		return userID, ErrUserNotFound
	}

	return userID, nil
}

// PutOrder put order in storage
func (s *Pg) PutOrder(ord order.Order) error {
	if _, err := s.db.Exec(sqlNewOrder, ord.UserID, ord.Code, order.NEW); err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == pgerrcode.UniqueViolation {
				return ErrOrderAlreadyExist
			}
			return err
		}
	}

	return nil
}

// SetStatus update status to order by code
func (s *Pg) SetStatus(orderCode int, status int, timeout int, points int) error {
	// If it's ended status
	if status == order.PROCESSED || status == order.INVALID {
		_, err := s.db.Exec(sqlUpdateDoneStatus, status, points, orderCode)

		return err
	} else {
		if timeout < 1 {
			timeout = 1
		}
		repeatAt := time.Now().Add(time.Duration(timeout) * time.Second).In(time.UTC)
		_, err := s.db.Exec(sqlUpdateStatus, status, orderCode, points, repeatAt)

		return err
	}
}

// Check if exist record by query
func (s *Pg) rowExists(query string, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := s.db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false
	}

	return exists
}

// AddPoints add points to user and done check
func (s *Pg) AddPoints(userID int, points int, orderCode int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := s.SetStatus(orderCode, order.PROCESSED, 0, points); err != nil {
		return err
	}

	_, err = s.db.Exec(sqlAddPoints, points, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Orders get user orders list
func (s *Pg) Orders(userID int) ([]order.Order, error) {
	var orders []order.Order
	rows, err := s.db.Query(sqlGetOrders, userID)
	if err != nil {
		return orders, err
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	for rows.Next() {
		var userOrder order.Order
		err = rows.Scan(&userOrder.Code, &userOrder.CheckStatus, &userOrder.UploadedAt, &userOrder.Accrual)
		if err != nil {
			return orders, err
		}
		orders = append(orders, userOrder)
	}
	return orders, nil
}

// OrderByCode get order by code
func (s *Pg) OrderByCode(code int) (order.Order, error) {
	var userOrder order.Order
	err := s.db.QueryRow(sqlGetOrder, code).Scan(&userOrder.Code, &userOrder.UserID, &userOrder.IsCheckDone, &userOrder.Attempts)

	return userOrder, err
}

// OrdersForCheck get chunk for check in loyalty machine
func (s *Pg) OrdersForCheck() ([]order.Order, error) {
	var orders []order.Order
	rows, err := s.db.Query(sqlGetOrdersForCheck)
	if err != nil {
		return orders, err
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	for rows.Next() {
		var userOrder order.Order
		err = rows.Scan(&userOrder.Code, &userOrder.UserID, &userOrder.Attempts)
		if err != nil {
			return orders, err
		}
		orders = append(orders, userOrder)
	}

	return orders, nil
}
