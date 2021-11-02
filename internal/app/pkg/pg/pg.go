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
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
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
const sqlCheckToken = "SELECT id, points, withdrawn FROM users WHERE auth_token=$1"

// sqlNewOrder create new order
const sqlNewOrder = "INSERT INTO orders (id, user_id, code, check_status) VALUES (default, $1, $2, $3)"

// sqlUpdateStatus update status order
const sqlUpdateStatus = `
	UPDATE orders SET check_status=$1, accrual=$3, repeat_at=$4, check_attempts = check_attempts + 1  
	WHERE code=$2 AND is_check_done=false
`

// sqlUpdateDoneStatus set ended status
const sqlUpdateDoneStatus = `
	UPDATE orders 
	SET check_status=$1, 
	accrual=$2, 
	is_check_done=true, 
	avail_for_withdraw=$4 
	WHERE code=$3
`

// sqlAddPoints update user points
const sqlAddPoints = "UPDATE users SET points=points+$1 WHERE id=$2"

// sqlSubAvailPointsInOrder update user points in order
const sqlSubAvailPointsInOrder = "UPDATE orders SET avail_for_withdraw=avail_for_withdraw-$1 WHERE id=$2"

// sqlUserSubPoints update user points
const sqlUserSubPoints = "UPDATE users SET points=points-$1, withdrawn=withdrawn+$2 WHERE id=$3"

// sqlAddWithdrawToQueue add queue
const sqlAddWithdrawToQueue = "INSERT INTO withdrawals (id, user_id, order_id, points) VALUES (default, $1, $2, $3)"

// sqlWithdrawUpdate update status to withdraw
const sqlWithdrawUpdate = `
	UPDATE withdrawals 
	SET status=1, processed_at=now()
	WHERE user_id=$1 AND order_id=$2 AND points=$3
`

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
	SELECT id, code, user_id, is_check_done, check_attempts, accrual, avail_for_withdraw
	FROM orders
	WHERE code=$1
`

// sqlGetOrdersForCheck get chunk orders for checking
const sqlGetOrdersForCheck = `
	SELECT code, user_id, check_attempts
	FROM orders WHERE is_check_done=false
	AND repeat_at < NOW() at time zone 'utc' LIMIT 1000
`

// sqlGetWithdrawals get withdrawals for withdraw
const sqlGetWithdrawals = `
	SELECT user_id, order_id, points FROM withdrawals WHERE status=0 ORDER BY processed_at DESC LIMIT 1000
`

// sqlGetWithdrawalsByUserID get list withdrawal by user id
const sqlGetWithdrawalsByUserID = `
	SELECT w.points, o.code, processed_at,
	 CASE
			   WHEN status = 1 THEN 'PROCESSED'
			   ELSE 'NEW'
			   END
				AS status
	FROM withdrawals AS w
	JOIN orders AS o ON o.id = w.order_id
	WHERE w.user_id=$1 
	ORDER BY processed_at DESC
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
func (s *Pg) Register(ctx context.Context, user models.User) error {
	if _, err := s.db.ExecContext(ctx, sqlNewUser, user.Login, user.HexPassword()); err != nil {
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
func (s *Pg) HasAuth(ctx context.Context, u models.User) (bool, error) {
	return s.rowExists(ctx, sqlGetUser, u.Login, u.HexPassword())
}

// SetToken update token to user
func (s *Pg) SetToken(ctx context.Context, u models.User, t string) error {
	_, err := s.db.ExecContext(ctx, sqlUpdateToken, t, u.Login)

	return err
}

// UserByToken check if token exist
func (s *Pg) UserByToken(ctx context.Context, t string) (models.User, error) {
	var usr models.User
	err := s.db.QueryRowContext(ctx, sqlCheckToken, t).Scan(&usr.UserID, &usr.Points, &usr.Withdrawn)
	if err != nil {
		return usr, ErrUserNotFound
	}

	return usr, nil
}

// PutOrder put order in storage
func (s *Pg) PutOrder(ctx context.Context, ord models.Order) error {
	if _, err := s.db.ExecContext(ctx, sqlNewOrder, ord.UserID, ord.Code, models.NEW); err != nil {
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
func (s *Pg) SetStatus(ctx context.Context, orderCode int, status int, timeout int, points int) error {
	// If it's ended status
	if status == models.PROCESSED || status == models.INVALID {
		_, err := s.db.ExecContext(ctx, sqlUpdateDoneStatus, status, points, orderCode, points)
		return err
	}

	if timeout < 1 {
		timeout = 1
	}
	repeatAt := time.Now().Add(time.Duration(timeout) * time.Second).In(time.UTC)
	_, err := s.db.ExecContext(ctx, sqlUpdateStatus, status, orderCode, points, repeatAt)

	return err
}

// Check if exist record by query
func (s *Pg) rowExists(ctx context.Context, query string, args ...interface{}) (bool, error) {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return exists, nil
}

// AddPoints add points to user and done check
func (s *Pg) AddPoints(ctx context.Context, userID int, points int, orderCode int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := s.SetStatus(ctx, orderCode, models.PROCESSED, 0, points); err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, sqlAddPoints, points, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Orders get user orders list
func (s *Pg) Orders(ctx context.Context, userID int) ([]models.Order, error) {
	var orders []models.Order
	rows, err := s.db.QueryContext(ctx, sqlGetOrders, userID)
	if err != nil {
		return orders, err
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	for rows.Next() {
		var userOrder models.Order
		err = rows.Scan(&userOrder.Code, &userOrder.CheckStatus, &userOrder.UploadedAt, &userOrder.Accrual)
		if err != nil {
			return orders, err
		}
		orders = append(orders, userOrder)
	}
	return orders, nil
}

// OrderByCode get order by code
func (s *Pg) OrderByCode(ctx context.Context, code int) (models.Order, error) {
	var userOrder models.Order
	err := s.db.QueryRowContext(ctx, sqlGetOrder, code).Scan(
		&userOrder.ID,
		&userOrder.Code,
		&userOrder.UserID,
		&userOrder.IsCheckDone,
		&userOrder.Attempts,
		&userOrder.Accrual,
		&userOrder.AvailForWithdraw,
	)

	return userOrder, err
}

// OrdersForCheck get chunk for check in loyalty machine
func (s *Pg) OrdersForCheck(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order
	rows, err := s.db.QueryContext(ctx, sqlGetOrdersForCheck)
	if err != nil {
		return orders, err
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	for rows.Next() {
		var userOrder models.Order
		err = rows.Scan(&userOrder.Code, &userOrder.UserID, &userOrder.Attempts)
		if err != nil {
			return orders, err
		}
		orders = append(orders, userOrder)
	}

	return orders, nil
}

// AddWithdraw add withdraw to queue
func (s *Pg) AddWithdraw(ctx context.Context, ord models.Order, points float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := s.db.ExecContext(ctx, sqlAddWithdrawToQueue, ord.UserID, ord.ID, points); err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, sqlUserSubPoints, points, points, ord.UserID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, sqlSubAvailPointsInOrder, points, ord.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Withdraw points from user account
func (s *Pg) Withdraw(ctx context.Context, ord models.Order, points float64) error {
	_, err := s.db.ExecContext(ctx, sqlWithdrawUpdate, ord.UserID, ord.ID, points)
	if err != nil {
		return err
	}

	return nil
}

// ActiveWithdrawals get active withdrawals from list
func (s *Pg) ActiveWithdrawals(ctx context.Context) ([]models.Withdraw, error) {
	var wds []models.Withdraw
	rows, err := s.db.QueryContext(ctx, sqlGetWithdrawals)
	if err != nil {
		return wds, err
	}

	err = rows.Err()
	if err != nil {
		return wds, err
	}

	for rows.Next() {
		var wd models.Withdraw
		err = rows.Scan(&wd.UserID, &wd.OrderID, &wd.Sum)
		if err != nil {
			return wds, err
		}
		wds = append(wds, wd)
	}

	return wds, nil
}

// WithdrawsByUserID get list of user withdrawals
func (s *Pg) WithdrawsByUserID(ctx context.Context, userID int) ([]models.Withdraw, error) {
	var wds []models.Withdraw
	rows, err := s.db.QueryContext(ctx, sqlGetWithdrawalsByUserID, userID)
	if err != nil {
		return wds, err
	}

	err = rows.Err()
	if err != nil {
		return wds, err
	}

	for rows.Next() {
		var wd models.Withdraw
		err = rows.Scan(&wd.Sum, &wd.OrderID, &wd.ProcessedAt, &wd.Status)
		if err != nil {
			return wds, err
		}
		wds = append(wds, wd)
	}

	return wds, nil
}
