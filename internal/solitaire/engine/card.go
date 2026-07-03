// Package engine implements classic Klondike solitaire rules with no UI dependencies.
package engine

type Suit uint8

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

type Rank uint8

const (
	Ace Rank = iota + 1
	Two
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
)

type Card struct {
	Rank Rank
	Suit Suit
}

func (s Suit) Red() bool { return s == Hearts || s == Diamonds }

type TableauCard struct {
	Card   Card
	FaceUp bool
}
