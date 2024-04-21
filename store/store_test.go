package store

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/stretchr/testify/assert"
)

func TestSaveDelegations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()
	delegations := []types.FetchedDelegation{
		{Timestamp: time.Now().String(), Amount: 100, Sender: types.Sender{Address: "tz1"}, Level: 1},
	}

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO delegations (timestamp, amount, delegator, block) VALUES ($1, $2, $3, $4)"))
	for _, d := range delegations {
		prep.ExpectExec().WithArgs(d.Timestamp, d.Amount, d.Sender.Address, d.Level).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	err = store.SaveDelegations(ctx, delegations)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetCurrentLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(MAX(block),0) FROM delegations")).
		WillReturnRows(sqlmock.NewRows([]string{"COALESCE"}).AddRow(10))

	level, err := store.GetCurrentLevel(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), level)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetDelegations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT timestamp, amount, delegator, block FROM delegations WHERE EXTRACT(YEAR FROM timestamp) = $1 ORDER BY timestamp DESC")).
		WithArgs("2024").
		WillReturnRows(sqlmock.NewRows([]string{"timestamp", "amount", "delegator", "block"}).
			AddRow("2024-04-21T16:23:27Z", 100, "tz1", 1))

	delegations, err := store.GetDelegations(ctx, "2024")
	assert.NoError(t, err)
	assert.Len(t, delegations, 1, "Expected one delegations fetched for year 2024")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT timestamp, amount, delegator, block FROM delegations ORDER BY timestamp DESC")).
		WillReturnRows(sqlmock.NewRows([]string{"timestamp", "amount", "delegator", "block"}).
			AddRow("2024-04-21T16:23:27Z", 200, "tz2", 2).
			AddRow("2023-04-21T16:23:27Z", 300, "tz3", 3))

	allDelegations, err := store.GetDelegations(ctx, "")
	assert.NoError(t, err)
	assert.Len(t, allDelegations, 2, "Expected two delegation fetched for all years")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT timestamp, amount, delegator, block FROM delegations ORDER BY timestamp DESC")).
		WillReturnError(sql.ErrConnDone)

	_, err = store.GetDelegations(ctx, "")
	assert.Error(t, err)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT timestamp, amount, delegator, block FROM delegations ORDER BY timestamp DESC")).
		WillReturnRows(sqlmock.NewRows([]string{"timestamp", "amount", "delegator", "block"}).
			AddRow(time.Now(), "not-a-number", "delegator4", "not-a-number"))

	_, err = store.GetDelegations(ctx, "")
	assert.Error(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestDeleteDelegationsFromLevel(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	level := uint64(10)

	mock.ExpectPrepare(regexp.QuoteMeta("DELETE FROM delegations WHERE level >= $1")).
		ExpectExec().
		WithArgs(level).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.DeleteDelegationsFromLevel(ctx, level)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	mock.ExpectPrepare(regexp.QuoteMeta("DELETE FROM delegations WHERE level >= $1")).
		WillReturnError(sql.ErrConnDone)

	err = store.DeleteDelegationsFromLevel(ctx, level)
	assert.Error(t, err)

	err = store.DeleteDelegationsFromLevel(ctx, level)
	assert.Error(t, err)
}
