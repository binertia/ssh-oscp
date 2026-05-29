package ansi



// DisplayWidth returns the number of terminal columns occupied by a string.
// Handles CJK wide characters and common double-width runes.
func DisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += RuneWidth(r)
	}
	return w
}

// RuneWidth returns the display width of a single rune.
func RuneWidth(r rune) int {
	if r < 0x20 || r == 0x7f {
		return 0
	}
	// CJK, Hangul, fullwidth forms, and other wide blocks
	if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x20000 && r <= 0x2fffd) ||
		(r >= 0x30000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

// TruncateDisplay truncates a string to fit within max display columns.
// Appends "…" if truncation occurs.
func TruncateDisplay(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if DisplayWidth(s) <= max {
		return s
	}
	var b []rune
	w := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if w+rw > max-1 { // reserve 1 column for ellipsis
			break
		}
		b = append(b, r)
		w += rw
	}
	return string(b) + "…"
}

// PadRight pads a string to exactly max display columns with spaces on the right.
func PadRight(s string, max int) string {
	pad := max - DisplayWidth(s)
	if pad <= 0 {
		return s
	}
	for i := 0; i < pad; i++ {
		s += " "
	}
	return s
}

// Box is a bordered rectangular region.
type Box struct {
	X, Y          int // top-left corner
	Width, Height int // total dimensions including borders
	Title         string
}

// InnerWidth returns the usable horizontal space inside the box.
func (b *Box) InnerWidth() int {
	iw := b.Width - 4 // 2 borders + 2 padding spaces
	if iw < 0 {
		return 0
	}
	return iw
}

// InnerHeight returns the usable vertical space inside the box.
func (b *Box) InnerHeight() int {
	ih := b.Height - 2 // top and bottom borders only
	if ih < 0 {
		return 0
	}
	return ih
}

// Draw renders the box border onto c.  Skips cells outside canvas bounds.
func (b *Box) Draw(c *Canvas, fg Color) {
	if b.Width < 2 || b.Height < 2 {
		return
	}

	// Corners
	b.safeSet(c, b.Y, b.X, '┏', fg)
	b.safeSet(c, b.Y, b.X+b.Width-1, '┓', fg)
	b.safeSet(c, b.Y+b.Height-1, b.X, '┗', fg)
	b.safeSet(c, b.Y+b.Height-1, b.X+b.Width-1, '┛', fg)

	// Horizontal borders
	for x := b.X + 1; x < b.X+b.Width-1; x++ {
		b.safeSet(c, b.Y, x, '━', fg)
		b.safeSet(c, b.Y+b.Height-1, x, '━', fg)
	}

	// Vertical borders
	for y := b.Y + 1; y < b.Y+b.Height-1; y++ {
		b.safeSet(c, y, b.X, '┃', fg)
		b.safeSet(c, y, b.X+b.Width-1, '┃', fg)
	}

	// Title integrated into top border
	if b.Title != "" && b.Width > 4 {
		titleMax := b.Width - 4
		title := TruncateDisplay(b.Title, titleMax)
		pad := b.Width - 2 - DisplayWidth(title)
		leftPad := pad / 2
		titleX := b.X + 1 + leftPad
		for _, ch := range title {
			b.safeSet(c, b.Y, titleX, ch, fg)
			titleX += RuneWidth(ch)
		}
	}
}

// SetContent draws a line of text inside the box at inner row `row`.
// Text is left-aligned inside the inner area.  Truncated if too long.
func (b *Box) SetContent(c *Canvas, row int, text string, fg Color) {
	if row < 0 || row >= b.InnerHeight() {
		return
	}
	y := b.Y + 1 + row
	if y < 0 || y >= c.Height {
		return
	}
	x := b.X + 2
	innerW := b.InnerWidth()
	text = TruncateDisplay(text, innerW)
	for _, ch := range text {
		if x >= c.Width {
			break
		}
		b.safeSet(c, y, x, ch, fg)
		x += RuneWidth(ch)
	}
}

// FillBackground fills the inner area of the box with a background color.
func (b *Box) FillBackground(c *Canvas, bg Color) {
	for y := b.Y + 1; y < b.Y+b.Height-1; y++ {
		for x := b.X + 1; x < b.X+b.Width-1; x++ {
			b.safeSetBG(c, y, x, ' ', "", bg)
		}
	}
}

// safeSet writes a cell only if the coordinates are inside the canvas.
func (b *Box) safeSet(c *Canvas, row, col int, ch rune, fg Color) {
	if row < 0 || row >= c.Height || col < 0 || col >= c.Width {
		return
	}
	c.Set(row, col, ch, fg, "")
}

// safeSetBG writes a cell with a specific background color.
func (b *Box) safeSetBG(c *Canvas, row, col int, ch rune, fg, bg Color) {
	if row < 0 || row >= c.Height || col < 0 || col >= c.Width {
		return
	}
	c.Set(row, col, ch, fg, bg)
}

// DrawHLine draws a horizontal line of ch from (x, y) for length count.
func DrawHLine(c *Canvas, x, y, count int, ch rune, fg Color) {
	for i := 0; i < count; i++ {
		col := x + i
		if col < 0 || col >= c.Width || y < 0 || y >= c.Height {
			continue
		}
		c.Set(y, col, ch, fg, "")
	}
}

// DrawVLine draws a vertical line of ch from (x, y) for length count.
func DrawVLine(c *Canvas, x, y, count int, ch rune, fg Color) {
	for i := 0; i < count; i++ {
		row := y + i
		if row < 0 || row >= c.Height || x < 0 || x >= c.Width {
			continue
		}
		c.Set(row, x, ch, fg, "")
	}
}

// CenterRect returns the top-left coordinate to center a rectangle.
// Result is clamped so the rectangle stays within (0,0) to (termW,termH).
func CenterRect(termW, termH, rectW, rectH int) (x, y int) {
	x = (termW - rectW) / 2
	y = (termH - rectH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	// Clamp right/bottom edges
	if x+rectW > termW {
		x = termW - rectW
		if x < 0 {
			x = 0
		}
	}
	if y+rectH > termH {
		y = termH - rectH
		if y < 0 {
			y = 0
		}
	}
	return
}

// Clamp returns v constrained to [min, max].
func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// WrapText wraps text into lines of at most width display columns.
func WrapText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	var lines []string
	words := splitWords(text)
	var current string
	curW := 0
	for _, w := range words {
		ww := DisplayWidth(w)
		if curW > 0 && curW+1+ww > width {
			lines = append(lines, current)
			current = w
			curW = ww
		} else {
			if curW > 0 {
				current += " "
				curW++
			}
			current += w
			curW += ww
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// splitWords splits text on whitespace, preserving words.
func splitWords(text string) []string {
	var words []string
	var current []rune
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// renderTitleBar draws a centered title bar with heavy borders.
func renderTitleBar(c *Canvas, y int, title string, fg Color) {
	if y < 0 || y >= c.Height {
		return
	}
	tw := DisplayWidth(title)
	start := (c.Width - tw) / 2
	if start < 0 {
		start = 0
	}
	x := start
	for _, ch := range title {
		if x >= c.Width {
			break
		}
		c.Set(y, x, ch, fg, "")
		x += RuneWidth(ch)
	}
}

// RenderKeybinds draws keybind hints right-aligned on the given row.
func RenderKeybinds(c *Canvas, row int, binds string, fg Color) {
	if row < 0 || row >= c.Height {
		return
	}
	bw := DisplayWidth(binds)
	startCol := c.Width - bw - 1
	if startCol < 0 {
		startCol = 0
	}
	x := startCol
	for _, ch := range binds {
		if x >= c.Width {
			break
		}
		c.Set(row, x, ch, fg, "")
		x += RuneWidth(ch)
	}
}

// RenderPlayerList draws a comma-separated player list left-aligned on the given row.
// Each name is sanitized before display.
func RenderPlayerList(c *Canvas, row int, names []string, fg Color) {
	if row < 0 || row >= c.Height {
		return
	}
	list := ""
	for i, n := range names {
		if i > 0 {
			list += "  "
		}
		list += SanitizeVisible(n)
	}
	list = TruncateDisplay(list, c.Width)
	x := 0
	for _, ch := range list {
		if x >= c.Width {
			break
		}
		c.Set(row, x, ch, fg, "")
		x += RuneWidth(ch)
	}
}

// SanitizeVisible strips control characters (0x00-0x1f, 0x7f) and ESC (0x1b)
// from strings that will be displayed on the terminal.
func SanitizeVisible(s string) string {
	var b []rune
	for _, r := range s {
		if r == 0x1b || r < 0x20 || r == 0x7f {
			continue
		}
		b = append(b, r)
	}
	return string(b)
}

// SanitizeUsername specifically sanitizes SSH usernames for display.
func SanitizeUsername(s string) string {
	return SanitizeVisible(s)
}
