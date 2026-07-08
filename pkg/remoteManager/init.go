package remoteManager

import (
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
)

func Init() {
	ServerQueue = NewLoadBalancerQueue()
	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active {
			// Skip SSH bootstrap for loopback workers; they use the local Docker socket from Beast.
			if config.Cfg.UseLocalDockerDaemon(serverDeployed) {
				continue
			}
			client, err := CreateSSHClient(server)
			if err != nil {
				log.Errorf("SSH connection to %s failed: %s\n", server.Host, err)
				continue
			}
			defer client.Close()
			ServerQueue.Push(server)
			_, err = RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s", shellQuote(filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR))))
			if err != nil {
				log.Errorf("failed to run command on server %s: %s", server.Host, err.Error())
			}
		}
	}
}

func Stop() {
	for {
		_, err := ServerQueue.Pop()
		if err != nil {
			break
		}
	}
}
