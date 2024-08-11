package db

import (
	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite"
)

var schema = `
CREATE TABLE IF NOT EXISTS alerts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	band TEXT NOT NULL,
	time DATETIME NOT NULL,
	min INTEGER NOT NULL,
	telegramid INTEGER NOT NULL
);
`

type Database struct {
	sql  *sqlx.DB
	file string
}

func New(file string) *Database {
	return &Database{
		file: file,
	}
}

func (d *Database) Connect() error {
	db, err := sqlx.Open("sqlite", d.file)
	if err != nil {
		return err
	}
	d.sql = db
	if err := d.sql.Ping(); err != nil {
		return err
	}
	return d.createTable()
}

func (d *Database) createTable() error {
	if _, err := d.sql.Exec(schema); err != nil {
		return err
	}
	return nil
}
