package utils

import (
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestCheckTimeRejectsMalformedWindowWithoutPanicking(t *testing.T) {
	previous := config.Cfg
	config.Cfg = &config.BeastConfig{CompetitionInfo: config.CompetitionInfo{
		StartingTime: "malformed",
		EndingTime:   "also malformed",
		TimeZone:     "UTC",
	}}
	t.Cleanup(func() { config.Cfg = previous })

	if err, state := CheckTime(); err == nil || state != -1 {
		t.Fatalf("expected malformed time error and state -1, got %v and %d", err, state)
	}
}
