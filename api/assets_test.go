package api

import (
	"testing"

	"github.com/sdslabs/beastv4/core"
)

func TestDeclaresAssetRequiresExactMetadataEntry(t *testing.T) {
	assets := "guide.pdf" + core.DELIMITER + "images/logo.png"
	if !declaresAsset(assets, "images/logo.png") {
		t.Fatal("declared nested asset was rejected")
	}
	for _, requested := range []string{"flag.txt", "../guide.pdf", "logo.png"} {
		if declaresAsset(assets, requested) {
			t.Fatalf("undeclared asset %q was accepted", requested)
		}
	}
}
