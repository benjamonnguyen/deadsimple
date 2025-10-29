// Package sqlite
package sqlite

import (
	"database/sql"
	"embed"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	migratesql "github.com/golang-migrate/migrate/v4/database/sqlite"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

type database struct {
	// TODO bring your own logger
	conn *sql.DB
}

func (db *database) Conn() *sql.DB {
	return db.conn
}

func Open(url string) (*database, error) {
	if err := os.MkdirAll(filepath.Dir(url), 0744); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(url, os.O_CREATE, 0744)
	if err != nil {
		return nil, err
	}
	file.Close()

	conn, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, err
	}
	return &database{
		conn: conn,
	}, nil
}

// RunMigrations expects an embedded folder of sql files
func (db *database) RunMigrations(migrations embed.FS) error {
	d, err := migratesql.WithInstance(db.conn, &migratesql.Config{})
	if err != nil {
		return err
	}
	fs, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance(
		"iofs", fs,
		"sqlite", d)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (db *database) Close() error {
	return db.conn.Close()
}
