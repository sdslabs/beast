package database

import (
	"net/url"
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestPostgresDSNEncodesCredentials(t *testing.T) {
	dsn := postgresDSN(config.PsqlConfig{
		User:        "beast user",
		Password:    "secret with =' delimiters",
		Dbname:      "beast-db",
		Host:        "127.0.0.1",
		Port:        "5432",
		SslMode:     "require",
		SSLRootCert: "/etc/ssl/certs/root.pem",
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if parsed.User.Username() != "beast user" || password != "secret with =' delimiters" {
		t.Fatalf("credentials did not round trip through DSN: %s", dsn)
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("sslmode did not round trip through DSN: %s", dsn)
	}
	if parsed.Query().Get("sslrootcert") != "/etc/ssl/certs/root.pem" {
		t.Fatalf("sslrootcert did not round trip through DSN: %s", dsn)
	}
}
