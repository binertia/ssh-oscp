package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"ssh-art/internal/rooms"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	keyPath := ".ssh/term_info_ed25519"
	if err := ensureHostKey(keyPath); err != nil {
		return fmt.Errorf("host key: %w", err)
	}

	rain := rooms.NewRainRoom()
	star := rooms.NewStarRoom()
	blackhole := rooms.NewBlackholeRoom()
	redteam := rooms.NewRedteamRoom()
	robot := rooms.NewRobotRoom()
	go rain.Run()
	go star.Run()
	go blackhole.Run()
	go redteam.Run()
	go robot.Run()

	roomMap := map[string]rooms.Room{
		"rain":      rain,
		"star":      star,
		"blackhole": blackhole,
		"redteam":   redteam,
		"robot":     robot,
	}

	return startServer(roomMap, "rain", ":2222", keyPath)
}

func ensureHostKey(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	log.Printf("Generating host key at %s...", path)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", path, "-N", "", "-C", "ssh-art")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen failed: %w\noutput: %s", err, out)
	}
	return nil
}
