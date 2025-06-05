package cardgen

import (
	"context"
	"fmt"

	"example/ent"
	"example/ent/card"
	"example/ent/item"
)

// SaveCardsAndItems saves generated cards and items to the database.
func SaveCardsAndItems(ctx context.Context, client *ent.Client, cards []Card, items []Item) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	// Save cards
	cardEntities := make(map[int]*ent.Card)
	for _, card := range cards {
		c, err := tx.Card.Create().
			Save(ctx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("saving card %d: %w", card.ID, err)
		}
		cardEntities[card.ID] = c
	}

	// Save items
	for _, item := range items {
		_, err := tx.Item.Create().
			SetSymbol(fmt.Sprintf("%d", item.Symbol)).
			AddParent(cardEntities[item.CardID]).
			Save(ctx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("saving item (cardID: %d, symbol: %d): %w", item.CardID, item.Symbol, err)
		}
	}

	return tx.Commit()
}

// FindCommonSymbol returns the common symbol between two cards given their IDs.
func FindCommonSymbol(ctx context.Context, client *ent.Client, cardID1, cardID2 int) (string, error) {
	items1, err := client.Item.Query().Where(item.HasParentWith(card.ID(cardID1))).All(ctx)
	if err != nil {
		return "", fmt.Errorf("querying items for card %d: %w", cardID1, err)
	}
	fmt.Printf("Items for card %d: %+v\n", cardID1, items1)
	if err != nil {
		return "", fmt.Errorf("querying items for card %d: %w", cardID1, err)
	}

	items2, err := client.Item.Query().Where(item.HasParentWith(card.ID(cardID2))).All(ctx)
	if err != nil {
		return "", fmt.Errorf("querying items for card %d: %w", cardID2, err)
	}
	fmt.Printf("Items for card %d: %+v\n", cardID2, items2)
	if err != nil {
		return "", fmt.Errorf("querying items for card %d: %w", cardID2, err)
	}

	symbolSet := make(map[string]struct{})
	for _, item := range items1 {
		symbolSet[item.Symbol] = struct{}{}
	}

	for _, item := range items2 {
		if _, exists := symbolSet[item.Symbol]; exists {
			return item.Symbol, nil
		}
	}

	return "", nil // no common symbol found
}
