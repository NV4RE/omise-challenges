package processor

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	csvreader "go-tamboon/csv-reader"
	"go-tamboon/omise_client"
	"go-tamboon/summary"

	"github.com/cenkalti/backoff/v4"
	"github.com/omise/omise-go"
	"golang.org/x/time/rate"
)

const (
	MaxRateLimitPerSecond = 5
	MaxConcurrentRequests = 4
	CooldownDuration      = 100 * time.Millisecond
)

func ProcessDonations(client omise_client.OmiseClient, donations []csvreader.Donation, summary *summary.Summary) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxConcurrentRequests) // Limit concurrent requests

	limiter := rate.NewLimiter(rate.Limit(MaxRateLimitPerSecond), 1)

	for i, donation := range donations {
		wg.Add(1)
		go func(d csvreader.Donation, idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Wait for rate limiter
			if err := limiter.Wait(context.Background()); err != nil {
				log.Printf("Rate limiter error: %v", err)
				return
			}

			processSingleDonation(idx, client, d, summary)
			time.Sleep(CooldownDuration)
		}(donation, i)
	}

	wg.Wait()
}

func doWithRetry(operation func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 60 * time.Second
	bo.MaxInterval = 10 * time.Second
	bo.InitialInterval = 500 * time.Millisecond
	return backoff.Retry(operation, bo)
}

func processSingleDonation(idx int, client omise_client.OmiseClient, donation csvreader.Donation, summary *summary.Summary) {
	summary.AddTotalReceived(donation.Amount)

	// Create token with retry
	var token *omise.Token
	err := doWithRetry(func() error {
		var err error
		token, err = client.CreateToken(
			donation.Name,
			donation.CCNumber,
			donation.CVV,
			time.Month(donation.ExpMonth),
			donation.ExpYear,
		)
		var ex *omise.Error
		if errors.As(err, &ex) && ex.Code == "too_many_requests" {
			log.Printf("Too many requests, retrying at record %d", idx)
			return err // Return error to trigger retry
		}
		// For all other errors, don't retry
		return backoff.Permanent(err)
	})
	if err != nil {
		log.Printf("Error creating token at record %d after retries: %v", idx, err)
		return
	}

	// Create charge with retry
	err = doWithRetry(func() error {
		_, err := client.CreateCharge(
			donation.Amount,
			"thb",
			token.ID,
		)
		var ex *omise.Error
		if errors.As(err, &ex) && ex.Code == "too_many_requests" {
			log.Printf("Too many requests, retrying charge at record %d", idx)
			return err // Return error to trigger retry
		}
		// For all other errors, don't retry
		return backoff.Permanent(err)
	})
	if err != nil {
		log.Printf("Error creating charge at record %d after retries: %v", idx, err)
		return
	}

	summary.AddSuccessDonation(donation.Amount, donation.Name)
}
