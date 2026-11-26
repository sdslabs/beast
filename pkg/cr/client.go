package cr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/client"
)

type dockerConfigMeta struct {
	CurrentContext string `json:"currentContext"`
}

type dockerContextMeta struct {
	Endpoints struct {
		Docker struct {
			Host string `json:"Host"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

func resolveDockerContextHost() string {
	if os.Getenv("DOCKER_HOST") != "" {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return ""
	}

	dockerDir := os.Getenv("DOCKER_CONFIG")
	if dockerDir == "" {
		dockerDir = filepath.Join(homeDir, ".docker")
	}

	ctxName := os.Getenv("DOCKER_CONTEXT")
	if ctxName == "" {
		cfgData, err := os.ReadFile(filepath.Join(dockerDir, "config.json"))
		if err != nil {
			return ""
		}
		var cfg dockerConfigMeta
		if err := json.Unmarshal(cfgData, &cfg); err != nil {
			return ""
		}
		ctxName = cfg.CurrentContext
	}

	if ctxName == "" || ctxName == "default" {
		return ""
	}

	sum := sha256.Sum256([]byte(ctxName))
	metaPath := filepath.Join(dockerDir, "contexts", "meta", hex.EncodeToString(sum[:]), "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return ""
	}

	var meta dockerContextMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return ""
	}
	return meta.Endpoints.Docker.Host
}

func dockerCommandEnv(extra ...string) []string {
	env := os.Environ()
	if os.Getenv("DOCKER_HOST") == "" {
		if host := resolveDockerContextHost(); host != "" {
			env = append(env, "DOCKER_HOST="+host)
		}
	}
	env = append(env, extra...)
	return env
}

func newDockerCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("docker", args...)
	cmd.Env = dockerCommandEnv()
	return cmd
}

func newDockerClient() (*client.Client, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if host := resolveDockerContextHost(); host != "" {
		opts = append(opts, client.WithHost(host))
	}
	return client.NewClientWithOpts(opts...)
}

func PingDaemon() error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.Ping(context.Background())
	return err
}
