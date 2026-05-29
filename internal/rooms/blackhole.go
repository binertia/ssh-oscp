package rooms

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"ssh-art/internal/ansi"
	"ssh-art/internal/memory"
)

// Matter is a particle spiraling into the singularity.
type Matter struct {
	x, y   float64
	angle  float64
	radius float64
	speed  float64
	bright bool
}

// BlackholeRoom — the Gargantua singularity.
type BlackholeRoom struct {
	mu      sync.Mutex
	subs    []chan string
	inputCh chan InputEvent
	quit    chan struct{}

	width      int
	height     int
	canvas     *ansi.Canvas
	matter     []Matter
	tick       int
	players map[string]struct{}

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

	// Refresh cooldown
	refreshCooldown int

	// Emotional memory
	memory   memory.RoomMemory
	roomName string

	// Per-session emotional energy (0-100)
	playerEnergy map[string]int
}

// NewBlackholeRoom creates the singularity room.
func NewBlackholeRoom() *BlackholeRoom {
	r := &BlackholeRoom{
		subs:         make([]chan string, 0),
		inputCh:      make(chan InputEvent, 16),
		quit:         make(chan struct{}),
		width:        80,
		height:       24,
		matter:       make([]Matter, 0, 120),
		players:      make(map[string]struct{}),
		roomName:     "blackhole",
		playerEnergy: make(map[string]int),
	}
	r.memory = memory.LoadMemory(r.roomName)
	r.canvas = ansi.NewCanvas(r.width, r.height)
	r.quizChosen = -1

	// Seed infalling matter
	for i := 0; i < 60; i++ {
		angle := rand.Float64() * 2 * math.Pi
		radius := 10 + rand.Float64()*25
		r.matter = append(r.matter, Matter{
			x:      0,
			y:      0,
			angle:  angle,
			radius: radius,
			speed:  0.02 + rand.Float64()*0.04,
			bright: rand.Float64() < 0.3,
		})
	}

	return r
}

func (r *BlackholeRoom) Resize(width, height int) {
	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if width > 300 { width = 300 }
	if height > 100 { height = 100 }
	r.width = width
	r.height = height
	r.canvas = ansi.NewCanvas(width, height)
}

// Subscribe registers a new session.
func (r *BlackholeRoom) Subscribe(playerID string) chan string {
	ch := make(chan string, 1)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.players[playerID] = struct{}{}
	r.mu.Unlock()
	return ch
}

// Unsubscribe removes a session.
func (r *BlackholeRoom) Unsubscribe(playerID string, ch chan string) {
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
func (r *BlackholeRoom) Input(ev InputEvent) {
	select {
	case r.inputCh <- ev:
	default:
	}
}

// Stop signals the room to shut down.
func (r *BlackholeRoom) Stop() {
	close(r.quit)
}

// Run is the room's main loop.
func (r *BlackholeRoom) Run() {
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

func (r *BlackholeRoom) broadcast(frame string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range r.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (r *BlackholeRoom) triggerQuiz() {
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

func (r *BlackholeRoom) update() {
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

	centerX := float64(r.width) / 2
	centerY := float64(r.height) / 2

	// Spiral matter inward
	for i := range r.matter {
		m := &r.matter[i]
		m.angle += m.speed * 2
		m.radius -= m.speed * 3

		// Heat up as it falls
		if m.radius < 8 {
			m.bright = true
		}
		if m.radius < 4 {
			m.bright = rand.Float64() < 0.7
		}

		// Respawn at edge if consumed
		if m.radius <= 1.5 {
			m.angle = rand.Float64() * 2 * math.Pi
			m.radius = 15 + rand.Float64()*20
			m.speed = 0.02 + rand.Float64()*0.05
			m.bright = rand.Float64() < 0.2
		}

		m.x = centerX + m.radius*math.Cos(m.angle)
		m.y = centerY + m.radius*math.Sin(m.angle)*0.35 // flattened for perspective
	}

	// Spawn extra matter during bursts
	if len(r.matter) < 100 && rand.Float32() < 0.15 {
		angle := rand.Float64() * 2 * math.Pi
		radius := 20 + rand.Float64()*15
		r.matter = append(r.matter, Matter{
			angle:  angle,
			radius: radius,
			speed:  0.02 + rand.Float64()*0.04,
			bright: rand.Float64() < 0.2,
		})
	}

}

func (r *BlackholeRoom) handleInput(ev InputEvent) {
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

	// Keystrokes spawn extra infalling matter
	for i := 0; i < 3 && len(r.matter) < 120; i++ {
		angle := rand.Float64() * 2 * math.Pi
		radius := 18 + rand.Float64()*15
		r.matter = append(r.matter, Matter{
			angle:  angle,
			radius: radius,
			speed:  0.03 + rand.Float64()*0.05,
			bright: true,
		})
	}
}

func (r *BlackholeRoom) renderFrame() string {
	r.canvas.Clear()

	centerX := float64(r.width) / 2
	centerY := float64(r.height) / 2

	// Draw accretion disk — bright horizontal band
	diskRows := []int{10, 11, 12, 13, 14}
	for _, row := range diskRows {
		offset := float64(row) - centerY
		diskWidth := 30 - int(math.Abs(offset)*8)
		if diskWidth < 8 {
			diskWidth = 8
		}
		startCol := int(centerX) - diskWidth/2
		for col := 0; col < diskWidth; col++ {
			// Doppler beaming: brighter on left (approaching)
			var fg ansi.Color
			if col < diskWidth/2 {
				fg = ansi.BHBright
			} else {
				fg = ansi.BHAccretion
			}
			if rand.Float32() < 0.3 {
				fg = ansi.BHGlow
			}
			ch := '·'
			if rand.Float32() < 0.2 {
				ch = ':'
			}
			if rand.Float32() < 0.05 {
				ch = '*'
			}
			r.canvas.Set(row, startCol+col, ch, fg, "")
		}
	}

	// Draw matter spiraling in
	for _, m := range r.matter {
		col := int(m.x)
		row := int(m.y)
		if col < 0 || col >= r.width || row < 0 || row >= r.height {
			continue
		}
		ch := '·'
		var fg ansi.Color
		if m.bright {
			ch = '+'
			fg = ansi.BHBright
		} else if m.radius < 8 {
			ch = ':'
			fg = ansi.BHAccretion
		} else {
			fg = ansi.Dim(ansi.BHDim, ansi.BHDimNight)
		}
		r.canvas.Set(row, col, ch, fg, "")
	}

	// Central void (event horizon) — leave empty, the black background does the work
	// Just draw a faint ring
	voidRadius := 2.5
	for angle := 0.0; angle < 2*math.Pi; angle += 0.15 {
		vx := int(centerX + voidRadius*math.Cos(angle))
		vy := int(centerY + voidRadius*math.Sin(angle)*0.35)
		if vx >= 0 && vx < r.width && vy >= 0 && vy < r.height {
			r.canvas.Set(vy, vx, '·', ansi.Dim(ansi.BHGlow, ansi.BHGlowNight), "")
		}
	}

	// Quiz card display
	if r.storyTimer > 0 && r.currentQuiz != nil {
		DrawQuizModal(r.canvas, r.currentQuiz, r.quizRevealed, r.quizChosen)
	}

	// Title
	title := "  ~ GARGANTUA.evt ~  "
	start := (r.width - len(title)) / 2
	for i, ch := range title {
		r.canvas.Set(1, start+i, ch, ansi.BHBright, "")
	}

	// Player presence list (top row)
	r.mu.Lock()
	names := make([]string, 0, len(r.players))
	for id := range r.players {
		names = append(names, id)
	}
	r.mu.Unlock()
	ansi.RenderPlayerList(r.canvas, 0, names, ansi.Dim(ansi.BHGlow, ansi.BHGlowNight))

	r.canvas.Set(0, r.width-2, ansi.SeasonalGlyph(), ansi.SeasonalAccent(), "")

	// Keybinds at bottom right
	ansi.RenderKeybinds(r.canvas, r.height-2, "[r]efresh [1-5]room [q]uit [a/b]answer", ansi.BHDim)

	frame := ansi.RenderCanvas(r.canvas)
	// Rare gravitational wave ping
	if rand.Float32() < 0.005 {
		frame += "\x07"
	}
	return frame
}



