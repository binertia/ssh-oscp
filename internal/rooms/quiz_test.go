package rooms

import (
	"testing"

	"ssh-art/internal/ansi"
)

func TestQuizModal80x24(t *testing.T) {
	c := ansi.NewCanvas(80, 24)
	card := &QuizCard{
		ID:       1,
		Question: "What nmap flag performs a TCP SYN stealth scan?",
		Choices:  [2]string{"-sS", "-sT"},
		Correct:  0,
	}
	DrawQuizModal(c, card, false, -1)

	// Modal should be visible and centered
	foundCorner := false
	for row := 0; row < c.Height; row++ {
		for col := 0; col < c.Width; col++ {
			if c.Get(row, col).Ch == '┏' {
				foundCorner = true
			}
		}
	}
	if !foundCorner {
		t.Error("modal should have a visible top-left corner on 80x24")
	}
}

func TestQuizModal120x30(t *testing.T) {
	c := ansi.NewCanvas(120, 30)
	card := &QuizCard{
		ID:       1,
		Question: "What nmap flag performs a TCP SYN stealth scan?",
		Choices:  [2]string{"-sS", "-sT"},
		Correct:  0,
	}
	DrawQuizModal(c, card, false, -1)

	// On large terminals, box should be wider
	foundCorner := false
	for row := 0; row < c.Height; row++ {
		for col := 0; col < c.Width; col++ {
			if c.Get(row, col).Ch == '┏' {
				foundCorner = true
			}
		}
	}
	if !foundCorner {
		t.Error("modal should be visible on 120x30")
	}
}

func TestQuizModal40x20(t *testing.T) {
	c := ansi.NewCanvas(40, 20)
	card := &QuizCard{
		ID:       1,
		Question: "What nmap flag performs a TCP SYN stealth scan?",
		Choices:  [2]string{"-sS", "-sT"},
		Correct:  0,
	}
	DrawQuizModal(c, card, false, -1)

	// On small terminals, modal should still fit
	foundCorner := false
	for row := 0; row < c.Height; row++ {
		for col := 0; col < c.Width; col++ {
			if c.Get(row, col).Ch == '┏' {
				foundCorner = true
			}
		}
	}
	if !foundCorner {
		t.Error("modal should be visible and fit on 40x20")
	}

	// Ensure no panic on render
	_ = ansi.RenderCanvas(c)
}

func TestQuizModalRevealed(t *testing.T) {
	c := ansi.NewCanvas(80, 24)
	card := &QuizCard{
		ID:          1,
		Question:    "What nmap flag performs a TCP SYN stealth scan?",
		Choices:     [2]string{"-sS", "-sT"},
		Correct:     0,
		Explanation: "-sS is the default SYN stealth scan.",
	}
	DrawQuizModal(c, card, true, 0)

	// Should show correct mark
	found := false
	for row := 0; row < c.Height; row++ {
		for col := 0; col < c.Width; col++ {
			if c.Get(row, col).Ch == '✓' {
				found = true
			}
		}
	}
	if !found {
		t.Error("revealed modal should show ✓ mark")
	}
}

func TestQuizModalTinyTerminal(t *testing.T) {
	// Below minimum size — should gracefully skip rendering
	c := ansi.NewCanvas(20, 8)
	card := &QuizCard{
		ID:       1,
		Question: "What?",
		Choices:  [2]string{"A", "B"},
		Correct:  0,
	}
	DrawQuizModal(c, card, false, -1)

	// Should not panic; modal may or may not render depending on clamping
	_ = ansi.RenderCanvas(c)
}
