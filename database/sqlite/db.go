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

type Connection interface {
	DB() *sql.DB
	RunMigrations(embed.FS) error
	Close() error
}

type conn struct {
	// TODO bring your own logger
	db *sql.DB
}

func (conn *conn) DB() *sql.DB {
	return conn.db
}

func Open(url string) (Connection, error) {
	if err := os.MkdirAll(filepath.Dir(url), 0744); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(url, os.O_CREATE, 0744)
	if err != nil {
		return nil, err
	}
	file.Close()

	db, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, err
	}
	return &conn{
		db: db,
	}, nil
}

// RunMigrations expects an embedded folder of sql files
func (conn *conn) RunMigrations(migrations embed.FS) error {
	d, err := migratesql.WithInstance(conn.db, &migratesql.Config{})
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

func (conn *conn) Close() error {
	return conn.db.Close()
}
