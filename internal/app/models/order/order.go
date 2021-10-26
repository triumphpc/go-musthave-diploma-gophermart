// Package order consist types and consist for order model
// @author Sergey Vrulin (aka Alex Versus) 2021
package order

import "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"

// Order statuses
const (
	NEW = iota
	PROCESSING
	INVALID
	PROCESSED
)

// Available statues in loyal machine
const (
	LoyalRegistered = "REGISTERED"
	LoyalInvalid    = "INVALID"
	LoyalProcessing = "PROCESSING"
	LoyalProcessed  = "PROCESSED"
)

// LoyalOrder order type from loyal machine
type LoyalOrder struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

// Order user list
type Order struct {
	Code        int `json:"number"`
	UserID      int
	CheckStatus string            `json:"status"`
	Accrual     int               `json:"accrual,omitempty"`
	UploadedAt  jsontime.JSONTime `json:"uploaded_at"`
	Attempts    int
	IsCheckDone bool
}
