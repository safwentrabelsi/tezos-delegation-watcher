package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
)

type PostgresStore struct {
	db *sql.DB
}

type Storer interface {
	SaveDelegations(ctx context.Context, delegations []types.FetchedDelegation) error
	GetDelegations(year string) ([]types.Delegation, error)
	GetCurrentLevel(ctx context.Context) (uint64, error)
	DeleteDelegationsFromLevel(ctx context.Context, level uint64) error
}

// NewPostgresStore creates a new instance of PostgresStore
func NewPostgresStore(cfg *config.DBConfig) (*PostgresStore, error) {
	db, err := sql.Open("postgres", cfg.GetPostgresqlDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	store := &PostgresStore{
		db: db,
	}

	if err := store.init(); err != nil {
		return nil, err
	}

	return store, nil
}

// init is called to initialize necessary tables in the database
func (s *PostgresStore) init() error {
	return s.createDelegationTable()
}

func (s *PostgresStore) createDelegationTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS delegations (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			amount TEXT NOT NULL,
			delegator TEXT NOT NULL,
			block INT NOT NULL
		);
	`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create delegations table: %v", err)
	}

	return nil
}

func (s *PostgresStore) SaveDelegations(ctx context.Context, delegations []types.FetchedDelegation) error {
	if len(delegations) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO delegations (timestamp, amount, delegator, block)
        VALUES ($1, $2, $3, $4)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range delegations {
		_, err = stmt.ExecContext(ctx, d.Timestamp, d.Amount, d.Sender.Address, d.Level)
		if err != nil {
			return fmt.Errorf("failed to save delegation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
func (s *PostgresStore) GetDelegations(year string) ([]types.Delegation, error) {
	var rows *sql.Rows
	var err error

	if year != "" {
		query := `
			SELECT timestamp, amount, delegator, block
			FROM delegations
			WHERE EXTRACT(YEAR FROM timestamp::date) = $1
			ORDER BY timestamp DESC
		`
		rows, err = s.db.Query(query, year)
	} else {
		query := `
			SELECT timestamp, amount, delegator, block
			FROM delegations
			ORDER BY timestamp DESC
		`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var delegations []types.Delegation
	for rows.Next() {
		var d types.Delegation
		if err := rows.Scan(&d.Timestamp, &d.Amount, &d.Delegator, &d.Block); err != nil {
			return nil, err
		}
		delegations = append(delegations, d)
	}

	return delegations, nil
}

func (s *PostgresStore) GetCurrentLevel(ctx context.Context) (uint64, error) {
	var level uint64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(block),0) FROM delegations`).Scan(&level)
	if err != nil {
		return 0, fmt.Errorf("failed to query database: %w", err)
	}
	return level, nil
}

func (s *PostgresStore) DeleteDelegationsFromLevel(ctx context.Context, level uint64) error {

	stmt, err := s.db.PrepareContext(ctx, "DELETE FROM delegations WHERE level >= $1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, level)
	if err != nil {
		return err
	}

	return nil
}
