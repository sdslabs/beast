package main

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/sdslabs/beastv4/core"
)

func TestEnsureLocalTLSCertificate(t *testing.T) {
	previous := core.BEAST_GLOBAL_DIR
	core.BEAST_GLOBAL_DIR = t.TempDir()
	defer func() { core.BEAST_GLOBAL_DIR = previous }()
	if err := os.MkdirAll(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR), 0700); err != nil {
		t.Fatal(err)
	}
	if err := ensureLocalTLSCertificate(); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.crt")
	keyPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.key")
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		t.Fatalf("load generated key pair: %v", err)
	}
	info, err := os.Stat(keyPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("unexpected key permissions: %v, %v", info, err)
	}
}
