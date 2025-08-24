package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"carousel/internal/models"
	"github.com/jackc/pgx/v5/pgxpool" 
)

// PayrollDB handles payroll-related database operations
type PayrollDB struct {
	p *Postgres
}

// NewPayrollDB creates a new PayrollDB instance
func NewPayrollDB(p *Postgres) *PayrollDB {
	return &PayrollDB{p: p}
}

// BeginTx starts a new transaction
func (pd *PayrollDB) BeginTx(ctx context.Context) (*pgxpool.Tx, error) {
    tx, err := pd.p.Begin(ctx)
    if err != nil {
        return nil, err
    }
    return tx.(*pgxpool.Tx), nil // Type assertion
}

// FetchUsers retrieves CustomUsers for a given scope in batches of 500
func (pd *PayrollDB) FetchUsers(ctx context.Context, periodEvent map[string]interface{}, offset int) ([]models.CustomUser, error) {
	// Cache key based on scope
	fmt.Printf("periodEvent: %+v\n", periodEvent)
	cacheKey := buildCacheKey(periodEvent)
	cached, err := pd.p.cache.Get(ctx, cacheKey)
	if err == nil {
		var users []models.CustomUser
		if err := json.Unmarshal([]byte(cached), &users); err == nil {
			return users, nil
		}
	}

	// Dynamic query based on scope
	query := `
		SELECT id, username, role, salary, preferred_language, timezone
		FROM cre_customuser
		WHERE status IN ('active', 'on_leave')
	`
	var args []interface{}
	// Remove the variable declarations and use them directly in the args append
	if branch, ok := periodEvent["branch_id"].(int64); ok {
		branchID := branch
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_branches WHERE branch_id = $1::bigint)"
		args = append(args, &branchID)
	} else if restaurant, ok := periodEvent["restaurant_id"].(int64); ok {
		restaurantID := restaurant
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_restaurants WHERE restaurant_id = $1::bigint)"
		args = append(args, &restaurantID)
	} else if company, ok := periodEvent["company_id"].(int64); ok {
		companyID := company
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_companies WHERE company_id = $1::bigint)"
		args = append(args, &companyID)
	}

	query += " LIMIT 500 OFFSET $2"
	args = append(args, offset)

	// fmt.Printf("Executing query: %s with args: %+v\n", query, args)
	rows, err := pd.p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.CustomUser
	for rows.Next() {
		var user models.CustomUser
		var roleStr string
		if err := rows.Scan(&user.ID, &user.Username, &roleStr, &user.Salary, &user.PreferredLanguage, &user.Timezone); err != nil {
			return nil, err
		}
		user.Role = roleStr
		users = append(users, user)
	}

	// Cache the result
	usersJSON, err := json.Marshal(users)
	if err == nil {
		pd.p.cache.Set(ctx, cacheKey, usersJSON, 3600*time.Second) // 1-hour TTL
	}

	return users, nil
}

// FetchRules retrieves active rules
func (pd *PayrollDB) FetchRules(ctx context.Context) ([]models.Rule, error) {
	rows, err := pd.p.pool.Query(ctx, `
		SELECT id, name, rule_type, amount, percentage, scope, company_id, restaurant_id, branch_id, priority, is_active
		FROM payroll_rule
		WHERE is_active = true AND effective_from <= $1
		ORDER BY priority DESC
	`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.Rule
	for rows.Next() {
		var rule models.Rule
		var percentage sql.NullFloat64
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Type, &rule.Amount, &percentage,
			&rule.Scope, &rule.CompanyID, &rule.RestaurantID, &rule.BranchID, &rule.Priority, &rule.IsActive); err != nil {
			return nil, err
		}
		if percentage.Valid {
			rule.Percentage = &percentage.Float64
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// FetchRuleTargets retrieves rule targets
func (pd *PayrollDB) FetchRuleTargets(ctx context.Context) ([]models.RuleTarget, error) {
	rows, err := pd.p.pool.Query(ctx, `
		SELECT rule_id, target_type, target_value, branch_id
		FROM payroll_ruletarget
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []models.RuleTarget
	for rows.Next() {
		var target models.RuleTarget
		if err := rows.Scan(&target.RuleID, &target.TargetType, &target.TargetValue, &target.BranchID); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

// FetchOverrides retrieves overrides for a period
func (pd *PayrollDB) FetchOverrides(ctx context.Context, periodID int64) ([]models.Override, error) {
	rows, err := pd.p.pool.Query(ctx, `
		SELECT rule_id, period_id, user_id, override_type, amount, percentage, branch_id, notes
		FROM payroll_override
		WHERE period_id = $1 AND effective_from <= $2 AND (expires_at IS NULL OR expires_at >= $2)
	`, periodID, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []models.Override
	for rows.Next() {
		var override models.Override
		var amount, percentage sql.NullFloat64
		if err := rows.Scan(&override.RuleID, &override.PeriodID, &override.UserID, &override.OverrideType,
			&amount, &percentage, &override.BranchID, &override.Notes); err != nil {
			return nil, err
		}
		if amount.Valid {
			override.Amount = &amount.Float64
		}
		if percentage.Valid {
			override.Percentage = &percentage.Float64
		}
		overrides = append(overrides, override)
	}
	return overrides, nil
}

// LogActivity logs a payroll-related activity using the branch_activity table
func (pd *PayrollDB) LogActivity(ctx context.Context, activityType string, details map[string]interface{}, branchID, userID *int64) error {
	jsonDetails, err := json.Marshal(details)
	if err != nil {
		return err
	}

	_, err = pd.p.pool.Exec(ctx, `
		INSERT INTO notifications_branchactivity (activity_type, timestamp, details, branch_id, user_id)
		VALUES ($1, $2, $3::jsonb, $4, $5)
	`, activityType, time.Now(), jsonDetails, branchID, userID)
	return err
}

func buildCacheKey(event map[string]interface{}) string {
	if branch, ok := event["branch"].(int); ok {
		return fmt.Sprintf("users:branch:%d", branch)
	} else if restaurant, ok := event["restaurant"].(int); ok {
		return fmt.Sprintf("users:restaurant:%d", restaurant)
	} else if company, ok := event["company"].(int); ok {
		return fmt.Sprintf("users:company:%d", company)
	}
	return "users:default"
}