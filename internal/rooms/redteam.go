package rooms

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"ssh-art/internal/ansi"
	"ssh-art/internal/memory"
)

// Glyph is a floating code symbol in the terminal.
type Glyph struct {
	x, y   int
	dx, dy int
	ch     rune
	bright bool
}

// RedteamRoom — cyberpunk fantasy red team pentest.
type RedteamRoom struct {
	mu      sync.Mutex
	subs    []chan string
	inputCh chan InputEvent
	quit    chan struct{}

	width      int
	height     int
	canvas     *ansi.Canvas
	glyphs     []Glyph
	tick       int
	players    map[string]struct{}

	// Story engine
	storyTimer    int
	storyCooldown int
	silenceTick   int
	burstCount    int
	wasEmpty      bool

	// Quiz card game
	currentQuiz  *QuizCard
	quizRevealed bool
	quizChosen   int

	refreshCooldown int

	// Emotional memory
	memory       memory.RoomMemory
	roomName     string
	playerEnergy map[string]int
}

// NewRedteamRoom creates the red team pentest room.
func NewRedteamRoom() *RedteamRoom {
	r := &RedteamRoom{
		subs:         make([]chan string, 0),
		inputCh:      make(chan InputEvent, 16),
		quit:         make(chan struct{}),
		width:        80,
		height:       24,
		glyphs:       make([]Glyph, 0, 80),
		players:      make(map[string]struct{}),
		roomName:     "redteam",
		playerEnergy: make(map[string]int),
	}
	r.memory = memory.LoadMemory(r.roomName)
	r.canvas = ansi.NewCanvas(r.width, r.height)

	runeChars := []rune("ᚠᚢᚦᚨᚱᚲᚷᚹᚺᚾᛁᛃᛇᛈᛉᛊᛏᛒᛖᛗᛚᛜᛞ<>{ }[]; &|%$#!?0xAF")
	for i := 0; i < 60; i++ {
		r.glyphs = append(r.glyphs, Glyph{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			dx:     rand.Intn(3) - 1,
			dy:     rand.Intn(3) - 1,
			ch:     runeChars[rand.Intn(len(runeChars))],
			bright: rand.Float32() < 0.3,
		})
	}

	r.quizChosen = -1
	return r
}

// Subscribe registers a new session.
func (r *RedteamRoom) Subscribe(playerID string) chan string {
	ch := make(chan string, 1)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.players[playerID] = struct{}{}
	r.mu.Unlock()
	return ch
}

// Unsubscribe removes a session.
func (r *RedteamRoom) Unsubscribe(playerID string, ch chan string) {
	r.mu.Lock()
	delete(r.players, playerID)
	for i, s := range r.subs {
		if s == ch {
			r.subs = append(r.subs[:i], r.subs[i+1:]...)
			break
		}
	}
	r.mu.Unlock()
	close(ch)
	r.Input(InputEvent{PlayerID: playerID, IsLeave: true})
}

// Input forwards a keystroke. Non-blocking.
func (r *RedteamRoom) Input(ev InputEvent) {
	select {
	case r.inputCh <- ev:
	default:
	}
}

// Stop signals the room to shut down.
func (r *RedteamRoom) Stop() {
	close(r.quit)
}

// Run is the room's main loop.
func (r *RedteamRoom) Run() {
	ticker := time.NewTicker(time.Second / 12)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.update()
			r.broadcast(r.renderFrame())
		case ev := <-r.inputCh:
			r.handleInput(ev)
			r.broadcast(r.renderFrame())
		case <-r.quit:
			return
		}
	}
}

func (r *RedteamRoom) broadcast(frame string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range r.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (r *RedteamRoom) triggerQuiz() {
	if r.storyCooldown > 0 {
		return
	}
	r.currentQuiz = PickRandomQuiz()
	r.quizRevealed = false
	r.quizChosen = -1
	r.storyTimer = 600
	r.storyCooldown = 1800
	if r.currentQuiz != nil {
		r.memory.RecordStory(fmt.Sprintf("quiz #%d", r.currentQuiz.ID))
	}
}

func (r *RedteamRoom) update() {
	r.tick++

	if r.storyTimer > 0 {
		r.storyTimer--
	}
	if r.storyCooldown > 0 {
		r.storyCooldown--
	}
	if r.refreshCooldown > 0 {
		r.refreshCooldown--
	}
	r.silenceTick++
	if r.silenceTick == 360 && r.storyCooldown == 0 {
		r.storyCooldown = 0
		r.triggerQuiz()
	}
	if r.burstCount > 0 && r.tick%10 == 0 {
		r.burstCount--
	}

	r.memory.DecayClimate(len(r.players) > 0)
	if r.tick%600 == 0 {
		memory.SaveMemory(r.roomName, r.memory)
	}
	for id := range r.players {
		if r.playerEnergy[id] > 0 {
			r.playerEnergy[id]--
		}
	}

	// Drift glyphs
	for i := range r.glyphs {
		g := &r.glyphs[i]
		g.x += g.dx
		g.y += g.dy
		if g.x < 0 {
			g.x = r.width - 1
		}
		if g.x >= r.width {
			g.x = 0
		}
		if g.y < 0 {
			g.y = r.height - 1
		}
		if g.y >= r.height {
			g.y = 0
		}
		// Occasionally swap character
		if rand.Float32() < 0.02 {
			runeChars := []rune("ᚠᚢᚦᚨᚱᚲᚷᚹᚺᚾᛁᛃᛇᛈᛉᛊᛏᛒᛖᛗᛚᛜᛞ<>{ }[]; &|%$#!?0xAF")
			g.ch = runeChars[rand.Intn(len(runeChars))]
		}
	}

}

func (r *RedteamRoom) handleInput(ev InputEvent) {
	if ev.IsJoin {
		r.silenceTick = 0
		r.playerEnergy[ev.PlayerID] = 50
		r.memory.TotalVisits++
		r.memory.LastActivityAt = time.Now()
		n := len(r.players)
		if n > r.memory.PeakPlayers {
			r.memory.PeakPlayers = n
		}
		r.memory.EmotionalClimate.Intensity += 5
		if r.memory.EmotionalClimate.Intensity > 100 {
			r.memory.EmotionalClimate.Intensity = 100
		}
		if r.storyCooldown == 0 {
			if n == 1 {
				r.storyCooldown = 0
		r.triggerQuiz()
			} else if n >= 2 {
				r.storyCooldown = 0
		r.triggerQuiz()
			}
			if r.wasEmpty {
				r.storyCooldown = 0
		r.triggerQuiz()
				r.wasEmpty = false
			}
			hour := time.Now().Hour()
			if hour >= 0 && hour < 5 {
				r.storyCooldown = 0
		r.triggerQuiz()
			}
			if rand.Float32() < 0.3 {
				r.storyCooldown = 0
		r.triggerQuiz()
			}
		}
		return
	}

	if ev.IsSwitch {
		r.silenceTick = 0
		r.memory.EmotionalClimate.Chaos += 3
		if r.memory.EmotionalClimate.Chaos > 100 {
			r.memory.EmotionalClimate.Chaos = 100
		}
		if r.storyCooldown == 0 {
			r.storyCooldown = 0
		r.triggerQuiz()
			hour := time.Now().Hour()
			if hour >= 0 && hour < 5 {
				r.storyCooldown = 0
		r.triggerQuiz()
			}
		}
		return
	}

	if ev.IsLeave {
		if len(r.players) == 0 {
			r.wasEmpty = true
			r.memory.LastEmptyAt = time.Now()
			r.memory.EmotionalClimate.Loneliness += 10
			if r.memory.EmotionalClimate.Loneliness > 100 {
				r.memory.EmotionalClimate.Loneliness = 100
			}
			memory.SaveMemory(r.roomName, r.memory)
		}
		return
	}

	// Refresh quiz on demand
	if len(ev.Data) == 1 && ev.Data[0] == 'r' {
		if r.refreshCooldown == 0 {
			r.storyCooldown = 0
			r.triggerQuiz()
			r.refreshCooldown = 12 // 1 second at 12 FPS
		}
		return
	}

	// Quiz answer
	if r.currentQuiz != nil && !r.quizRevealed {
		switch {
		case len(ev.Data) == 1 && ev.Data[0] == 'a':
			r.quizChosen = 0
			r.quizRevealed = true
			r.storyTimer = 180
		case len(ev.Data) == 1 && ev.Data[0] == 'b':
			r.quizChosen = 1
			r.quizRevealed = true
			r.storyTimer = 180
		}
	}

	// Regular keystroke
	r.silenceTick = 0
	if r.playerEnergy[ev.PlayerID] < 100 {
		r.playerEnergy[ev.PlayerID] += 2
		if r.playerEnergy[ev.PlayerID] > 100 {
			r.playerEnergy[ev.PlayerID] = 100
		}
	}
	r.burstCount++
	if r.burstCount >= 5 && r.storyCooldown == 0 {
		r.storyCooldown = 0
		r.triggerQuiz()
		r.burstCount = 0
	}

	// Keystrokes spawn bright exploit glyphs
	runeChars := []rune("ᚠᚢᚦᚨᚱᚲᚷᚹᚺᚾᛁᛃᛇᛈᛉᛊᛏᛒᛖᛗᛚᛜᛞ<>{ }[]; &|%$#!?0xAF")
	for i := 0; i < 4 && len(r.glyphs) < 100; i++ {
		r.glyphs = append(r.glyphs, Glyph{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			dx:     rand.Intn(3) - 1,
			dy:     rand.Intn(3) - 1,
			ch:     runeChars[rand.Intn(len(runeChars))],
			bright: true,
		})
	}
}

func (r *RedteamRoom) renderFrame() string {
	r.canvas.Clear()

	// Draw glyphs
	for _, g := range r.glyphs {
		var fg ansi.Color
		if g.bright {
			fg = ansi.ArcaneRed
		} else {
			fg = ansi.Dim(ansi.ArcaneDim, ansi.ArcaneDimNight)
		}
		r.canvas.Set(g.y, g.x, g.ch, fg, "")
	}

	// Draw central hexagram sigil
	r.renderSigil()

	// Quiz card display
	if r.storyTimer > 0 && r.currentQuiz != nil {
		DrawQuizModal(r.canvas, r.currentQuiz, r.quizRevealed, r.quizChosen)
	}

	// Title
	title := "  ~ HEXSHELL.pwn ~  "
	start := (r.width - len(title)) / 2
	for i, ch := range title {
		r.canvas.Set(1, start+i, ch, ansi.Dim(ansi.ArcaneGold, ansi.ArcaneGoldNight), "")
	}

	// Player presence list (top row)
	r.mu.Lock()
	names := make([]string, 0, len(r.players))
	for id := range r.players {
		names = append(names, id)
	}
	r.mu.Unlock()
	ansi.RenderPlayerList(r.canvas, 0, names, ansi.Dim(ansi.ArcaneGold, ansi.ArcaneGoldNight))

	r.canvas.Set(0, r.width-2, ansi.SeasonalGlyph(), ansi.SeasonalAccent(), "")

	// Keybinds at bottom right
	ansi.RenderKeybinds(r.canvas, r.height-2, "[r]efresh [1-5]room [q]uit [a/b]answer", ansi.ArcaneDim)

	frame := ansi.RenderCanvas(r.canvas)
	// Rare exploit trigger ping
	if rand.Float32() < 0.005 {
		frame += "\x07"
	}
	return frame
}

func (r *RedteamRoom) Resize(width, height int) {
	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if width > 300 { width = 300 }
	if height > 100 { height = 100 }
	r.width = width
	r.height = height
	r.canvas = ansi.NewCanvas(width, height)
}

// renderSigil draws a pulsing hexagram in the center.
// Size grows with player count; brightness pulses every 12 ticks.
func (r *RedteamRoom) renderSigil() {
	centerRow := r.height / 2
	centerCol := r.width / 2
	armLen := 2 + len(r.players)
	if armLen > 6 {
		armLen = 6
	}

	fg := ansi.ArcanePink
	if r.tick%12 < 6 {
		fg = ansi.ArcaneRed
	}

	// Upper triangle
	for i := 0; i < armLen; i++ {
		r.canvas.Set(centerRow-armLen+i, centerCol-i, '/', fg, "")
		r.canvas.Set(centerRow-armLen+i, centerCol+i, '\\', fg, "")
	}
	// Horizontal bar
	for i := -armLen; i <= armLen; i++ {
		r.canvas.Set(centerRow, centerCol+i, '-', fg, "")
	}
	// Lower triangle
	for i := 0; i < armLen; i++ {
		r.canvas.Set(centerRow+i, centerCol-armLen+i, '\\', fg, "")
		r.canvas.Set(centerRow+i, centerCol+armLen-i, '/', fg, "")
	}
}



