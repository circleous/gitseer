package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type databaseConnection struct {
	conn *sql.DB
}

type Service interface {
	Initialize() error
}

func NewDatabase(dbURI string) (Service, error) {
	conn, err := sql.Open("sqlite3", dbURI)

	return &databaseConnection{
		conn: conn,
	}, err
}

func (dbc *databaseConnection) Initialize() error {
	var err error
	_, err = dbc.conn.Exec(`
		CREATE TABLE IF NOT EXISTS analysis(

		);
	`)
	if err != nil {
		return err
	}

	_, err = dbc.conn.Exec(`
		CREATE TABLE IF NOT EXISTS stats(

		);
	`)
	if err != nil {
		return err
	}

	_, err = dbc.conn.Exec(`
		CREATE TABLE IF NOT EXISTS findings(

		);
	`)
	if err != nil {
		return err
	}

	return nil
}
