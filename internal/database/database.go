package database

import (
	"context"
	"time"

	"github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/signature"
)

func (db *databaseConnection) GetRepoLatestCommit(ctx context.Context,
	repoName string) (string, error) {
	var hash string
	err := db.conn.QueryRowContext(ctx,
		`SELECT last_commit FROM analysis WHERE repo_name = ? LIMIT 1`,
		repoName).Scan(&hash)
	return hash, err
}

func (db *databaseConnection) UpsertRepo(ctx context.Context,
	repo git.Repository) error {
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO analysis (
			repo_name, last_commit
		) 
		VALUES (?, ?)
		ON CONFLICT (repo_name) DO UPDATE SET
			last_commit = excluded.last_commit`,
		repo.Name, repo.LatestCommit)
	return err
}

func (db *databaseConnection) AddFinding(ctx context.Context,
	repoName, filename string, commitHash string, matches []signature.Match) error {
	var err error

	if len(matches) == 0 {
		// return errors.New("nothing to add")
		return nil
	}

	currentDate := time.Now()

	for _, match := range matches {
		_, err = db.conn.ExecContext(ctx, `
			INSERT INTO findings (
				repo_name, filename, signature_id, commit_hash,
				description, match_string, line_num, created_at
			) VALUES (?,?,?,?,?,?)`,
			repoName, filename, match.SignatureID, commitHash,
			match.Description, match.Substring, match.LineNumber, currentDate,
		)
		if err != nil {
			break
		}
	}

	return err
}

// func (db *databaseConnection) GetLatestFinding(ctx context.Context,
// 	relTime *time.Time)
