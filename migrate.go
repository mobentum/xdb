package xdb

import (
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrateUp runs all pending up migrations from the given embedded filesystem.
//
// The caller must blank-import the appropriate golang-migrate database driver
// matching the DB's dialect:
//
//	import _ "github.com/golang-migrate/migrate/v4/database/postgres"
//	import _ "github.com/golang-migrate/migrate/v4/database/sqlite3"
//	import _ "github.com/golang-migrate/migrate/v4/database/mysql"
//
// Migration files follow golang-migrate's naming convention:
//
//	000001_create_users.up.sql
//	000001_create_users.down.sql
//
// Returns nil when already up-to-date (migrate.ErrNoChange is suppressed).
// Returns an error if DSN is not available (Wrap'd DB).
func (d *DB) MigrateUp(fsys fs.FS, path string) error {
	return d.migrate(fsys, path, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	})
}

// MigrateDown rolls back all down migrations.
func (d *DB) MigrateDown(fsys fs.FS, path string) error {
	return d.migrate(fsys, path, func(m *migrate.Migrate) error {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	})
}

// MigrateTo migrates to a specific version (version 0 = initial state).
func (d *DB) MigrateTo(fsys fs.FS, path string, version uint) error {
	return d.migrate(fsys, path, func(m *migrate.Migrate) error {
		if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	})
}

// MigrateStep applies n migrations (positive = up, negative = down).
func (d *DB) MigrateStep(fsys fs.FS, path string, n int) error {
	return d.migrate(fsys, path, func(m *migrate.Migrate) error {
		if err := m.Steps(n); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	})
}

func (d *DB) migrate(fsys fs.FS, path string, fn func(*migrate.Migrate) error) error {
	if d.dsn == "" {
		return fmt.Errorf("xdb.Migrate: DSN not available (DB was created via Wrap, not New)")
	}

	src, err := iofs.New(fsys, path)
	if err != nil {
		return fmt.Errorf("xdb.Migrate: source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, d.migrationURL())
	if err != nil {
		return fmt.Errorf("xdb.Migrate: %w", err)
	}
	defer m.Close()

	if err := fn(m); err != nil {
		return fmt.Errorf("xdb.Migrate: %w", err)
	}
	return nil
}

func (d *DB) migrationURL() string {
	switch d.dialect.DriverName() {
	case "sqlite3":
		return "sqlite3://" + d.dsn
	default:
		return d.dsn
	}
}
