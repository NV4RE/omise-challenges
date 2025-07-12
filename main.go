package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"go-tamboon/cipher"

	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/time/rate"
)

const (
	maxRateLimitPerSecond = 6
	maxConcurrentRequests = 4
)

type Donation struct {
	Name     string
	CCNumber string
	CVV      string
	ExpMonth int
	ExpYear  int
	Amount   int64 // in satang
}

type Donor struct {
	Name   string
	Amount int64
}

type Summary struct {
	TotalReceived       int64
	SuccessfulDonations int64
	Count               int
	Donors              map[string]int64
	mu                  sync.Mutex
}

func (s *Summary) addTotalReceived(amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalReceived += amount
	s.Count++
}

func (s *Summary) addSuccessDonation(amount int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Donors == nil {
		s.Donors = make(map[string]int64)
	}
	s.Donors[name] += amount
	s.SuccessfulDonations += amount
}

func (s *Summary) getTopDonator(top int) []Donor {
	s.mu.Lock()
	defer s.mu.Unlock()

	donors := make([]Donor, 0, len(s.Donors))
	for name, amount := range s.Donors {
		donors = append(donors, Donor{Name: name, Amount: amount})
	}

	sort.Slice(donors, func(i, j int) bool {
		return donors[i].Amount > donors[j].Amount
	})

	if len(donors) > top {
		donors = donors[:top]
	}

	return donors
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go-tamboon <csv-file>")
		os.Exit(1)
	}

	csvFile := os.Args[1]

	donations, err := readAndDecryptCSV(csvFile)
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	client, err := omise.NewClient(
		os.Getenv("OMISE_PUBLIC_KEY"),
		os.Getenv("OMISE_SECRET_KEY"),
	)
	if err != nil {
		log.Fatalf("Error creating Omise client: %v", err)
	}

	summary := &Summary{}
	processDonations(client, donations, summary)

	printSummary(summary)
}

func readAndDecryptCSV(filename string) ([]Donation, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Warning: failed to close file: %v", closeErr)
		}
	}()

	rot128Reader, err := cipher.NewRot128Reader(file)
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(rot128Reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	var donations []Donation
	for i, record := range records {
		if i == 0 {
			continue
		}

		if len(record) != 6 {
			log.Printf("Incorrect number of fields at record %d", i)
			continue
		}

		amount, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			log.Printf("Error parsing amount at record %d", i)
			continue
		}

		expMonth, err := strconv.ParseUint(record[4], 10, 32)
		if err != nil {
			log.Printf("Error parsing expiry month at record %d", i)
			continue
		}

		expYear, err := strconv.ParseUint(record[5], 10, 32)
		if err != nil {
			log.Printf("Error parsing expiry year at record %d", i)
			continue
		}

		donation := Donation{
			Name:     record[0],
			Amount:   amount,
			CCNumber: record[2],
			CVV:      record[3],
			ExpMonth: int(expMonth),
			ExpYear:  int(expYear),
		}
		donations = append(donations, donation)
	}

	return donations, nil
}

func processDonations(client *omise.Client, donations []Donation, summary *Summary) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests) // Limit concurrent requests

	limiter := rate.NewLimiter(rate.Limit(maxRateLimitPerSecond), 1)

	for _, donation := range donations {
		wg.Add(1)
		go func(d Donation) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Wait for rate limiter
			if err := limiter.Wait(context.Background()); err != nil {
				log.Printf("Rate limiter error: %v", err)
				return
			}

			processSingleDonation(client, d, summary)
		}(donation)
	}

	wg.Wait()
}

func processSingleDonation(client *omise.Client, donation Donation, summary *Summary) {
	summary.addTotalReceived(donation.Amount)

	// Create token
	token := &omise.Token{}
	createToken := &operations.CreateToken{
		Name:            donation.Name,
		Number:          donation.CCNumber,
		SecurityCode:    donation.CVV,
		ExpirationMonth: time.Month(donation.ExpMonth),
		ExpirationYear:  donation.ExpYear,
	}

	if err := client.Do(token, createToken); err != nil {
		log.Printf("Error creating token: %v", err)
		return
	}

	// Create charge
	charge := &omise.Charge{}
	createCharge := &operations.CreateCharge{
		Amount:   donation.Amount,
		Currency: "thb",
		Card:     token.ID,
	}

	if err := client.Do(charge, createCharge); err != nil {
		log.Printf("Error creating charge: %v", err)
		return
	}

	summary.addSuccessDonation(donation.Amount, donation.Name)
}

func printSummary(summary *Summary) {
	fmt.Printf("        total received: THB %s\n", formatAmount(summary.TotalReceived))
	fmt.Printf("  successfully donated: THB %s\n", formatAmount(summary.SuccessfulDonations))
	fmt.Printf("       faulty donation: THB %s\n", formatAmount(summary.TotalReceived-summary.SuccessfulDonations))
	fmt.Println()

	if summary.Count > 0 {
		avg := summary.TotalReceived / int64(summary.Count)
		fmt.Printf("    average per person: THB %s\n", formatAmount(avg))
	}

	topDonors := summary.getTopDonator(3)
	if len(topDonors) > 0 {
		fmt.Printf("            top donors: %s (THB %s)\n",
			topDonors[0].Name, formatAmount(topDonors[0].Amount))
		for i := 1; i < len(topDonors); i++ {
			fmt.Printf("                        %s (THB %s)\n",
				topDonors[i].Name, formatAmount(topDonors[i].Amount))
		}
	}
}

func formatAmount(amount int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d.%02d", amount/100, amount%100)
}
