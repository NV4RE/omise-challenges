package summary

import (
	"sort"
	"sync"
)

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

func (s *Summary) AddTotalReceived(amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalReceived += amount
	s.Count++
}

func (s *Summary) AddSuccessDonation(amount int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Donors == nil {
		s.Donors = make(map[string]int64)
	}
	s.Donors[name] += amount
	s.SuccessfulDonations += amount
}

func (s *Summary) GetTopDonators(top int) []Donor {
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
