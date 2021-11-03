package models

import "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"

type Withdraw struct {
	UserID      int               `json:"-"`
	OrderID     string            `json:"order"`
	Sum         float64           `json:"sum"`
	Status      string            `json:"-"`
	ProcessedAt jsontime.JSONTime `json:"processed_at"`
}
