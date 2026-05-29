package rooms

import (
	"testing"
	"time"
)

func TestRainRoomStoryEngine(t *testing.T) {
	r := NewRainRoom()

	// Simulate Subscribe manually (add player, no goroutine needed)
	r.players["alice"] = struct{}{}

	// Test 1: First entry triggers a story
	r.handleInput(InputEvent{PlayerID: "alice", IsJoin: true})
	if r.storyTimer == 0 {
		t.Error("expected story timer > 0 after first join")
	}
	if r.storyCooldown == 0 {
		t.Error("expected story cooldown > 0 after trigger")
	}
	// Test 2: Cooldown prevents immediate second story
	oldQuiz := r.currentQuiz
	r.players["bob"] = struct{}{}
	r.handleInput(InputEvent{PlayerID: "bob", IsJoin: true})
	if r.currentQuiz != oldQuiz {
		t.Error("cooldown should have blocked second story")
	}

	// Test 3: Leave sets wasEmpty when last player leaves
	r.wasEmpty = false
	delete(r.players, "alice")
	delete(r.players, "bob")
	r.handleInput(InputEvent{PlayerID: "alice", IsLeave: true})
	if !r.wasEmpty {
		t.Error("expected wasEmpty=true after last player leaves")
	}

	// Test 4: Empty room transition story on rejoin
	r.storyCooldown = 0
	r.storyTimer = 0
	r.players["alice"] = struct{}{}
	r.handleInput(InputEvent{PlayerID: "alice", IsJoin: true})
	// wasEmpty should be consumed and reset
	if r.wasEmpty {
		t.Error("expected wasEmpty=false after empty story consumed")
	}

	// Test 5: Silence story after 360 ticks
	r.storyCooldown = 0
	r.silenceTick = 359
	r.update()
	if r.storyTimer == 0 {
		t.Error("expected silence story after 360 ticks")
	}

	// Test 6: Typing burst story
	r.storyCooldown = 0
	r.burstCount = 0
	r.typingIntensity = 0
	for i := 0; i < 3; i++ {
		r.handleInput(InputEvent{PlayerID: "alice", Data: []byte("x")})
	}
	if r.storyTimer == 0 {
		t.Error("expected burst story after 3 intense keystrokes")
	}

	// Test 7: Switch story
	r.storyCooldown = 0
	r.storyTimer = 0
	r.handleInput(InputEvent{PlayerID: "alice", IsSwitch: true})
	if r.storyTimer == 0 {
		t.Error("expected switch story")
	}

	// Test 8: Burst count decays
	r.burstCount = 5
	for i := 0; i < 15; i++ {
		r.update()
	}
	if r.burstCount >= 5 {
		t.Error("expected burst count to decay")
	}

	// Test 9: Story timer and cooldown decrement
	r.storyTimer = 10
	r.storyCooldown = 10
	r.update()
	if r.storyTimer != 9 {
		t.Errorf("expected storyTimer=9, got %d", r.storyTimer)
	}
	if r.storyCooldown != 9 {
		t.Errorf("expected storyCooldown=9, got %d", r.storyCooldown)
	}

	// Test 10: Silence reset on keystroke
	r.silenceTick = 100
	r.handleInput(InputEvent{PlayerID: "alice", Data: []byte("x")})
	if r.silenceTick != 0 {
		t.Error("expected silenceTick reset to 0 on keystroke")
	}

	// Test 11: Night story between 0-5 AM
	hour := time.Now().Hour()
	if hour >= 0 && hour < 5 {
		r.storyCooldown = 0
		r.storyTimer = 0
		r.handleInput(InputEvent{PlayerID: "alice", IsJoin: true})
		if r.storyTimer == 0 {
			t.Error("expected night story during late hours")
		}
	}
	// If not late night, we just verify the code path doesn't crash
}

func TestStarRoomStoryEngine(t *testing.T) {
	r := NewStarRoom()

	// Test 1: First entry
	r.players["alice"] = struct{}{}
	r.handleInput(InputEvent{PlayerID: "alice", IsJoin: true})
	if r.storyTimer == 0 {
		t.Error("expected story after first join")
	}

	// Test 2: Silence story
	r.storyCooldown = 0
	r.silenceTick = 359
	r.update()
	if r.storyTimer == 0 {
		t.Error("expected silence story")
	}

	// Test 3: Keystroke burst (star room: 5 keystrokes)
	r.storyCooldown = 0
	r.storyTimer = 0
	r.burstCount = 0
	for i := 0; i < 5; i++ {
		r.handleInput(InputEvent{PlayerID: "alice", Data: []byte("x")})
	}
	if r.storyTimer == 0 {
		t.Error("expected burst story after 5 keystrokes")
	}

	// Test 4: Leave empties room
	r.wasEmpty = false
	delete(r.players, "alice")
	r.handleInput(InputEvent{PlayerID: "alice", IsLeave: true})
	if !r.wasEmpty {
		t.Error("expected wasEmpty after last player leaves")
	}

	// Test 5: Switch story
	r.storyCooldown = 0
	r.storyTimer = 0
	r.handleInput(InputEvent{PlayerID: "alice", IsSwitch: true})
	if r.storyTimer == 0 {
		t.Error("expected switch story")
	}
}
