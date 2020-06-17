package dbtest

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbops "isc.org/stork/server/database"
)


// Tests the logic that creates new transaction or returns an
// existing one.
func TestTransaction(t *testing.T) {
	db, _, teardown := SetupDatabaseTestCase(t)
	defer teardown()

	// Start new transaction.
	tx, rollback, commit, err := dbops.Transaction(db)
	require.NotNil(t, tx)
	require.NoError(t, err)
	require.NotNil(t, rollback)
	require.NotNil(t, commit)
	// Check that the commit operation returns no error.
	err = commit()
	require.NoError(t, err)

	// Start new transaction here.
	tx, err = db.Begin()
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback()
	}()
	require.NotNil(t, tx)

	// This time pass the transaction to the function under test. The function
	// should determine that the transaction was already started and return
	// it back to the caller.
	tx2, rollback, commit, err := dbops.Transaction(tx)
	require.NoError(t, err)
	require.NotNil(t, rollback)
	defer rollback()
	require.NotNil(t, tx2)
	require.NotNil(t, commit)
	// Those two pointers should point at the same object.
	require.Same(t, tx, tx2)
}
