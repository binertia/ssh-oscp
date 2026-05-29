package rooms

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"ssh-art/internal/ansi"
	"ssh-art/internal/memory"
)

// Star is a single star in the sky.
type Star struct {
	x, y   int
	bright bool
}

// StarRoom renders a starfield with reactive cross sculpture and quiz cards.
type StarRoom struct {
	mu      sync.Mutex
	subs    []chan string
	inputCh chan InputEvent
	quit    chan struct{}

	width   int
	height  int
	canvas  *ansi.Canvas
	stars   []Star
	tick    int
	players map[string]struct{}

	// Story engine
	storyTimer    int
	storyCooldown int
	silenceTick   int
	burstCount    int
	wasEmpty      bool

	// Quiz card
	currentQuiz  *QuizCard
	quizRevealed bool
	quizChosen   int

	// Refresh cooldown
	refreshCooldown int

	// Emotional memory
	memory   memory.RoomMemory
	roomName string
	playerEnergy map[string]int
}

// NewStarRoom creates a star room.
func NewStarRoom() *StarRoom {
	r := &StarRoom{
		subs:     make([]chan string, 0),
		inputCh:  make(chan InputEvent, 16),
		quit:     make(chan struct{}),
		width:    80,
		height:   24,
		stars:    make([]Star, 0, 80),
		players:  make(map[string]struct{}),
		roomName:     "star",
		playerEnergy: make(map[string]int),
		quizChosen:   -1,
	}
	r.memory = memory.LoadMemory(r.roomName)
	r.canvas = ansi.NewCanvas(r.width, r.height)
	for i := 0; i < 60; i++ {
		r.stars = append(r.stars, Star{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			bright: rand.Float32() < 0.5,
		})
	}
	return r
}

// Subscribe registers a new session.
// The caller must send the appropriate lifecycle event (isJoin / isSwitch) after subscribing.
func (r *StarRoom) Subscribe(playerID string) chan string {
	ch := make(chan string, 1)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.players[playerID] = struct{}{}
	r.mu.Unlock()
	return ch
}

// Unsubscribe removes a session.
func (r *StarRoom) Unsubscribe(playerID string, ch chan string) {
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
func (r *StarRoom) Input(ev InputEvent) {
	select {
	case r.inputCh <- ev:
	default:
	}
}

// Stop signals the room to shut down.
func (r *StarRoom) Stop() {
	close(r.quit)
}

func (r *StarRoom) Resize(width, height int) {
	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if width > 300 { width = 300 }
	if height > 100 { height = 100 }
	r.width = width
	r.height = height
	r.canvas = ansi.NewCanvas(width, height)
}

// Run is the room's main loop.
func (r *StarRoom) Run() {
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

func (r *StarRoom) broadcast(frame string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range r.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (r *StarRoom) triggerQuiz() {
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

func (r *StarRoom) update() {
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
	for id := range r.players {
		if r.playerEnergy[id] > 0 {
			r.playerEnergy[id]--
		}
	}
	if r.tick%600 == 0 {
		memory.SaveMemory(r.roomName, r.memory)
	}

	// Twinkle: 5% chance per star per tick
	for i := range r.stars {
		if rand.Float32() < 0.05 {
			r.stars[i].bright = !r.stars[i].bright
		}
	}

	// Slow drift right every 8 ticks
	if r.tick%8 == 0 {
		for i := range r.stars {
			r.stars[i].x++
			if r.stars[i].x >= r.width {
				r.stars[i].x = 0
				r.stars[i].y = rand.Intn(r.height)
			}
		}
	}
}

func (r *StarRoom) handleInput(ev InputEvent) {
	if ev.IsJoin {
		r.playerEnergy[ev.PlayerID] = 50
		r.silenceTick = 0
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

	// Regular keystroke — silence broken, burst detection.
	r.silenceTick = 0
		r.playerEnergy[ev.PlayerID] += 2
		if r.playerEnergy[ev.PlayerID] > 100 {
			r.playerEnergy[ev.PlayerID] = 100
		}
	r.burstCount++
	if r.burstCount >= 5 && r.storyCooldown == 0 {
		r.storyCooldown = 0
		r.triggerQuiz()
		r.burstCount = 0
	}

	// Stars react to keystrokes by spawning a brief meteor
	if len(r.stars) < 80 {
		r.stars = append(r.stars, Star{
			x:      rand.Intn(r.width),
			y:      0,
			bright: true,
		})
	}
}

func (r *StarRoom) renderFrame() string {
	r.canvas.Clear()

	// Draw stars
	for _, s := range r.stars {
		ch := '·'
		var fg ansi.Color
		if s.bright {
			ch = '*'
			fg = ansi.StarWhite
		} else {
			fg = ansi.Dim(ansi.StarDim, ansi.StarDimNight)
		}
		r.canvas.Set(s.y, s.x, ch, fg, "")
	}

	// Quiz card display
	if r.storyTimer > 0 && r.currentQuiz != nil {
		DrawQuizModal(r.canvas, r.currentQuiz, r.quizRevealed, r.quizChosen)
	}

	// Title
	title := "  ~ ENDURANCE.tel ~  "
	start := (r.width - len(title)) / 2
	for i, ch := range title {
		r.canvas.Set(1, start+i, ch, ansi.StarWhite, "")
	}

	// ANSI sculpture: reactive cross that grows with player count
	r.renderSculpture()

	// Player presence list
	r.mu.Lock()
	names := make([]string, 0, len(r.players))
	for id := range r.players {
		names = append(names, id)
	}
	r.mu.Unlock()
	ansi.RenderPlayerList(r.canvas, 0, names, ansi.Dim(ansi.StarAmber, ansi.StarAmberNight))

	// Seasonal glyph
	r.canvas.Set(0, r.width-2, ansi.SeasonalGlyph(), ansi.SeasonalAccent(), "")

	// Keybinds at bottom right
	ansi.RenderKeybinds(r.canvas, r.height-2, "[r]efresh [1-5]room [q]uit [a/b]answer", ansi.StarDim)

	frame := ansi.RenderCanvas(r.canvas)
	// Rare cosmic signal
	if rand.Float32() < 0.005 {
		frame += "\x07"
	}
	return frame
}

// renderSculpture draws a reactive cross in the center of the sky.
// Arm length grows with player count; brightness pulses every 12 ticks.
func (r *StarRoom) renderSculpture() {
	centerRow := r.height / 2
	centerCol := r.width / 2
	armLen := 2 + len(r.players)
	if armLen > 8 {
		armLen = 8
	}

	fg := ansi.Dim(ansi.StarAmber, ansi.StarAmberNight)
	if r.tick%12 < 6 {
		fg = ansi.StarWhite
	}

	// Center
	r.canvas.Set(centerRow, centerCol, '+', fg, "")
	// Arms
	for i := 1; i <= armLen; i++ {
		r.canvas.Set(centerRow-i, centerCol, '|', fg, "")
		r.canvas.Set(centerRow+i, centerCol, '|', fg, "")
		r.canvas.Set(centerRow, centerCol-i, '-', fg, "")
		r.canvas.Set(centerRow, centerCol+i, '-', fg, "")
	}
}
