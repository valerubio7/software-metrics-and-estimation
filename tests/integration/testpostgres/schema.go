// Package testpostgres provides loopback-only PostgreSQL test database isolation.
package testpostgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var schemaSequence atomic.Uint64

// OpenIsolated validates a local PostgreSQL URL and returns a pool scoped to a
// unique schema. It owns and cleans up both the scoped pool and admin pool.
func OpenIsolated(t *testing.T, dsn, envName, prefix string) *pgxpool.Pool {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || !isLoopback(parsed.Hostname()) {
		t.Fatalf("%s must be a PostgreSQL URL on a loopback host", envName)
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect local PostgreSQL: %v", err)
	}
	schema := fmt.Sprintf("%s_%x_%x", prefix, time.Now().UnixNano(), schemaSequence.Add(1))
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("create isolated schema: %v", err)
	}
	config := admin.Config().Copy()
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
		t.Fatalf("connect isolated schema: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema %s: %v", schema, err)
		}
		admin.Close()
	})
	return pool
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
