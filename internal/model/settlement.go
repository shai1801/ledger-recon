package model

import "time"

//This model is to describe a row from bank_settlements.csv

type BankSettlement struct {
	SettlementID string
	BankRef      string
	AccountID    string
	AmountCents  int64
	Currency     string
	SettledAt    time.Time
	Status       string //SETTLED, PENDING, RETURNED
	Processor    string
}
