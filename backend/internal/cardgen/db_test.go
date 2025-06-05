package cardgen

import (
	"context"
	"testing"

	"example/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

func TestSaveCardsAndItems(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:cardgen_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// Dobbleカード生成
	n := 3
	cards, items, err := GenerateDobbleCards(n)
	if err != nil {
		t.Fatalf("failed to generate cards: %v", err)
	}

	// DBへ保存
	if err := SaveCardsAndItems(ctx, client, cards, items); err != nil {
		t.Fatalf("failed to save cards and items: %v", err)
	}

	// Test FindCommonSymbol
	cardEntities, err := client.Card.Query().All(ctx)
	if err != nil {
		t.Fatalf("failed to query cards: %v", err)
	}
	cardIDMap := make(map[int]int)
	for i, card := range cardEntities {
		cardIDMap[cards[i].ID] = card.ID
	}

	cardID1 := cardIDMap[cards[0].ID]
	cardID2 := cardIDMap[cards[1].ID]
	commonSymbol, err := FindCommonSymbol(ctx, client, cardID1, cardID2)
	if err != nil {
		t.Fatalf("failed to find common symbol: %v", err)
	}
	if commonSymbol == "" {
		t.Errorf("expected a common symbol between card %d and card %d, but got none", cardID1, cardID2)
	}

	// カード数検証
	cardCount, err := client.Card.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count cards: %v", err)
	}
	expectedCardCount := n*n + n + 1
	if cardCount != expectedCardCount {
		t.Errorf("expected %d cards, got %d", expectedCardCount, cardCount)
	}

	// アイテム数検証
	itemCount, err := client.Item.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count items: %v", err)
	}
	expectedItemCount := 0
	for _, c := range cards {
		expectedItemCount += len(c.Symbols)
	}
	if itemCount != expectedItemCount {
		t.Errorf("expected %d items, got %d", expectedItemCount, itemCount)
	}
}
