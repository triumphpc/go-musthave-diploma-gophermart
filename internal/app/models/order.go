package models

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
	ID               int               `json:"-"`
	Code             int               `json:"number"`
	UserID           int               `json:"-"`
	CheckStatus      string            `json:"status"`
	Accrual          float64           `json:"accrual,omitempty"`
	UploadedAt       jsontime.JSONTime `json:"uploaded_at"`
	Attempts         int               `json:"-"`
	IsCheckDone      bool              `json:"-"`
	AvailForWithdraw float64           `json:"-"`
}
