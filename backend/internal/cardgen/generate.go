package cardgen

import (
	"errors"
	"fmt"
)

// Card represents a Dobble card with an ID and symbols.
type Card struct {
	ID      int
	Symbols []int
}

// Item represents a symbol belonging to a card.
type Item struct {
	CardID int
	Symbol int
}

// GenerateDobbleCards generates Dobble cards based on the mathematical theory.
// n must be a prime number.
func GenerateDobbleCards(n int) ([]Card, []Item, error) {
	if n < 2 {
		return nil, nil, errors.New("n must be a prime number greater than 1")
	}

	var cards []Card
	var items []Item
	cardID := 0

	// First card
	firstCard := Card{ID: cardID}
	for i := 0; i <= n; i++ {
		firstCard.Symbols = append(firstCard.Symbols, i)
	}
	cards = append(cards, firstCard)
	cardID++

	// Next n cards
	for i := 0; i < n; i++ {
		card := Card{ID: cardID}
		card.Symbols = append(card.Symbols, 0)
		for j := 0; j < n; j++ {
			card.Symbols = append(card.Symbols, (n+1)+(n*i)+j)
		}
		cards = append(cards, card)
		cardID++
	}

	// Remaining n*n cards
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			card := Card{ID: cardID}
			card.Symbols = append(card.Symbols, i+1)
			for k := 0; k < n; k++ {
				card.Symbols = append(card.Symbols, (n+1)+(n*k)+((i*k+j)%n))
			}
			cards = append(cards, card)
			cardID++
		}
	}

	// Generate items
	for _, card := range cards {
		for _, symbol := range card.Symbols {
			items = append(items, Item{CardID: card.ID, Symbol: symbol})
		}
	}

	return cards, items, nil
}

// DebugPrint prints cards for debugging purposes.
func DebugPrint(cards []Card) {
	for _, card := range cards {
		fmt.Printf("Card %d: %v\n", card.ID, card.Symbols)
	}
}
