package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageBillingRepository) FindRequestBillingEvidence(ctx context.Context, apiKeyID, userID int64, ids []string) (*service.RequestBillingEvidence, error) {
	if r == nil || r.db == nil || apiKeyID <= 0 || userID <= 0 || len(ids) == 0 || len(ids) > 3 {
		return nil, errors.New("invalid request billing lookup")
	}
	args := []any{apiKeyID, userID}
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", i+3)
	}
	// actual_cost is the user's billed amount; the dedup marker confirms that
	// the corresponding billing transaction committed.
	rows, err := r.db.QueryContext(ctx, `SELECT COALESCE(ul.requested_model, ''), COALESCE(ul.billing_mode, ''), COALESCE(ul.image_count, 0),
		ul.actual_cost::text,
		(EXISTS (SELECT 1 FROM usage_billing_dedup d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id)
		 OR EXISTS (SELECT 1 FROM usage_billing_dedup_archive d WHERE d.api_key_id = ul.api_key_id AND d.request_id = ul.request_id))
		FROM usage_logs ul WHERE ul.api_key_id = $1 AND ul.user_id = $2 AND ul.request_id IN (`+strings.Join(placeholders, ",")+`) LIMIT 2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result *service.RequestBillingEvidence
	count := 0
	for rows.Next() {
		count++
		item := &service.RequestBillingEvidence{}
		var actualCost sql.NullString
		if err := rows.Scan(&item.Model, &item.BillingMode, &item.ImageCount, &actualCost, &item.TransactionConfirmed); err != nil {
			return nil, err
		}
		if actualCost.Valid {
			item.ActualCostUSD = &actualCost.String
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
