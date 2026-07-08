package manager

import (
	"strings"
	"testing"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
)

func TestSadServersCheckerImageUsesImmutableImageID(t *testing.T) {
	imageID := strings.Repeat("a", 64)
	got, err := sadServersCheckerImage(imageID)
	if err != nil {
		t.Fatalf("normalize checker image: %v", err)
	}
	if got != "sha256:"+imageID {
		t.Fatalf("expected sha256 image ID, got %q", got)
	}
}

func TestSadServersCheckerImageRejectsTag(t *testing.T) {
	if _, err := sadServersCheckerImage("beast-checker-test:latest"); err == nil {
		t.Fatalf("expected mutable tag to be rejected")
	}
}

func TestValidateSadServersInstanceCheckerFailsClosed(t *testing.T) {
	imageID := strings.Repeat("b", 64)
	challenge := database.Challenge{
		Name:           "sad",
		SadServers:     true,
		CheckerImageId: imageID,
	}

	tests := []struct {
		name     string
		instance cache.Instance
	}{
		{
			name: "missing mode",
			instance: cache.Instance{
				InstanceID:     "inst-missing-mode",
				CheckerImageID: imageID,
			},
		},
		{
			name: "missing image",
			instance: cache.Instance{
				InstanceID:  "inst-missing-image",
				CheckerMode: core.CHECKER_MODE_SAD_SERVERS,
			},
		},
		{
			name: "mismatched image",
			instance: cache.Instance{
				InstanceID:     "inst-mismatch",
				CheckerMode:    core.CHECKER_MODE_SAD_SERVERS,
				CheckerImageID: strings.Repeat("c", 64),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateSadServersInstanceChecker(&tt.instance, challenge); err == nil {
				t.Fatalf("expected validation failure")
			}
		})
	}
}

func TestValidateSadServersInstanceCheckerAcceptsMatchingImage(t *testing.T) {
	imageID := strings.Repeat("d", 64)
	instance := cache.Instance{
		InstanceID:     "inst-ok",
		CheckerMode:    core.CHECKER_MODE_SAD_SERVERS,
		CheckerImageID: "sha256:" + imageID,
	}
	challenge := database.Challenge{
		Name:           "sad",
		SadServers:     true,
		CheckerImageId: imageID,
	}

	if err := ValidateSadServersInstanceChecker(&instance, challenge); err != nil {
		t.Fatalf("expected matching checker image to pass: %v", err)
	}
}
