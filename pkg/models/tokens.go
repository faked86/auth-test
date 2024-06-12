package models

import "time"

type Tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

type ParsedToken struct {
	IP  string
	ID  string
	Exp time.Time
}
