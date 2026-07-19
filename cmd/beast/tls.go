package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sdslabs/beastv4/core"
)

func ensureLocalTLSCertificate() error {
	certPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.crt")
	keyPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.key")
	certInfo, certErr := os.Lstat(certPath)
	keyInfo, keyErr := os.Lstat(keyPath)
	if certErr == nil && keyErr == nil {
		if !certInfo.Mode().IsRegular() || !keyInfo.Mode().IsRegular() || keyInfo.Mode().Perm() != 0600 {
			return fmt.Errorf("existing TLS certificate and key must be regular files and the key must be 0600")
		}
		return nil
	}
	if !os.IsNotExist(certErr) || !os.IsNotExist(keyErr) || os.IsNotExist(certErr) != os.IsNotExist(keyErr) {
		return fmt.Errorf("TLS certificate and key must either both exist or both be absent")
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate TLS private key: %w", err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("generate certificate serial: %w", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	certificate, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("create TLS certificate: %w", err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal TLS private key: %w", err)
	}
	if err := writeExclusivePEM(keyPath, 0600, "PRIVATE KEY", keyBytes); err != nil {
		return err
	}
	if err := writeExclusivePEM(certPath, 0644, "CERTIFICATE", certificate); err != nil {
		_ = os.Remove(keyPath)
		return err
	}
	return nil
}

func writeExclusivePEM(path string, mode os.FileMode, blockType string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := pem.Encode(file, &pem.Block{Type: blockType, Bytes: contents}); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}
