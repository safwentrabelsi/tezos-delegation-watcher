package store

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
)

type PostgresStore struct {
	db *sql.DB
}

type Storer interface {
	SaveDelegation(d types.Delegation) error
	GetDelegations(year string) ([]types.Delegation, error)
}

// NewPostgresStore creates a new instance of PostgresStore
func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=delegates password=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
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
        block TEXT NOT NULL
    );`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create delegations table: %v", err)
	}
	return nil
}

func (s *PostgresStore) SaveDelegation(d types.Delegation) error {
	query := `INSERT INTO delegations (timestamp, amount, delegator, block) VALUES ($1, $2, $3, $4)`
	_, err := s.db.Exec(query, d.Timestamp, d.Amount, d.Delegator, d.Block)
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
