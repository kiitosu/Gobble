package cardgen

import (
	"testing"
)

func TestGenerateDobbleCards(t *testing.T) {
	n := 3
	cards, _, err := GenerateDobbleCards(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	DebugPrint(cards)

	// Check the number of cards
	expectedCardCount := n*n + n + 1
	if len(cards) != expectedCardCount {
		t.Fatalf("expected %d cards, got %d", expectedCardCount, len(cards))
	}

	// Check Dobble rule: exactly one common symbol between any two cards
	for i := 0; i < len(cards); i++ {
		for j := i + 1; j < len(cards); j++ {
			commonSymbols := 0
			for _, sym1 := range cards[i].Symbols {
				for _, sym2 := range cards[j].Symbols {
					if sym1 == sym2 {
						commonSymbols++
					}
				}
			}
			if commonSymbols != 1 {
				t.Errorf("cards %d and %d have %d common symbols, expected exactly 1", cards[i].ID, cards[j].ID, commonSymbols)
			}
		}
	}
}
