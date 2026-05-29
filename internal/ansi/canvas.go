package ansi

import (
	"fmt"
	"strconv"
	"strings"
)

// Cell is a single terminal cell.
type Cell struct {
	Ch rune
	FG Color
	BG Color
}

// Canvas is a fixed-size 2D grid of cells.
type Canvas struct {
	Width  int
	Height int
	cells  []Cell
}

// NewCanvas creates a canvas of the given dimensions.
// Negative or zero dimensions are clamped to a minimum of 1x1.
func NewCanvas(w, h int) *Canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &Canvas{
		Width:  w,
		Height: h,
		cells:  make([]Cell, w*h),
	}
}

// Set writes a cell. Out-of-bounds writes are silently ignored.
func (c *Canvas) Set(row, col int, ch rune, fg, bg Color) {
	if row < 0 || row >= c.Height || col < 0 || col >= c.Width {
		return
	}
	c.cells[row*c.Width+col] = Cell{Ch: ch, FG: fg, BG: bg}
}

// Get reads a cell. Out-of-bounds reads return an empty Cell.
func (c *Canvas) Get(row, col int) Cell {
	if row < 0 || row >= c.Height || col < 0 || col >= c.Width {
		return Cell{}
	}
	return c.cells[row*c.Width+col]
}

// Clear resets all cells to empty.
func (c *Canvas) Clear() {
	for i := range c.cells {
		c.cells[i] = Cell{}
	}
}

func colorToANSI(c Color, fg bool) string {
	hex := string(c)
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return ""
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 0)
	g, _ := strconv.ParseInt(hex[2:4], 16, 0)
	b, _ := strconv.ParseInt(hex[4:6], 16, 0)
	if fg {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// RenderCanvas converts the canvas to an ANSI string.
// It emits escape codes only when colors change between cells.
func RenderCanvas(c *Canvas) string {
	var b strings.Builder
	b.Grow(c.Width * c.Height * 8)

	var lastFG, lastBG Color

	for row := 0; row < c.Height; row++ {
		b.WriteString(fmt.Sprintf("\x1b[%d;1H", row+1))
		for col := 0; col < c.Width; col++ {
			cell := c.cells[row*c.Width+col]
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			if cell.FG != lastFG {
				if cell.FG == "" {
					b.WriteString("\x1b[39m")
				} else {
					b.WriteString(colorToANSI(cell.FG, true))
				}
				lastFG = cell.FG
			}
			if cell.BG != lastBG {
				if cell.BG == "" {
					b.WriteString("\x1b[49m")
				} else {
					b.WriteString(colorToANSI(cell.BG, false))
				}
				lastBG = cell.BG
			}
			b.WriteRune(ch)
		}
		b.WriteString("\x1b[K")
	}
	b.WriteString("\x1b[0m")
	return b.String()
}
