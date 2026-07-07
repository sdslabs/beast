package cr

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sdslabs/beastv4/utils"
)

func TestCreateSearchAndRemoveContainerIntegration(t *testing.T) {
	if os.Getenv("BEAST_TEST_DOCKER") != "1" {
		t.Skip("set BEAST_TEST_DOCKER=1 to run Docker container integration tests")
	}

	image := os.Getenv("BEAST_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "redis:7-alpine"
	}

	challengeName := fmt.Sprintf("beast-cr-integration-%d", time.Now().UnixNano())
	containerName := utils.ProjectNameNotInstanced(challengeName)

	containerID, err := CreateContainerFromImage(&CreateContainerConfig{
		ImageId:       image,
		ContainerName: containerName,
		ChallengeName: challengeName,
		MountsMap:     map[string]string{},
		Labels: map[string]string{
			"beast.integration_test": "true",
		},
	})
	if err != nil {
		t.Fatalf("create container from image %s: %v", image, err)
	}
	defer func() {
		_ = StopAndRemoveContainer(containerID)
	}()

	containers, err := SearchRunningContainerByFilter(map[string]string{"id": containerID})
	if err != nil {
		t.Fatalf("search running container by id: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected one running container, got %d", len(containers))
	}

	container := containers[0]
	if container.Labels["beast.challenge"] != challengeName {
		t.Fatalf("expected beast.challenge label %q, got %q", challengeName, container.Labels["beast.challenge"])
	}
	if container.Labels["com.sdslabs.beast.project"] != utils.ProjectNameNotInstanced(challengeName) {
		t.Fatalf("unexpected Beast project label %q", container.Labels["com.sdslabs.beast.project"])
	}

	if err := StopAndRemoveContainer(containerID); err != nil {
		t.Fatalf("stop and remove container: %v", err)
	}
	containers, err = SearchContainerByFilter(map[string]string{"id": containerID})
	if err != nil {
		t.Fatalf("search removed container by id: %v", err)
	}
	if len(containers) != 0 {
		t.Fatalf("expected removed container to be absent, got %d matches", len(containers))
	}
}
