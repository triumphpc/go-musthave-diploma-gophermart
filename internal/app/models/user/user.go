package user

import "encoding/hex"

// User model
type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// HexPassword return hex password
func (u *User) HexPassword() string {
	return hex.EncodeToString([]byte(u.Password))
}
