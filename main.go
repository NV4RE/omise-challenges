package main

import (
	"fmt"
	"log"
	"os"

	csvreader "go-tamboon/csv-reader"
	"go-tamboon/omise_client"
	"go-tamboon/processor"
	"go-tamboon/summary"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

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

	donations, err := csvreader.ReadAndDecryptCSV(csvFile)
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	client, err := omise_client.NewClient(pkey, skey)
	if err != nil {
		log.Fatalf("Error creating Omise client: %v", err)
	}

	//client := omise_client.NewMockClient()

	s := &summary.Summary{}
	processor.ProcessDonations(client, donations, s)

	printSummary(s)
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
