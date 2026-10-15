//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestFindRequestBillingEvidenceReturnsUniqueUsageAndCustomerCost(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(ul.requested_model`).
		WithArgs(int64(7), int64(11), "client:req-1", "req-1").
		WillReturnRows(sqlmock.NewRows([]string{"model", "billing_mode", "image_count", "actual_cost", "confirmed"}).
			AddRow("gpt-image-1", "image", 2, "0.0125000000", true))

	evidence, err := (&usageBillingRepository{db: db}).FindRequestBillingEvidence(context.Background(), 7, 11, []string{"client:req-1", "req-1"})
	require.NoError(t, err)
	require.NotNil(t, evidence)
	require.Equal(t, "gpt-image-1", evidence.Model)
	require.Equal(t, "image", evidence.BillingMode)
	require.Equal(t, 2, evidence.ImageCount)
	require.Equal(t, "0.0125000000", *evidence.ActualCostUSD)
	require.True(t, evidence.TransactionConfirmed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindRequestBillingEvidenceRejectsAmbiguousUsageRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(ul.requested_model`).
		WithArgs(int64(7), int64(11), "req-1", "client:req-1").
		WillReturnRows(sqlmock.NewRows([]string{"model", "billing_mode", "image_count", "actual_cost", "confirmed"}).
			AddRow("gpt-image-1", "image", 1, "0.01", true).
			AddRow("gpt-image-2", "image", 1, "0.02", true))

	evidence, err := (&usageBillingRepository{db: db}).FindRequestBillingEvidence(context.Background(), 7, 11, []string{"req-1", "client:req-1"})
	require.NoError(t, err)
	require.Nil(t, evidence)
	require.NoError(t, mock.ExpectationsWereMet())
}
