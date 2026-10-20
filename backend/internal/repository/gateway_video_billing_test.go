//go:build unit

package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// Assert ownership predicates, exact-request dedup (live or archive), and the
// two-row ambiguity bound in the actual SELECT, not only returned mock values.
const videoEvidenceSQL = `SELECT ul.requested_model, COALESCE(ul.billing_mode, ''), ul.video_count,
		ul.video_duration_seconds, ul.video_resolution, ul.actual_cost::text,
		(EXISTS (SELECT 1 FROM usage_billing_dedup d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id)
		 OR EXISTS (SELECT 1 FROM usage_billing_dedup_archive d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id))
		FROM usage_logs ul WHERE ul.api_key_id = $1 AND ul.user_id = $2 AND ul.request_id IN ($3,$4,$5) LIMIT 2`

func videoEvidenceRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"model", "billing_mode", "video_count", "duration_seconds", "resolution", "actual_cost", "confirmed"})
}

func TestFindVideoRequestBillingEvidenceScopedUniqueAndNullable(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rows      *sqlmock.Rows
		absent    bool
		null      bool
		confirmed bool
	}{
		{name: "confirmed", rows: videoEvidenceRows().AddRow("minimax-h3", "video", 1, 4, "720p", "0.1296000000", true), confirmed: true},
		{name: "unconfirmed retained", rows: videoEvidenceRows().AddRow("minimax-h3", "video", 1, 4, "720p", "0.1296000000", false)},
		{name: "nullable metadata", rows: videoEvidenceRows().AddRow(nil, "", nil, nil, nil, nil, false), null: true},
		{name: "foreign or missing usage", rows: videoEvidenceRows(), absent: true},
		{name: "duplicate original rows", rows: videoEvidenceRows().AddRow("minimax-h3", "video", 1, 4, "720p", "0.1", true).AddRow("minimax-h3", "video", 1, 4, "720p", "0.1", true), absent: true},
		{name: "alias collides with image", rows: videoEvidenceRows().AddRow("minimax-h3", "video", 1, 4, "720p", "0.1", true).AddRow("gpt-image-2", "image", 0, nil, nil, "0.2", true), absent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectQuery(regexp.QuoteMeta(videoEvidenceSQL)).WithArgs(int64(7), int64(11), "req-1", "client:req-1", "local:req-1").WillReturnRows(tc.rows).RowsWillBeClosed()
			evidence, err := (&usageBillingRepository{db: db}).FindVideoRequestBillingEvidence(context.Background(), 7, 11, []string{"req-1", "client:req-1", "local:req-1"})
			require.NoError(t, err)
			if tc.absent {
				require.Nil(t, evidence)
			} else {
				require.NotNil(t, evidence)
				require.Equal(t, tc.confirmed, evidence.TransactionConfirmed)
				if tc.null {
					require.Nil(t, evidence.Model)
					require.Nil(t, evidence.VideoCount)
					require.Nil(t, evidence.DurationSeconds)
					require.Nil(t, evidence.Resolution)
					require.Nil(t, evidence.ActualCostUSD)
				} else {
					require.Equal(t, "minimax-h3", *evidence.Model)
					require.Equal(t, "video", evidence.BillingMode)
					require.Equal(t, 1, *evidence.VideoCount)
					require.Equal(t, 4, *evidence.DurationSeconds)
					require.Equal(t, "720p", *evidence.Resolution)
					require.Equal(t, "0.1296000000", *evidence.ActualCostUSD)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindVideoRequestBillingEvidenceErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*sqlmock.ExpectedQuery)
	}{
		{name: "query", set: func(q *sqlmock.ExpectedQuery) { q.WillReturnError(errors.New("query failed")) }},
		{name: "row", set: func(q *sqlmock.ExpectedQuery) {
			q.WillReturnRows(videoEvidenceRows().AddRow(nil, "video", nil, nil, nil, nil, false).RowError(0, errors.New("row failed")))
		}},
		{name: "scan", set: func(q *sqlmock.ExpectedQuery) {
			q.WillReturnRows(videoEvidenceRows().AddRow(nil, "video", "invalid integer", nil, nil, nil, false))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tc.set(mock.ExpectQuery(`SELECT ul.requested_model`).WithArgs(int64(7), int64(11), "req-1"))
			evidence, err := (&usageBillingRepository{db: db}).FindVideoRequestBillingEvidence(context.Background(), 7, 11, []string{"req-1"})
			require.Error(t, err)
			require.Nil(t, evidence)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
	for _, repo := range []*usageBillingRepository{nil, {}} {
		_, err := repo.FindVideoRequestBillingEvidence(context.Background(), 7, 11, []string{"req-1"})
		require.Error(t, err)
	}
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &usageBillingRepository{db: db}
	for _, tc := range []struct {
		keyID, userID int64
		ids           []string
	}{{0, 11, []string{"req-1"}}, {7, 0, []string{"req-1"}}, {7, 11, nil}, {7, 11, []string{"a", "b", "c", "d"}}} {
		_, err := repo.FindVideoRequestBillingEvidence(context.Background(), tc.keyID, tc.userID, tc.ids)
		require.Error(t, err)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
