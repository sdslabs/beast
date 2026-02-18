package remoteManager

import (
	"fmt"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
	"path/filepath"
)

func Init() {
	ServerQueue = NewLoadBalancerQueue()
	for _, server := range config.Cfg.AvailableServers {
		if server.Active {
			if server.Host == core.LOCALHOST {
				continue
				ServerQueue.Push(server)
			}
			client, err := CreateSSHClient(server)
			if err != nil {
				log.Errorf("SSH connection to %s failed: %s\n", server.Host, err)
				continue
			}
			defer client.Close()
			ServerQueue.Push(server)
			_, err = RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s", filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR)))
			if err != nil {
				log.Errorf("fialed to run command on server %s: %s", server.Host, err.Error())
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
