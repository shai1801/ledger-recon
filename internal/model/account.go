package model

import "time"

type Account struct {
	AccountID           string
	CustomerID          string
	AccountType         string
	Currency            string
	OpeningBalanceCents int64
	Status              string
	CreatedAt           time.Time
}
