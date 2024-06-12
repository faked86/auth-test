package models

type User struct {
	ID           string
	RefreshToken []byte
	Email        string
}
