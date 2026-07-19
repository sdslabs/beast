package cr

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestDefaultContainerSecurity(t *testing.T) {
	hostConfig := &container.HostConfig{}
	applyDefaultContainerSecurity(hostConfig)
	if len(hostConfig.CapDrop) != 1 || hostConfig.CapDrop[0] != "ALL" {
		t.Fatalf("unexpected dropped capabilities: %v", hostConfig.CapDrop)
	}
	if len(hostConfig.SecurityOpt) != 1 || hostConfig.SecurityOpt[0] != "no-new-privileges" {
		t.Fatalf("unexpected security options: %v", hostConfig.SecurityOpt)
	}
	mounts := readOnlyBindMounts(map[string]string{"/host": "/container"})
	if len(mounts) != 1 || !mounts[0].ReadOnly {
		t.Fatalf("bind mount was not read-only: %+v", mounts)
	}
}
