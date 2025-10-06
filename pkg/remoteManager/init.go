package remoteManager

import (
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
)

func Init() {
	ServerQueue = NewLoadBalancerQueue()
	for _, server := range config.Cfg.AvailableServers {
		if server.Active {
			if server.Host == core.LOCALHOST {
				ServerQueue.Push(server)
				continue
			}
			client, err := CreateSSHClient(server)
			if err != nil {
				log.Errorf("SSH connection to %s failed: %s\n", server.Host, err)
				continue
			}
			defer client.Close()
			ServerQueue.Push(server)
			RunCommandOnServer(server, "mkdir -p $HOME/.beast/staging/")
		}
	}
}

func Stop() {
	for {
		_, err := ServerQueue.Pop()
		if err == nil {
			break
		}
	}
}
