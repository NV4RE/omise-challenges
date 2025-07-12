package omise_client

import (
	"time"

	"github.com/omise/omise-go"
)

type OmiseClient interface {
	CreateToken(name, number, securityCode string, expMonth time.Month, expYear int) (*omise.Token, error)
	CreateCharge(amount int64, currency, cardID string) (*omise.Charge, error)
}
