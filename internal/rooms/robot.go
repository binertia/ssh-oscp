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

// Spark is a drifting interference particle in front of the robot.
type Spark struct {
	x, y   int
	speed  int
	bright bool
	ch     rune
}

// RobotLight is a blinking indicator on the robot's body.
type RobotLight struct {
	row, col int
	phase    int
}

// RobotRoom — animated robot simulation with lights, sparks, and quiz cards.
type RobotRoom struct {
	mu      sync.Mutex
	subs    []chan string
	inputCh chan InputEvent
	quit    chan struct{}

	width      int
	height     int
	canvas     *ansi.Canvas
	sparks     []Spark
	lights     []RobotLight
	tick       int
	players    map[string]struct{}
	// Robot animation state
	breathOffset int
	eyeOpen      bool
	eyeTimer     int
	coreBright   bool
	coreTimer    int
	glitchActive bool
	glitchTimer  int

	// Story engine
	storyTimer    int
	storyCooldown int
	silenceTick   int
	burstCount    int
	wasEmpty      bool

	// Quiz
	currentQuiz  *QuizCard
	quizRevealed bool
	quizChosen   int

	refreshCooldown int

	// Emotional memory
	memory   memory.RoomMemory
	roomName string
	playerEnergy map[string]int
}

var robotTemplate = []string{
	"      ▓▓▓▓▓▓▓▓▓▓▓      ",
	"    ▓▓           ▓▓    ",
	"   ▓   ◯       ◯   ▓   ",
	"   ▓       ▽       ▓   ",
	"    ▓   ▓▓▓▓▓▓▓   ▓    ",
	"     ▓▓▓▓▓▓▓▓▓▓▓▓▓     ",
	"       ▓▓▓▓▓▓▓▓▓       ",
	"      ▓▓▓▓▓▓▓▓▓▓▓      ",
	"     ▓▓▓▓▓▓▓▓▓▓▓▓▓     ",
	"    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    ",
	"    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    ",
	"     ▓▓▓▓▓▓▓▓▓▓▓▓▓     ",
	"      ▓▓▓▓▓▓▓▓▓▓▓      ",
	"       ▓▓▓▓▓▓▓▓▓       ",
	"        ▓▓▓▓▓▓▓        ",
	"       ▓▓     ▓▓       ",
	"      ▓▓       ▓▓      ",
	"     ▓▓         ▓▓     ",
	"    ▓▓           ▓▓    ",
}

var robotLightPositions = []struct{ row, col int }{
	{5, 8}, {5, 16}, {7, 6}, {7, 18}, {9, 12}, {13, 10}, {13, 14}, {15, 8}, {15, 16},
}

// NewRobotRoom creates the robot room.
func NewRobotRoom() *RobotRoom {
	r := &RobotRoom{
		subs:        make([]chan string, 0),
		inputCh:     make(chan InputEvent, 16),
		quit:        make(chan struct{}),
		width:       80,
		height:      24,
		sparks:      make([]Spark, 0, 80),
		lights:      make([]RobotLight, 0, len(robotLightPositions)),
		players:      make(map[string]struct{}),
		playerEnergy: make(map[string]int),
		roomName:     "robot",
		eyeOpen:     true,
		eyeTimer:    180 + rand.Intn(180),
		coreTimer:   30,
		glitchTimer: 300 + rand.Intn(300),
		quizChosen:  -1,
	}
	r.memory = memory.LoadMemory(r.roomName)
	r.canvas = ansi.NewCanvas(r.width, r.height)

	for i := 0; i < 50; i++ {
		r.sparks = append(r.sparks, Spark{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			speed:  1 + rand.Intn(2),
			bright: rand.Float32() < 0.2,
			ch:     sparkChar(),
		})
	}

	for _, pos := range robotLightPositions {
		r.lights = append(r.lights, RobotLight{
			row:   pos.row,
			col:   pos.col,
			phase: rand.Intn(120),
		})
	}

	return r
}

func sparkChar() rune {
	chars := []rune{'|', '░', '▒', '·', ':', '│'}
	return chars[rand.Intn(len(chars))]
}

// Subscribe registers a new session.
func (r *RobotRoom) Subscribe(playerID string) chan string {
	ch := make(chan string, 1)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.players[playerID] = struct{}{}
	r.mu.Unlock()
	return ch
}

// Unsubscribe removes a session.
func (r *RobotRoom) Unsubscribe(playerID string, ch chan string) {
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
func (r *RobotRoom) Input(ev InputEvent) {
	select {
	case r.inputCh <- ev:
	default:
	}
}

// Stop signals the room to shut down.
func (r *RobotRoom) Stop() {
	close(r.quit)
}

// Resize resizes the room canvas.
func (r *RobotRoom) Resize(width, height int) {
	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if width > 300 { width = 300 }
	if height > 100 { height = 100 }
	r.width = width
	r.height = height
	r.canvas = ansi.NewCanvas(width, height)
}

// Run is the room's main loop.
func (r *RobotRoom) Run() {
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

func (r *RobotRoom) broadcast(frame string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range r.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (r *RobotRoom) triggerQuiz() {
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

func (r *RobotRoom) update() {
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

	// Breathing — slow sine wave offset
	r.breathOffset = int(math.Sin(float64(r.tick)*0.05) * 1.5)

	// Eye blink
	r.eyeTimer--
	if r.eyeTimer <= 0 {
		r.eyeOpen = !r.eyeOpen
		if r.eyeOpen {
			r.eyeTimer = 180 + rand.Intn(180)
		} else {
			r.eyeTimer = 5 + rand.Intn(10)
		}
	}

	// Core pulse
	r.coreTimer--
	if r.coreTimer <= 0 {
		r.coreBright = !r.coreBright
		r.coreTimer = 30
	}

	// Glitch frame
	if r.glitchActive {
		r.glitchTimer--
		if r.glitchTimer <= 0 {
			r.glitchActive = false
			r.glitchTimer = 300 + rand.Intn(300)
		}
	} else {
		r.glitchTimer--
		if r.glitchTimer <= 0 {
			r.glitchActive = true
			r.glitchTimer = 3 + rand.Intn(5)
		}
	}

	// Advance light phases
	for i := range r.lights {
		r.lights[i].phase++
	}

	// Drift sparks
	for i := range r.sparks {
		s := &r.sparks[i]
		s.y += s.speed
		if s.y >= r.height {
			s.y = 0
			s.x = rand.Intn(r.width)
			s.ch = sparkChar()
		}
	}

	// Spawn new sparks occasionally
	if len(r.sparks) < 70 && rand.Float32() < 0.1 {
		r.sparks = append(r.sparks, Spark{
			x:      rand.Intn(r.width),
			y:      0,
			speed:  1 + rand.Intn(2),
			bright: rand.Float32() < 0.2,
			ch:     sparkChar(),
		})
	}

}

func (r *RobotRoom) handleInput(ev InputEvent) {
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

	// Regular keystroke
	if r.playerEnergy[ev.PlayerID] < 100 {
		r.playerEnergy[ev.PlayerID] += 2
		if r.playerEnergy[ev.PlayerID] > 100 {
			r.playerEnergy[ev.PlayerID] = 100
		}
	}
	r.silenceTick = 0
	r.burstCount++
	if r.burstCount >= 5 && r.storyCooldown == 0 {
		r.storyCooldown = 0
		r.triggerQuiz()
		r.burstCount = 0
	}

	// Keystrokes spawn bright sparks
	for i := 0; i < 4 && len(r.sparks) < 100; i++ {
		r.sparks = append(r.sparks, Spark{
			x:      rand.Intn(r.width),
			y:      rand.Intn(r.height),
			speed:  1 + rand.Intn(2),
			bright: true,
			ch:     sparkChar(),
		})
	}
}

func (r *RobotRoom) renderFrame() string {
	r.canvas.Clear()

	// Layer 1: Atmospheric scanlines
	r.renderScanlines()

	// Layer 2: The robot silhouette
	r.renderRobot()

	// Layer 3: Drifting interference / rain
	r.renderSparks()

	// Layer 4: Glitch corruption
	if r.glitchActive {
		r.renderGlitch()
	}

	// Layer 5: Quiz card display
	if r.storyTimer > 0 && r.currentQuiz != nil {
		DrawQuizModal(r.canvas, r.currentQuiz, r.quizRevealed, r.quizChosen)
	}

	// Layer 6: Title
	title := "  ~ TITAN.forgotten ~  "
	start := (r.width - len(title)) / 2
	for i, ch := range title {
		r.canvas.Set(1, start+i, ch, ansi.TitanLight, "")
	}

	// Layer 7: Player presence list (top row)
	r.mu.Lock()
	names := make([]string, 0, len(r.players))
	for id := range r.players {
		names = append(names, id)
	}
	r.mu.Unlock()
	ansi.RenderPlayerList(r.canvas, 0, names, ansi.TitanLight)

	r.canvas.Set(0, r.width-2, ansi.SeasonalGlyph(), ansi.SeasonalAccent(), "")

	// Layer 8: Keybinds at bottom right
	ansi.RenderKeybinds(r.canvas, r.height-2, "[r]efresh [1-5]room [q]uit [a/b]answer", ansi.TitanDim)

	frame := ansi.RenderCanvas(r.canvas)
	// Rare mechanical groan
	if rand.Float32() < 0.005 {
		frame += "\x07"
	}
	return frame
}

// renderScanlines draws drifting horizontal shimmer lines.
func (r *RobotRoom) renderScanlines() {
	for row := 0; row < r.height; row++ {
		if (row+r.tick/4)%6 == 0 {
			for col := 0; col < r.width; col++ {
				if rand.Float32() < 0.15 {
					r.canvas.Set(row, col, '·', ansi.TitanScan, "")
				}
			}
		}
	}
}

// renderRobot draws the animated robot with breathing, blinking, and pulsing.
func (r *RobotRoom) renderRobot() {
	robotH := len(robotTemplate)
	robotW := len([]rune(robotTemplate[0]))
	startRow := (r.height - robotH) / 2
	startCol := (r.width-robotW)/2 + r.breathOffset

	for rowIdx, line := range robotTemplate {
		runes := []rune(line)
		for colIdx, ch := range runes {
			canvasRow := startRow + rowIdx
			canvasCol := startCol + colIdx
			if canvasCol < 0 || canvasCol >= r.width || canvasRow < 0 || canvasRow >= r.height {
				continue
			}
			if ch == ' ' {
				continue
			}

			var fg ansi.Color
			switch ch {
			case '◯':
				if r.eyeOpen {
					fg = ansi.TitanEyeOpen
					ch = '●'
				} else {
					fg = ansi.TitanEyeDim
					ch = '·'
				}
			case '▽':
				if r.coreBright {
					fg = ansi.TitanCore
				} else {
					fg = ansi.TitanCoreDim
				}
			case '▓':
				fg = ansi.TitanSteel
			default:
				fg = ansi.TitanDim
			}
			r.canvas.Set(canvasRow, canvasCol, ch, fg, "")
		}
	}

	// Blinking lights
	for _, light := range r.lights {
		canvasRow := startRow + light.row
		canvasCol := startCol + light.col
		if canvasCol < 0 || canvasCol >= r.width || canvasRow < 0 || canvasRow >= r.height {
			continue
		}
		if light.phase%90 < 45 {
			r.canvas.Set(canvasRow, canvasCol, '*', ansi.TitanLight, "")
		}
	}
}

// renderSparks draws drifting electrical interference in front of the robot.
func (r *RobotRoom) renderSparks() {
	for _, s := range r.sparks {
		if s.x < 0 || s.x >= r.width || s.y < 0 || s.y >= r.height {
			continue
		}
		var fg ansi.Color
		if s.bright {
			fg = ansi.TitanRainLit
		} else {
			fg = ansi.TitanRain
		}
		r.canvas.Set(s.y, s.x, s.ch, fg, "")
	}
}

// renderGlitch corrupts random cells with block characters.
func (r *RobotRoom) renderGlitch() {
	glitchChars := []rune{'▓', '░', '▒', '█', '·', ':'}
	glitchColors := []ansi.Color{ansi.TitanSteel, ansi.TitanDim, ansi.TitanBright, ansi.TitanScan}
	for i := 0; i < 25; i++ {
		row := rand.Intn(r.height)
		col := rand.Intn(r.width)
		ch := glitchChars[rand.Intn(len(glitchChars))]
		fg := glitchColors[rand.Intn(len(glitchColors))]
		r.canvas.Set(row, col, ch, fg, "")
	}
}



