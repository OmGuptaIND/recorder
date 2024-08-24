package config

import "github.com/OmGuptaIND/display"

var DEFAULT_DISPLAY_OPTS = display.DisplayOptions{
	Width:  1280,
	Height: 720,
	Depth:  24,
}

type ContextKey string

const (
	StoreKey ContextKey = "store"
)
