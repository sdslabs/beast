package remoteManager

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
)

func Init() error {
	ServerQueue = NewLoadBalancerQueue()
	var failures []error
	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active {
			// Skip SSH bootstrap for loopback workers; they use the local Docker socket from Beast.
			if config.Cfg.UseLocalDockerDaemon(serverDeployed) {
				continue
			}
			_, err := RunArgsOnServer(server, "mkdir", "-p", "--", filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR))
			if err != nil {
				failures = append(failures, fmt.Errorf("prepare remote %s: %w", serverDeployed, err))
				continue
			}
			ServerQueue.Push(server)
		}
	}
	return errors.Join(failures...)
}

func Stop() {
	for {
		_, err := ServerQueue.Pop()
		if err != nil {
			break
		}
	}
}
