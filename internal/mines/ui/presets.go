// Package ui implements the Bubble Tea presentation layer.
package ui

// Preset describes a board configuration. Key is the store key for best
// times; it is empty for custom boards, which never record times.
type Preset struct {
	Name   string
	Key    string
	Width  int
	Height int
	Mines  int
}

var presets = []Preset{
	{Name: "Beginner", Key: "beginner", Width: 9, Height: 9, Mines: 10},
	{Name: "Intermediate", Key: "intermediate", Width: 16, Height: 16, Mines: 40},
	{Name: "Expert", Key: "expert", Width: 30, Height: 16, Mines: 99},
}
