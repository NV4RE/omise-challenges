package omise_client

import (
	"time"

	"github.com/omise/omise-go"
)

type MockClient struct {
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (c *MockClient) CreateToken(name, number, securityCode string, expMonth time.Month, expYear int) (*omise.Token, error) {
	token := &omise.Token{
		Used: false,
		Card: &omise.Card{
			LastDigits:      number[len(number)-4:],
			Brand:           "Visa",
			ExpirationMonth: expMonth,
			ExpirationYear:  expYear,
			Name:            name,
		},
	}
	return token, nil
}

func (c *MockClient) CreateCharge(amount int64, currency, cardID string) (*omise.Charge, error) {
	charge := &omise.Charge{
		Amount:   amount,
		Currency: currency,
		Status:   "successful",
		Paid:     true,
	}
	return charge, nil
}
