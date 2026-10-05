// Package migrations carries the versioned schema files and applies them on boot.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	// Registers the "postgres" database driver.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var files embed.FS

// Up applies every pending migration. An already migrated database is a no-op, which
// is what lets the container run it on every start.
func Up(databaseURL string) error {
	src, err := iofs.New(files, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// Files exposes the embedded schema, used by tooling and tests.
func Files() fs.FS { return files }