package database

import (
	"context"
	"database/sql"

	// for database/sql
	_ "github.com/mattn/go-sqlite3"

	"github.com/circleous/gitseer/pkg/signature"
)

type databaseConnection struct {
	conn *sql.DB
}

// Service is the main interface for database package
type Service interface {
	Initialize() error
	Close()

	AddFinding(ctx context.Context, repoName, filename string, commitHash string, matches []signature.Match) error
	GetRepoLatestCommit(ctx context.Context, repoName string) (string, error)
}

// NewDatabase create a new connection to database
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
			id INTEGER PRIMARY KEY,
			repo_name VARCHAR(275) UNIQUE NOT NULL,
			last_commit VARCHAR(40)
		);
	`)
	if err != nil {
		return err
	}

	// Should we use db for storing stats?
	// _, err = dbc.conn.Exec(`
	// 	CREATE TABLE IF NOT EXISTS stats(
	// 	);
	// `)
	// if err != nil {
	// 	return err
	// }

	_, err = dbc.conn.Exec(`
		CREATE TABLE IF NOT EXISTS findings(
			id INTEGER PRIMARY KEY,
			repo_name VARCHAR(255),
			signature_id VARCHAR(40),
			commit_hash VARCHAR(40),
			filename TEXT,
			description TEXT,
			match_string TEXT,
			line_num INTEGER,
			created_at TIMESTAMP,
			UNIQUE(signature_id,repo_name,commit_hash,filename)
		);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (dbc *databaseConnection) Close() {
	dbc.conn.Close()
}
