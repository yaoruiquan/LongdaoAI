package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.VideoRequestBillingEvidenceReader = (*usageBillingRepository)(nil)

func (r *usageBillingRepository) FindVideoRequestBillingEvidence(ctx context.Context, apiKeyID, userID int64, ids []string) (*service.VideoRequestBillingEvidence, error) {
	if r == nil || r.db == nil || apiKeyID <= 0 || userID <= 0 || len(ids) == 0 || len(ids) > 3 {
		return nil, errors.New("invalid video request billing lookup")
	}
	args := []any{apiKeyID, userID}
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", i+3)
	}
	// Scope by BOTH key and user. Do not filter non-video rows here: a collision
	// with any other usage row must remain ambiguous. Dedup is committed in the
	// charging transaction; querying usage logs alone does not prove a charge.
	rows, err := r.db.QueryContext(ctx, `SELECT ul.requested_model, COALESCE(ul.billing_mode, ''), ul.video_count,
		ul.video_duration_seconds, ul.video_resolution, ul.actual_cost::text,
		(EXISTS (SELECT 1 FROM usage_billing_dedup d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id)
		 OR EXISTS (SELECT 1 FROM usage_billing_dedup_archive d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id))
		FROM usage_logs ul WHERE ul.api_key_id = $1 AND ul.user_id = $2 AND ul.request_id IN (`+strings.Join(placeholders, ",")+`) LIMIT 2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result *service.VideoRequestBillingEvidence
	count := 0
	for rows.Next() {
		count++
		item := &service.VideoRequestBillingEvidence{}
		var model, resolution, amount sql.NullString
		var videoCount, duration sql.NullInt64
		if err := rows.Scan(&model, &item.BillingMode, &videoCount, &duration, &resolution, &amount, &item.TransactionConfirmed); err != nil {
			return nil, err
		}
		if model.Valid {
			item.Model = &model.String
		}
		if resolution.Valid {
			item.Resolution = &resolution.String
		}
		if amount.Valid {
			item.ActualCostUSD = &amount.String
		}
		if videoCount.Valid {
			value := int(videoCount.Int64)
			item.VideoCount = &value
		}
		if duration.Valid {
			value := int(duration.Int64)
			item.DurationSeconds = &value
		}
		result = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, nil
	}
	return result, nil
}
