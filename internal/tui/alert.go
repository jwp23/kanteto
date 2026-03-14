package tui

import (
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwp23/kanteto/internal/task"
)

// AlertPlayer plays alert sounds when tasks become due.
type AlertPlayer interface {
	Play()
}

type soundPlayer struct {
	soundFile string
}

func (p soundPlayer) Play() {
	if p.soundFile != "" {
		for _, name := range []string{"paplay", "afplay", "aplay"} {
			if path, err := exec.LookPath(name); err == nil {
				exec.Command(path, p.soundFile).Start()
				return
			}
		}
	}
	os.Stderr.Write([]byte("\a"))
}

// NewSoundPlayer returns an AlertPlayer that tries system sound players
// with the given file, falling back to terminal bell.
func NewSoundPlayer(soundFile string) AlertPlayer {
	return soundPlayer{soundFile: soundFile}
}

// alertPlayedMsg signals that an alert sound has been played.
type alertPlayedMsg struct{}

// playAlert returns a tea.Cmd that plays the alert and sends alertPlayedMsg.
func playAlert(player AlertPlayer) tea.Cmd {
	return func() tea.Msg {
		player.Play()
		return alertPlayedMsg{}
	}
}

// newlyDueTasks returns IDs of tasks whose deadline has passed but haven't
// been alerted yet. A task is "newly due" if DueAt != nil, !DueAt.After(now),
// and it's not already in alertedIDs.
func newlyDueTasks(tasks []task.Task, now time.Time, alertedIDs map[string]bool) []string {
	var ids []string
	for _, t := range tasks {
		if t.DueAt == nil {
			continue
		}
		if t.DueAt.After(now) {
			continue
		}
		if alertedIDs[t.ID] {
			continue
		}
		ids = append(ids, t.ID)
	}
	return ids
}
