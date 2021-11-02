package models

import "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"

type Withdraw struct {
	UserID      int
	OrderID     int
	Sum         float64
	Status      string
	ProcessedAt jsontime.JSONTime
}
