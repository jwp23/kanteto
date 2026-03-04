package daemon

import (
	"testing"
	"time"
)

func TestSoundCommand(t *testing.T) {
	cmd := soundCommand("")
	if cmd == "" {
		t.Skip("no sound player found on this system")
	}
	// Just verify it returns something plausible
	if cmd != "paplay" && cmd != "afplay" && cmd != "aplay" {
		t.Errorf("unexpected sound command: %q", cmd)
	}
}

func TestCheckInterval(t *testing.T) {
	if checkInterval <= 0 {
		t.Error("checkInterval must be positive")
	}
	if checkInterval > time.Minute {
		t.Error("checkInterval should be <= 1 minute for timely reminders")
	}
}
