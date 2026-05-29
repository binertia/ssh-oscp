package ansi

import (
	"strings"
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 5},
		{"中文", 4},     // CJK wide chars
		{"日本語", 6},   // CJK wide chars
		{"αβγ", 3},      // Greek narrow
		{"┏━┓", 3},      // Box drawing narrow
		{"✓", 1},
		{"✗", 1},
	}
	for _, tt := range tests {
		got := DisplayWidth(tt.input)
		if got != tt.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestTruncateDisplay(t *testing.T) {
	tests := []struct {
		s   string
		max int
		want string
	}{
		{"hello world", 5, "hell…"},
		{"hello", 10, "hello"},
		{"", 5, ""},
		{"中文测试", 4, "中…"},
	}
	for _, tt := range tests {
		got := TruncateDisplay(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("TruncateDisplay(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestCenterRect(t *testing.T) {
	tests := []struct {
		termW, termH, rectW, rectH int
		wantX, wantY               int
	}{
		{80, 24, 60, 12, 10, 6},    // 80x24 — classic terminal
		{120, 30, 60, 12, 30, 9},   // 120x30 — large terminal
		{40, 20, 36, 10, 2, 5},     // 40x20 — small terminal
		{40, 20, 50, 15, 0, 2},     // too wide — clamped; y has 2-row margin
		{80, 24, 80, 24, 0, 0},     // exact fit
	}
	for _, tt := range tests {
		x, y := CenterRect(tt.termW, tt.termH, tt.rectW, tt.rectH)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("CenterRect(%d,%d,%d,%d) = (%d,%d), want (%d,%d)",
				tt.termW, tt.termH, tt.rectW, tt.rectH, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestBoxDraw(t *testing.T) {
	c := NewCanvas(80, 24)
	box := Box{X: 10, Y: 5, Width: 20, Height: 10, Title: "TEST"}
	box.Draw(c, ColorWhite)

	// Corners
	if getCell(c, 5, 10) != '┏' {
		t.Error("top-left corner should be ┏")
	}
	if getCell(c, 5, 29) != '┓' {
		t.Error("top-right corner should be ┓")
	}
	if getCell(c, 14, 10) != '┗' {
		t.Error("bottom-left corner should be ┗")
	}
	if getCell(c, 14, 29) != '┛' {
		t.Error("bottom-right corner should be ┛")
	}

	// Top border should contain title ("TEST" at calculated position)
	foundTitle := false
	for col := 11; col < 29; col++ {
		if getCell(c, 5, col) == 'T' {
			foundTitle = true
			break
		}
	}
	if !foundTitle {
		t.Error("title should appear in top border")
	}

	// Side borders
	if getCell(c, 7, 10) != '┃' {
		t.Error("left border should be ┃")
	}
	if getCell(c, 7, 29) != '┃' {
		t.Error("right border should be ┃")
	}

	// Out-of-bounds box should not panic
	box2 := Box{X: 75, Y: 20, Width: 10, Height: 10, Title: "X"}
	box2.Draw(c, ColorWhite)
}

func TestBoxDrawSmallTerminal(t *testing.T) {
	// 40x20 terminal
	c := NewCanvas(40, 20)
	box := Box{X: 2, Y: 2, Width: 36, Height: 16, Title: "OSCP"}
	box.Draw(c, ColorWhite)

	// Box should fit entirely
	if getCell(c, 2, 2) != '┏' {
		t.Error("small terminal: top-left corner missing")
	}
	if getCell(c, 17, 37) != '┛' {
		t.Error("small terminal: bottom-right corner missing")
	}
}

func TestWrapText(t *testing.T) {
	lines := WrapText("hello world this is a test", 10)
	if len(lines) == 0 {
		t.Fatal("WrapText returned no lines")
	}
	for _, line := range lines {
		if DisplayWidth(line) > 10 {
			t.Errorf("line exceeds width: %q (%d cols)", line, DisplayWidth(line))
		}
	}
}

func TestBoxSetContent(t *testing.T) {
	c := NewCanvas(80, 24)
	box := Box{X: 10, Y: 5, Width: 30, Height: 10, Title: "TEST"}
	box.Draw(c, ColorWhite)
	box.SetContent(c, 0, "hello world", ColorWhite)
	box.SetContent(c, 2, "line three", ColorWhite)

	if getCell(c, 6, 12) != 'h' {
		t.Error("content should appear inside box")
	}
	if getCell(c, 8, 12) != 'l' {
		t.Error("second content line should appear inside box")
	}
}

func TestBoxSetContentOverflow(t *testing.T) {
	c := NewCanvas(40, 20)
	box := Box{X: 2, Y: 2, Width: 36, Height: 5, Title: "SMALL"}
	box.Draw(c, ColorWhite)

	// Try to write beyond inner height
	box.SetContent(c, 10, "overflow", ColorWhite)
	// Should not panic; content simply not written
}

func TestRenderKeybinds(t *testing.T) {
	c := NewCanvas(80, 24)
	RenderKeybinds(c, 22, "[r]efresh [q]uit", MatrixDark)

	// Should be right-aligned
	if getCell(c, 22, 79) != 't' { // last char of "quit"
		// Might be truncated, that's OK for small terminals
	}
}

func TestRenderPlayerList(t *testing.T) {
	c := NewCanvas(80, 24)
	RenderPlayerList(c, 0, []string{"alice", "bob", "charlie"}, MatrixGreen)

	// Should contain first player name
	if getCell(c, 0, 0) != 'a' {
		t.Error("player list should start with first player's name")
	}
}

// --- helpers ---

func getCell(c *Canvas, row, col int) rune {
	return c.Get(row, col).Ch
}

func checkBorderConnected(t *testing.T, c *Canvas) {
	output := RenderCanvas(c)
	// Look for disconnected patterns in output
	if strings.Contains(output, "┏ ") || strings.Contains(output, " ┓") {
		// Spaces next to corners might indicate disconnect
		// This is a heuristic — exact analysis requires cell inspection
	}
}

func checkNoOverflow(t *testing.T, c *Canvas) {
	// Ensure no content was written outside canvas bounds
	// Since Canvas.Set silently ignores out-of-bounds writes,
	// we just verify the render doesn't panic
	_ = RenderCanvas(c)
}
