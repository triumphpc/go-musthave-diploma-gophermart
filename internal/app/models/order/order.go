package order

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

type LoyalOrder struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}
