package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultConnectTimeout = 5 * time.Second
	connectTimeoutEnvVar  = "DB_CONNECT_TIMEOUT_SEC"
)

// Connect opens a PostgreSQL connection using GORM and applies pool settings.
func Connect(dsn string) (*gorm.DB, error) {
	timeout := connectTimeout()
	dsnWithTimeout, target := prepareDSN(dsn, timeout)

	logLevel := logger.Warn
	if strings.EqualFold(os.Getenv("ENV"), "production") {
		logLevel = logger.Error
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsnWithTimeout,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
		DisableAutomaticPing:                     true,
	})
	if err != nil {
		return nil, fmt.Errorf("db.Connect: open %s: %w", target, err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("db.Connect: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("db.Connect: timed out after %s while connecting to %s; check DATABASE_URL/DB_URL, network access, and database availability", timeout, target)
		}
		return nil, fmt.Errorf("db.Connect: ping %s: %w", target, err)
	}

	return database, nil
}

func connectTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(connectTimeoutEnvVar))
	if raw == "" {
		return defaultConnectTimeout
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultConnectTimeout
	}

	return time.Duration(seconds) * time.Second
}

func prepareDSN(dsn string, timeout time.Duration) (string, string) {
	if parsed, err := url.Parse(dsn); err == nil && parsed.Scheme != "" {
		return prepareURLDSN(parsed, timeout)
	}

	return prepareKeywordDSN(dsn, timeout), "configured database"
}

func prepareURLDSN(parsed *url.URL, timeout time.Duration) (string, string) {
	query := parsed.Query()
	if query.Get("connect_timeout") == "" {
		query.Set("connect_timeout", strconv.Itoa(timeoutSeconds(timeout)))
		parsed.RawQuery = query.Encode()
	}

	target := parsed.Host
	if dbName := strings.TrimPrefix(parsed.Path, "/"); dbName != "" {
		target += "/" + dbName
	}

	if target == "" {
		target = "configured database"
	}

	return parsed.String(), target
}

func prepareKeywordDSN(dsn string, timeout time.Duration) string {
	if strings.Contains(dsn, "connect_timeout=") {
		return dsn
	}

	return strings.TrimSpace(dsn) + fmt.Sprintf(" connect_timeout=%d", timeoutSeconds(timeout))
}

func timeoutSeconds(timeout time.Duration) int {
	seconds := int(timeout / time.Second)
	if seconds <= 0 {
		return 1
	}

	return seconds
}
