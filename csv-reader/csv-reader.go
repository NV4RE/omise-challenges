package csv_reader

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"go-tamboon/cipher"
)

type Donation struct {
	Name     string
	CCNumber string
	CVV      string
	ExpMonth int
	ExpYear  int
	Amount   int64 // in satang
}

func ReadAndDecryptCSV(filename string) ([]Donation, error) {
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
