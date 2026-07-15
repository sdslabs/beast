package manager

import (
	"context"
	"testing"
)

func TestBackgroundProbersRespectCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	BeastHealthCheckProber(ctx, 1)
	InstanceCleanupProber(ctx)
}
