package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"go-tamboon/cipher"
	"go-tamboon/summary"

	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/time/rate"
)

const (
	maxRateLimitPerSecond = 5
	maxConcurrentRequests = 4
	cooldownDuration      = 100 * time.Millisecond
)

type Donation struct {
	Name     string
	CCNumber string
	CVV      string
	ExpMonth int
	ExpYear  int
	Amount   int64 // in satang
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go-tamboon <csv-file>")
		os.Exit(1)
	}

	pkey := os.Getenv("OMISE_PUBLIC_KEY")
	if pkey == "" {
		log.Fatal("OMISE_PUBLIC_KEY is not set")
	}

	skey := os.Getenv("OMISE_SECRET_KEY")
	if skey == "" {
		log.Fatal("OMISE_SECRET_KEY is not set")
	}

	csvFile := os.Args[1]

	donations, err := readAndDecryptCSV(csvFile)
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	client, err := omise.NewClient(pkey, skey)
	if err != nil {
		log.Fatalf("Error creating Omise client: %v", err)
	}

	s := &summary.Summary{}
	processDonations(client, donations, s)

	printSummary(s)
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

		amount, err := strconv.ParseUint(record[1], 10, 64)
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
			Amount:   int64(amount),
			CCNumber: record[2],
			CVV:      record[3],
			ExpMonth: int(expMonth),
			ExpYear:  int(expYear),
		}
		donations = append(donations, donation)
	}

	return donations, nil
}

func processDonations(client *omise.Client, donations []Donation, summary *summary.Summary) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests) // Limit concurrent requests

	limiter := rate.NewLimiter(rate.Limit(maxRateLimitPerSecond), 1)

	for i, donation := range donations {
		wg.Add(1)
		go func(d Donation, idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Wait for rate limiter
			if err := limiter.Wait(context.Background()); err != nil {
				log.Printf("Rate limiter error: %v", err)
				return
			}

			processSingleDonation(idx, client, d, summary)
			time.Sleep(cooldownDuration)
		}(donation, i)
	}

	wg.Wait()
}

func processSingleDonation(idx int, client *omise.Client, donation Donation, summary *summary.Summary) {
	summary.AddTotalReceived(donation.Amount)

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
		log.Printf("Error creating token at record %d: %v", idx, err)
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
		log.Printf("Error creating charge at record %d: %v", idx, err)
		return
	}

	summary.AddSuccessDonation(donation.Amount, donation.Name)
}

func printSummary(summary *summary.Summary) {
	fmt.Printf("        total received: THB %s\n", formatAmount(summary.TotalReceived))
	fmt.Printf("  successfully donated: THB %s\n", formatAmount(summary.SuccessfulDonations))
	fmt.Printf("       faulty donation: THB %s\n", formatAmount(summary.TotalReceived-summary.SuccessfulDonations))
	fmt.Println()

	if summary.Count > 0 {
		avg := summary.TotalReceived / int64(summary.Count)
		fmt.Printf("    average per person: THB %s\n", formatAmount(avg))
	}

	topDonors := summary.GetTopDonators(3)
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
