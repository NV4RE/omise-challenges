package summary

import (
	"sync"
	"testing"
)

func TestSummary_AddTotalReceived(t *testing.T) {
	s := &Summary{}

	s.AddTotalReceived(1000)
	if s.TotalReceived != 1000 {
		t.Errorf("Expected TotalReceived to be 1000, got %d", s.TotalReceived)
	}
	if s.Count != 1 {
		t.Errorf("Expected Count to be 1, got %d", s.Count)
	}

	s.AddTotalReceived(500)
	if s.TotalReceived != 1500 {
		t.Errorf("Expected TotalReceived to be 1500, got %d", s.TotalReceived)
	}
	if s.Count != 2 {
		t.Errorf("Expected Count to be 2, got %d", s.Count)
	}
}

func TestSummary_AddSuccessDonation(t *testing.T) {
	s := &Summary{}

	s.AddSuccessDonation(1000, "Alice")
	if s.SuccessfulDonations != 1000 {
		t.Errorf("Expected SuccessfulDonations to be 1000, got %d", s.SuccessfulDonations)
	}
	if s.Donors["Alice"] != 1000 {
		t.Errorf("Expected Alice's donation to be 1000, got %d", s.Donors["Alice"])
	}

	s.AddSuccessDonation(500, "Bob")
	if s.SuccessfulDonations != 1500 {
		t.Errorf("Expected SuccessfulDonations to be 1500, got %d", s.SuccessfulDonations)
	}
	if s.Donors["Bob"] != 500 {
		t.Errorf("Expected Bob's donation to be 500, got %d", s.Donors["Bob"])
	}

	// Test multiple donations from same person
	s.AddSuccessDonation(300, "Alice")
	if s.SuccessfulDonations != 1800 {
		t.Errorf("Expected SuccessfulDonations to be 1800, got %d", s.SuccessfulDonations)
	}
	if s.Donors["Alice"] != 1300 {
		t.Errorf("Expected Alice's total donation to be 1300, got %d", s.Donors["Alice"])
	}
}

func TestSummary_GetTopDonators(t *testing.T) {
	s := &Summary{}

	// Add some donations
	s.AddSuccessDonation(1000, "Alice")
	s.AddSuccessDonation(500, "Bob")
	s.AddSuccessDonation(1500, "Charlie")
	s.AddSuccessDonation(200, "David")
	s.AddSuccessDonation(800, "Eve")

	// Test getting top 3
	top3 := s.GetTopDonators(3)
	if len(top3) != 3 {
		t.Errorf("Expected 3 top donators, got %d", len(top3))
	}

	// Should be sorted by amount descending
	if top3[0].Name != "Charlie" || top3[0].Amount != 1500 {
		t.Errorf("Expected Charlie to be first with 1500, got %s with %d", top3[0].Name, top3[0].Amount)
	}
	if top3[1].Name != "Alice" || top3[1].Amount != 1000 {
		t.Errorf("Expected Alice to be second with 1000, got %s with %d", top3[1].Name, top3[1].Amount)
	}
	if top3[2].Name != "Eve" || top3[2].Amount != 800 {
		t.Errorf("Expected Eve to be third with 800, got %s with %d", top3[2].Name, top3[2].Amount)
	}

	// Test getting more than available
	top10 := s.GetTopDonators(10)
	if len(top10) != 5 {
		t.Errorf("Expected 5 donators (all available), got %d", len(top10))
	}

	// Test getting top 0
	top0 := s.GetTopDonators(0)
	if len(top0) != 0 {
		t.Errorf("Expected 0 donators, got %d", len(top0))
	}
}

func TestSummary_EmptyGetTopDonators(t *testing.T) {
	s := &Summary{}

	top := s.GetTopDonators(5)
	if len(top) != 0 {
		t.Errorf("Expected 0 donators for empty summary, got %d", len(top))
	}
}

func TestSummary_ConcurrentAccess(t *testing.T) {
	s := &Summary{}
	var wg sync.WaitGroup

	// Test concurrent AddTotalReceived
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(amount int64) {
			defer wg.Done()
			s.AddTotalReceived(amount)
		}(int64(i))
	}

	// Test concurrent AddSuccessDonation
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.AddSuccessDonation(int64(i*10), "User"+string(rune(i%26+'A')))
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	expectedTotal := int64(0)
	for i := 0; i < 100; i++ {
		expectedTotal += int64(i)
	}
	if s.TotalReceived != expectedTotal {
		t.Errorf("Expected TotalReceived to be %d, got %d", expectedTotal, s.TotalReceived)
	}

	if s.Count != 100 {
		t.Errorf("Expected Count to be 100, got %d", s.Count)
	}

	// Test concurrent GetTopDonators doesn't panic
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.GetTopDonators(5)
		}()
	}

	wg.Wait()
}

func TestSummary_MultipleSuccessfulDonationsFromSamePerson(t *testing.T) {
	s := &Summary{}

	s.AddSuccessDonation(1000, "Alice")
	s.AddSuccessDonation(500, "Alice")
	s.AddSuccessDonation(300, "Alice")

	if s.SuccessfulDonations != 1800 {
		t.Errorf("Expected SuccessfulDonations to be 1800, got %d", s.SuccessfulDonations)
	}

	if s.Donors["Alice"] != 1800 {
		t.Errorf("Expected Alice's total to be 1800, got %d", s.Donors["Alice"])
	}

	if len(s.Donors) != 1 {
		t.Errorf("Expected 1 unique donor, got %d", len(s.Donors))
	}
}
