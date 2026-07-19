package manager

import (
	"context"
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestBackgroundProbersRespectCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	BeastHealthCheckProber(ctx, 1)
	InstanceCleanupProber(ctx)
}

func TestActiveLocalServerNameSupportsConfiguredAlias(t *testing.T) {
	previous := config.Cfg
	config.Cfg = &config.BeastConfig{AvailableServers: map[string]config.AvailableServer{
		"worker-local": {Host: "127.0.0.1", Active: true},
	}}
	defer func() { config.Cfg = previous }()

	name, ok := activeLocalServerName()
	if !ok || name != "worker-local" {
		t.Fatalf("activeLocalServerName() = %q, %t", name, ok)
	}
}
