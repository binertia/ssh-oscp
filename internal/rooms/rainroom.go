package rooms

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"ssh-art/internal/ansi"
	"ssh-art/internal/memory"
)

// Drop is a single rain drop.
type Drop struct {
	x, y   int
	speed  int
	bright bool
}

// RainRoom is the first room: procedural animated rain.
type RainRoom struct {
	mu      sync.Mutex
	subs    []chan string
	inputCh chan InputEvent
	quit    chan struct{}

	width           int
	height          int
	canvas          *ansi.Canvas
	drops           []Drop
	tick            int
	players         map[string]struct{}
	wallText        string
	typingIntensity int

	// Story engine
	storyTimer    int
	storyCooldown int
	silenceTick   int
	burstCount    int
	wasEmpty      bool

	// Emotional memory
	memory   memory.RoomMemory
	roomName string

	// Per-session emotional energy (0-100)
	playerEnergy map[string]int

	// Quiz state
	currentQuiz  *QuizCard
	quizRevealed bool
	quizChosen   int

	refreshCooldown int
}

// NewRainRoom creates and seeds a rain room.
func NewRainRoom() *RainRoom {
	r := &RainRoom{
		subs:     make([]chan string, 0),
		inputCh:  make(chan InputEvent, 16),
		quit:     make(chan struct{}),
		width:    80,
		height:   24,
		drops:        make([]Drop, 0, 120),
		players:      make(map[string]struct{}),
		roomName:     "rain",
		playerEnergy: make(map[string]int),
		quizChosen:   -1,
	}
	r.memory = memory.LoadMemory(r.roomName)
	r.canvas = ansi.NewCanvas(r.width, r.height)
	for i := 0; i < 50; i++ {
		r.drops = append(r.drops, Drop{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			speed:  1 + rand.Intn(2),
			bright: rand.Float32() < 0.3,
		})
	}
	return r
}

// Subscribe registers a new session to receive rendered frames.
// The caller must send the appropriate lifecycle event (isJoin / isSwitch) after subscribing.
func (r *RainRoom) Subscribe(playerID string) chan string {
	ch := make(chan string, 1)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.players[playerID] = struct{}{}
	r.mu.Unlock()
	return ch
}

// Unsubscribe removes a session and closes its channel.
func (r *RainRoom) Unsubscribe(playerID string, ch chan string) {
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

// Input forwards a keystroke to the room. Non-blocking.
func (r *RainRoom) Input(ev InputEvent) {
	select {
	case r.inputCh <- ev:
	default:
	}
}

// Stop signals the room to shut down.
func (r *RainRoom) Stop() {
	close(r.quit)
}

func (r *RainRoom) Resize(width, height int) {
	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if width > 300 { width = 300 }
	if height > 100 { height = 100 }
	r.width = width
	r.height = height
	r.canvas = ansi.NewCanvas(width, height)
}

// Run is the room's main loop. Must be called in its own goroutine.
func (r *RainRoom) Run() {
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

func (r *RainRoom) broadcast(frame string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range r.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (r *RainRoom) update() {
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

	// Emotional climate decay and wall text fade
	occupied := len(r.players) > 0
	r.memory.DecayClimate(occupied)
	if !occupied && r.tick%10 == 0 && len(r.wallText) > 0 {
		r.wallText = r.wallText[:len(r.wallText)-1]
	}
	if r.tick%600 == 0 {
		memory.SaveMemory(r.roomName, r.memory)
	}
	// Per-session energy decay
	for id := range r.players {
		if r.playerEnergy[id] > 0 {
			r.playerEnergy[id]--
		}
	}

	// Move drops down
	for i := range r.drops {
		r.drops[i].y += r.drops[i].speed
		if r.drops[i].y >= r.height {
			r.drops[i].y = 0
			r.drops[i].x = rand.Intn(r.width)
			r.drops[i].speed = 1 + rand.Intn(2)
			r.drops[i].bright = rand.Float32() < 0.3
		}
	}

	// Spawn drops — typing intensity increases rate, speed, and brightness.
	spawnChance := 0.3 + float32(r.typingIntensity)*0.02
	if spawnChance > 1.0 {
		spawnChance = 1.0
	}
	if rand.Float32() < spawnChance && len(r.drops) < 120 {
		speed := 1 + rand.Intn(2)
		if r.typingIntensity > 20 {
			speed = 1 + rand.Intn(3)
		}
		r.drops = append(r.drops, Drop{
			x:      rand.Intn(r.width),
			y:      0,
			speed:  speed,
			bright: rand.Float32() < 0.3+float32(r.typingIntensity)*0.01,
		})
	}

	// Decay typing intensity
	if r.typingIntensity > 0 {
		r.typingIntensity--
	}
}

func (r *RainRoom) triggerQuiz() {
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

func (r *RainRoom) handleInput(ev InputEvent) {
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
			if r.wallText != "" {
				r.memory.LastWords = append([]string{r.wallText}, r.memory.LastWords...)
				if len(r.memory.LastWords) > 3 {
					r.memory.LastWords = r.memory.LastWords[:3]
				}
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

	// Typing activity resets silence timer and increases intensity.
	r.silenceTick = 0
	r.playerEnergy[ev.PlayerID] += 2
	if r.playerEnergy[ev.PlayerID] > 100 {
		r.playerEnergy[ev.PlayerID] = 100
	}

	r.typingIntensity += 2
	if r.typingIntensity > 50 {
		r.typingIntensity = 50
	}
	if r.typingIntensity > 20 {
		r.burstCount++
		if r.burstCount >= 3 && r.storyCooldown == 0 {
			r.storyCooldown = 0
		r.triggerQuiz()
			r.burstCount = 0
		}
	}

	// Any keystroke spawns a bright burst of rain.
	for i := 0; i < 5 && len(r.drops) < 120; i++ {
		r.drops = append(r.drops, Drop{
			x:      rand.Intn(r.width),
			y:      0,
			speed:  1 + rand.Intn(3),
			bright: true,
		})
	}

	// Collaborative wall text: printable ASCII appends, DEL/BS deletes.
	for _, b := range ev.Data {
		switch b {
		case 127, 8: // DEL / Backspace
			if len(r.wallText) > 0 {
				r.wallText = r.wallText[:len(r.wallText)-1]
			}
		case 27: // Escape — ignore the rest of this input packet
			return
		case '\r', '\n':
			// Enter currently ignored; future: submit to history
		default:
			if b >= 32 && b < 127 && len(r.wallText) < 60 {
				r.wallText += string(b)
			}
		}
	}
}

func (r *RainRoom) renderFrame() string {
	r.canvas.Clear()

	// Draw rain
	for _, d := range r.drops {
		ch := '1'
		if d.speed == 1 {
			ch = '│'
		} else if d.speed == 3 {
			ch = '0'
		}
		var fg ansi.Color
		if d.bright {
			fg = ansi.MatrixBright
		} else {
			fg = ansi.MatrixGreen
		}
		r.canvas.Set(d.y, d.x, ch, fg, "")

		// Trail — residual data
		if d.y > 0 {
			r.canvas.Set(d.y-1, d.x, '·', ansi.Dim(ansi.MatrixDeep, ansi.MatrixDeepNight), "")
		}
		if d.y > 1 && d.speed > 1 {
			r.canvas.Set(d.y-2, d.x, '.', ansi.Dim(ansi.MatrixDark, ansi.MatrixDarkNight), "")
		}
	}

	// Quiz card display
	if r.storyTimer > 0 && r.currentQuiz != nil {
		DrawQuizModal(r.canvas, r.currentQuiz, r.quizRevealed, r.quizChosen)
	}

	// Title
	title := "  ~ CASCADEnode.07 ~  "
	start := (r.width - len(title)) / 2
	for i, ch := range title {
		r.canvas.Set(1, start+i, ch, ansi.MatrixBright, "")
	}

	// Player presence list (top row)
	r.mu.Lock()
	names := make([]string, 0, len(r.players))
	for id := range r.players {
		names = append(names, id)
	}
	r.mu.Unlock()
	ansi.RenderPlayerList(r.canvas, 0, names, ansi.Dim(ansi.MatrixGreen, ansi.MatrixGreenNight))

	// Seasonal glyph
	r.canvas.Set(0, r.width-2, ansi.SeasonalGlyph(), ansi.SeasonalAccent(), "")

	// Collaborative wall text (bottom row)
	wallSafe := ansi.SanitizeVisible(r.wallText)
	prompt := "> " + wallSafe
	if len(prompt) > r.width {
		prompt = prompt[:r.width]
	}
	for i, ch := range prompt {
		r.canvas.Set(r.height-1, i, ch, ansi.MatrixBright, "")
	}

	// Keybinds at bottom right
	ansi.RenderKeybinds(r.canvas, r.height-2, "[r]efresh [1-5]room [q]uit [a/b]answer", ansi.MatrixDark)

	frame := ansi.RenderCanvas(r.canvas)
	// Ambient bell: intensity increases chance of a bell
	var bellChance float32 = 0.01
	if r.typingIntensity > 20 {
		bellChance = 0.03
	}
	if r.typingIntensity > 40 {
		bellChance = 0.05
	}
	if rand.Float32() < bellChance {
		frame += "\x07"
	}

	return frame
}


