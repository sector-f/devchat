package main

import (
	"github.com/jwalton/gchalk"
)

var (
	chalk     = gchalk.New(gchalk.ForceLevel(gchalk.LevelAnsi256))
	green     = ansi256(1, 5, 1)
	darkGreen = ansi256(1, 3, 1)
	red       = ansi256(5, 1, 1)
	cyan      = ansi256(1, 5, 5)
	magenta   = ansi256(5, 1, 5)
	yellow    = ansi256(5, 5, 1)
	orange    = ansi256(5, 3, 0)
	blue      = ansi256(0, 3, 5)
	white     = ansi256(5, 5, 5)
)

// with r, g and b values from 0 to 5
func ansi256(r, g, b uint8) *gchalk.Builder {
	return chalk.WithRGB(255/5*r, 255/5*g, 255/5*b)
}

func bgAnsi256(r, g, b uint8) *gchalk.Builder {
	return chalk.WithBgRGB(255/5*r, 255/5*g, 255/5*b)
}
