package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
)

type PostgresStore struct {
	db *sql.DB
}

type Storer interface {
	SaveDelegations(d []types.GetDelegationsResponse) error
	GetDelegations(year string) ([]types.Delegation, error)
	GetCurrentLevel(ctx context.Context) (uint64, error)
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

// createDelegationTable creates the delegations table if it does not exist
func (s *PostgresStore) createDelegationTable() error {
	query := `
    CREATE TABLE IF NOT EXISTS delegations (
        id SERIAL PRIMARY KEY,
        timestamp TIMESTAMP NOT NULL,
        amount TEXT NOT NULL,
        delegator TEXT NOT NULL,
        block INT NOT NULL
    );`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create delegations table: %v", err)
	}
	return nil
}

func (s *PostgresStore) SaveDelegations(delegations []types.GetDelegationsResponse) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(pq.CopyIn("delegations", "timestamp", "amount", "delegator", "block"))
	if err != nil {
		return err
	}

	for _, d := range delegations {
		_, err = stmt.Exec(d.Timestamp, d.Amount, d.Sender.Address, d.Level)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Flush the buffered data
	_, err = stmt.Exec()
	if err != nil {
		tx.Rollback()
		return err
	}

	// Close the statement
	err = stmt.Close()
	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	return err
}

func (s *PostgresStore) GetDelegations(year string) ([]types.Delegation, error) {
	var rows *sql.Rows
	var err error

	if year != "" {
		query := `SELECT timestamp, amount, delegator, block FROM delegations WHERE EXTRACT(YEAR FROM timestamp::date) = $1 ORDER BY timestamp DESC`
		rows, err = s.db.Query(query, year)
	} else {
		query := `SELECT timestamp, amount, delegator, block FROM delegations ORDER BY timestamp DESC`
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
