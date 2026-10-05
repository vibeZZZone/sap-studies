// Package tests holds integration tests that need a real Postgres.
//
// By default they start one with testcontainers, so `go test ./...` needs nothing but
// a Docker daemon. Set TEST_DATABASE_URL to run them against an already running
// Postgres instead, which is useful where Docker is unavailable.
package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sap-studies/shop-crm/backend/internal/config"
	"github.com/sap-studies/shop-crm/backend/migrations"
)

// testDB is the single database shared by every test in this package.
type testDB struct {
	Pool *pgxpool.Pool
	URL  string
}

var db *testDB

// TestMain prepares one database for the whole package and applies the schema
// including the demo seed.
func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "test bootstrap: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		started, err := startContainer(ctx)
		if err != nil {
			return 0, err
		}
		defer started.terminate(context.Background())
		url = started.url
	}

	if err := migrations.Up(url); err != nil {
		return 0, err
	}

	pool, err := newPool(ctx, url)
	if err != nil {
		return 0, err
	}
	defer pool.Close()

	db = &testDB{Pool: pool, URL: url}
	return m.Run(), nil
}

type runningDB struct {
	container testcontainers.Container
	url       string
}

func (r *runningDB) terminate(ctx context.Context) {
	_ = r.container.Terminate(ctx)
}

func startContainer(ctx context.Context) (*runningDB, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "shop_crm",
				"POSTGRES_USER":     "shop",
				"POSTGRES_PASSWORD": "shop",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(context.Background())
		return nil, fmt.Errorf("container host: %w", err)
	}
	mapped, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(context.Background())
		return nil, fmt.Errorf("container port: %w", err)
	}
	url := fmt.Sprintf("postgres://shop:shop@%s:%s/shop_crm?sslmode=disable", host, mapped.Port())
	return &runningDB{container: container, url: url}, nil
}

func newPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	// Enough connections for the race tests that run two transactions at once.
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return pool, nil
}

func testConfig() config.Config {
	return config.Config{
		BonusCashbackPercent: 5,
		BonusSpendLimitPct:   50,
	}
}