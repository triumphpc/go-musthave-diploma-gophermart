package user

import "encoding/hex"

// User model
type User struct {
	UserID    int     `json:"-"`
	Login     string  `json:"login"`
	Password  string  `json:"password"`
	Points    float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// HexPassword return hex password
func (u *User) HexPassword() string {
	return hex.EncodeToString([]byte(u.Password))
}
