package memory

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// EmotionalClimate is a simple 3-axis mood vector for a room.
type EmotionalClimate struct {
	Intensity  int `json:"intensity"`  // excitement, chaos, heat
	Loneliness int `json:"loneliness"` // emptiness, longing
	Chaos      int `json:"chaos"`      // unpredictability, disorder
}

// RoomMemory is the emotional and historical state a room carries across sessions.
// Stored as a simple JSON file — poetic, not administrative.
type RoomMemory struct {
	LastEmptyAt      time.Time        `json:"lastEmptyAt"`
	TotalVisits      int              `json:"totalVisits"`
	PeakPlayers      int              `json:"peakPlayers"`
	LastWords        []string         `json:"lastWords"` // fading wall text fragments
	EmotionalClimate EmotionalClimate `json:"emotionalClimate"`
	RecentStories    []string         `json:"recentStories"` // last 5 stories told
	LastActivityAt   time.Time        `json:"lastActivityAt"`
}

const MemoryDir = ".memory"

func EnsureMemoryDir() error {
	return os.MkdirAll(MemoryDir, 0755)
}

func MemoryPath(roomName string) string {
	return filepath.Join(MemoryDir, roomName+".json")
}

// LoadMemory reads a room's memory from disk. Returns empty memory if no file exists.
func LoadMemory(roomName string) RoomMemory {
	var m RoomMemory
	data, err := os.ReadFile(MemoryPath(roomName))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

// SaveMemory writes a room's memory to disk. Best-effort; errors are swallowed.
func SaveMemory(roomName string, m RoomMemory) {
	_ = EnsureMemoryDir()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(MemoryPath(roomName), data, 0644)
}

// RecordStory appends a story to recent memory, keeping only the last 5.
func (m *RoomMemory) RecordStory(text string) {
	m.RecentStories = append([]string{text}, m.RecentStories...)
	if len(m.RecentStories) > 5 {
		m.RecentStories = m.RecentStories[:5]
	}
}

// DecayClimate slowly normalizes emotional values toward zero.
// Intensity decays when empty; loneliness decays when occupied.
func (m *RoomMemory) DecayClimate(occupied bool) {
	if !occupied && m.EmotionalClimate.Intensity > 0 {
		m.EmotionalClimate.Intensity--
	}
	if occupied && m.EmotionalClimate.Loneliness > 0 {
		m.EmotionalClimate.Loneliness--
	}
	if m.EmotionalClimate.Chaos > 0 {
		m.EmotionalClimate.Chaos--
	}
}

// MemoryStory picks a random template and fills it with memory values.
func MemoryStory(m RoomMemory, templates []string) string {
	if len(templates) == 0 {
		return ""
	}
	tmpl := templates[rand.Intn(len(templates))]
	lastWord := ""
	if len(m.LastWords) > 0 {
		lastWord = m.LastWords[0]
	}
	return fmt.Sprintf(tmpl, m.TotalVisits, m.PeakPlayers, lastWord)
}
