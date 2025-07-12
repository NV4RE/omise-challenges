package omise_client

import (
	"time"

	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
)

type Client struct {
	omiseClient *omise.Client
}

func NewClient(publicKey, secretKey string) (*Client, error) {
	client, err := omise.NewClient(publicKey, secretKey)
	if err != nil {
		return nil, err
	}
	return &Client{omiseClient: client}, nil
}

func (c *Client) CreateToken(name, number, securityCode string, expMonth time.Month, expYear int) (*omise.Token, error) {
	token := &omise.Token{}
	createToken := &operations.CreateToken{
		Name:            name,
		Number:          number,
		SecurityCode:    securityCode,
		ExpirationMonth: expMonth,
		ExpirationYear:  expYear,
	}

	err := c.omiseClient.Do(token, createToken)
	return token, err
}

func (c *Client) CreateCharge(amount int64, currency, cardID string) (*omise.Charge, error) {
	charge := &omise.Charge{}
	createCharge := &operations.CreateCharge{
		Amount:   amount,
		Currency: currency,
		Card:     cardID,
	}

	err := c.omiseClient.Do(charge, createCharge)
	return charge, err
}
