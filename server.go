package main

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	"ssh-art/internal/ansi"
	"ssh-art/internal/rooms"
)

func startServer(roomMap map[string]rooms.Room, startRoom string, addr, keyPath string) error {
	s, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(keyPath),
		wish.WithMiddleware(func(next ssh.Handler) ssh.Handler {
			return func(sess ssh.Session) {
				// Enter alternate screen, hide cursor, clear
				io.WriteString(sess, "\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
				defer io.WriteString(sess, "\x1b[?25h\x1b[?1049l")

					// Detect terminal size from PTY
					termW, termH := 80, 24
					if pty, _, ok := sess.Pty(); ok {
						termW = pty.Window.Width
						termH = pty.Window.Height
						if termW < 40 {
							termW = 40
						}
						if termW > 300 {
							termW = 300
						}
						if termH < 12 {
							termH = 12
						}
						if termH > 100 {
							termH = 100
						}
					}

				roomCh := make(chan chan string, 1)

				currentRoom := roomMap[startRoom]
					currentRoom.Resize(termW, termH)
				user := ansi.SanitizeUsername(sess.User())
					frames := currentRoom.Subscribe(user)
				currentRoom.Input(rooms.InputEvent{PlayerID: user, IsJoin: true})
				roomCh <- frames

				// Output goroutine: reads from current room frame channel
				go func() {
					var currentFrames chan string
					for {
						select {
						case newFrames := <-roomCh:
							currentFrames = newFrames
						case frame, ok := <-currentFrames:
							if !ok {
								currentFrames = nil
								continue
							}
							io.WriteString(sess, frame)
						case <-sess.Context().Done():
							return
						}
					}
				}()

				// Input goroutine: terminal -> room, with room switching
				go func() {
					buf := make([]byte, 1024)
					for {
						n, err := sess.Read(buf)
						if err != nil {
							return
						}
						if n == 0 {
							continue
						}
						data := buf[:n]

						// Room switching commands
						switch {
						case len(data) == 1 && data[0] == '1':
							if target, ok := roomMap["rain"]; ok && target != currentRoom {
								currentRoom.Unsubscribe(user, frames)
								for i := 0; i < 2; i++ {
									io.WriteString(sess, renderNoiseFrame(termW, termH))
									time.Sleep(83 * time.Millisecond)
								}
								currentRoom = target
							target.Resize(termW, termH)
								frames = currentRoom.Subscribe(user)
								currentRoom.Input(rooms.InputEvent{PlayerID: user, IsSwitch: true})
								roomCh <- frames
							}
						case len(data) == 1 && data[0] == '2':
							if target, ok := roomMap["star"]; ok && target != currentRoom {
								currentRoom.Unsubscribe(user, frames)
								for i := 0; i < 2; i++ {
									io.WriteString(sess, renderNoiseFrame(termW, termH))
									time.Sleep(83 * time.Millisecond)
								}
								currentRoom = target
							target.Resize(termW, termH)
								frames = currentRoom.Subscribe(user)
								currentRoom.Input(rooms.InputEvent{PlayerID: user, IsSwitch: true})
								roomCh <- frames
							}
						case len(data) == 1 && data[0] == '3':
							if target, ok := roomMap["blackhole"]; ok && target != currentRoom {
								currentRoom.Unsubscribe(user, frames)
								for i := 0; i < 2; i++ {
									io.WriteString(sess, renderNoiseFrame(termW, termH))
									time.Sleep(83 * time.Millisecond)
								}
								currentRoom = target
							target.Resize(termW, termH)
								frames = currentRoom.Subscribe(user)
								currentRoom.Input(rooms.InputEvent{PlayerID: user, IsSwitch: true})
								roomCh <- frames
							}
						case len(data) == 1 && data[0] == '4':
							if target, ok := roomMap["redteam"]; ok && target != currentRoom {
								currentRoom.Unsubscribe(user, frames)
								for i := 0; i < 2; i++ {
									io.WriteString(sess, renderNoiseFrame(termW, termH))
									time.Sleep(83 * time.Millisecond)
								}
								currentRoom = target
							target.Resize(termW, termH)
								frames = currentRoom.Subscribe(user)
								currentRoom.Input(rooms.InputEvent{PlayerID: user, IsSwitch: true})
								roomCh <- frames
							}
						case len(data) == 1 && data[0] == '5':
							if target, ok := roomMap["robot"]; ok && target != currentRoom {
								currentRoom.Unsubscribe(user, frames)
								for i := 0; i < 2; i++ {
									io.WriteString(sess, renderNoiseFrame(termW, termH))
									time.Sleep(83 * time.Millisecond)
								}
								currentRoom = target
							target.Resize(termW, termH)
								frames = currentRoom.Subscribe(user)
								currentRoom.Input(rooms.InputEvent{PlayerID: user, IsSwitch: true})
								roomCh <- frames
							}
						case len(data) == 1 && data[0] == 'q':
							_ = sess.Exit(0)
							return
						default:
							currentRoom.Input(rooms.InputEvent{
								PlayerID: user,
								Data:     append([]byte(nil), data...),
							})
						}
					}
				}()

				// Block until SSH session ends
				<-sess.Context().Done()
				currentRoom.Unsubscribe(user, frames)
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("wish server: %w", err)
	}

	fmt.Printf("SSH art museum listening on %s\n", addr)
	return s.ListenAndServe()
}

// renderNoiseFrame generates a single frame of static for room transitions.
func renderNoiseFrame(width, height int) string {
	c := ansi.NewCanvas(width, height)
	chars := []rune{'·', '.', ':', '*', '+', '|', '-', ' '}
	colors := []ansi.Color{ansi.MatrixDeep, ansi.MatrixDark, ansi.MatrixGreen}
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			ch := chars[rand.Intn(len(chars))]
			fg := colors[rand.Intn(len(colors))]
			c.Set(row, col, ch, fg, "")
		}
	}
	return ansi.RenderCanvas(c)
}
