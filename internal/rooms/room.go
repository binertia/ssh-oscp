package rooms

// Room is the interface shared by all rooms in the museum.
type Room interface {
	Subscribe(playerID string) chan string
	Unsubscribe(playerID string, ch chan string)
	Input(ev InputEvent)
	Resize(width, height int)
}

// InputEvent is a keystroke or lifecycle event from a player.
// All events flow through the room's input channel and are processed
// by the single room goroutine.
type InputEvent struct {
	PlayerID string
	Data     []byte
	IsJoin   bool // fresh connection to this room
	IsLeave  bool // disconnect from this room
	IsSwitch bool // arrived from another room
}
