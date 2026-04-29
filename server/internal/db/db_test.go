package db

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConnectTimeoutDefaultsToFiveSeconds(t *testing.T) {
	t.Setenv(connectTimeoutEnvVar, "")

	if got := connectTimeout(); got != defaultConnectTimeout {
		t.Fatalf("connectTimeout() = %v, want %v", got, defaultConnectTimeout)
	}
}

func TestConnectTimeoutHonorsPositiveEnvOverride(t *testing.T) {
	t.Setenv(connectTimeoutEnvVar, "12")

	if got := connectTimeout(); got != 12*time.Second {
		t.Fatalf("connectTimeout() = %v, want %v", got, 12*time.Second)
	}
}

func TestConnectTimeoutFallsBackOnInvalidEnvValue(t *testing.T) {
	t.Setenv(connectTimeoutEnvVar, "not-a-number")

	if got := connectTimeout(); got != defaultConnectTimeout {
		t.Fatalf("connectTimeout() = %v, want %v", got, defaultConnectTimeout)
	}
}

func TestPrepareDSNAddsConnectTimeoutToURLDSN(t *testing.T) {
	dsn := "postgresql://user:pass@db.example.com:6543/postgres?sslmode=disable"

	prepared, target := prepareDSN(dsn, 7*time.Second)

	parsed, err := url.Parse(prepared)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", prepared, err)
	}
	if got := parsed.Query().Get("connect_timeout"); got != "7" {
		t.Fatalf("connect_timeout = %q, want %q", got, "7")
	}
	if target != "db.example.com:6543/postgres" {
		t.Fatalf("target = %q, want %q", target, "db.example.com:6543/postgres")
	}
}

func TestPrepareDSNPreservesExistingConnectTimeout(t *testing.T) {
	dsn := "postgresql://user:pass@db.example.com:6543/postgres?connect_timeout=15&sslmode=disable"

	prepared, _ := prepareDSN(dsn, 7*time.Second)

	parsed, err := url.Parse(prepared)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", prepared, err)
	}
	if got := parsed.Query().Get("connect_timeout"); got != "15" {
		t.Fatalf("connect_timeout = %q, want %q", got, "15")
	}
}

func TestPrepareDSNAddsConnectTimeoutToKeywordDSN(t *testing.T) {
	dsn := "host=db.example.com port=5432 user=postgres password=secret dbname=ticketing sslmode=disable"

	prepared, target := prepareDSN(dsn, 9*time.Second)

	if !strings.Contains(prepared, "connect_timeout=9") {
		t.Fatalf("prepared DSN = %q, want connect_timeout=9", prepared)
	}
	if target != "configured database" {
		t.Fatalf("target = %q, want %q", target, "configured database")
	}
}

func TestMain(m *testing.M) {
	_ = os.Unsetenv(connectTimeoutEnvVar)
	os.Exit(m.Run())
}
