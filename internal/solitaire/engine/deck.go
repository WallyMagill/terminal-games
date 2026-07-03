package engine

import "math/rand/v2"

func standardDeck() []Card {
	deck := make([]Card, 0, 52)
	for suit := Spades; suit <= Clubs; suit++ {
		for rank := Ace; rank <= King; rank++ {
			deck = append(deck, Card{Rank: rank, Suit: suit})
		}
	}
	return deck
}

func shuffle(deck []Card, rng *rand.Rand) {
	rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}
