package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"carousel/internal/models"
	"github.com/jackc/pgx/v5"
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
	fmt.Println("Starting FetchUsers with periodEvent:", periodEvent)
	// Cache key based on scope
	cacheKey := buildCacheKey(periodEvent)
	// fmt.Printf("periodEvent: %+v\n %s\n", periodEvent, cacheKey)
	// fmt.Printf("cacheKey: %+v\n", cacheKey)
	cached, err := pd.p.cache.Get(ctx, cacheKey)
	if err == nil {
		var users []models.CustomUser
		if err := json.Unmarshal([]byte(cached), &users); err == nil {
			return users, nil
		}
	}
	var filterType string
	var filterID int64

	// Dynamic query based on scope
	query := `
		SELECT id, username, role, salary, preferred_language, timezone
		FROM cre_customuser
		WHERE status IN ('active', 'on_leave')
	`
	var args []interface{}
	// Remove the variable declarations and use them directly in the args append
	if branch, ok := periodEvent["branch_id"].(int64); ok && branch != 0 {
		fmt.Println("In branch:", branch)
		branchID := branch
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_branches WHERE branch_id = $1::bigint)"
		args = append(args, &branchID)
		filterType = "branch"
		filterID = branch
	} else if restaurant, ok := periodEvent["restaurant_id"].(int64); ok && restaurant != 0 {
		fmt.Println("In restaurant:", restaurant)
		restaurantID := restaurant
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_restaurants WHERE restaurant_id = $1::bigint)"
		args = append(args, &restaurantID)
		filterType = "restaurant"
		filterID = restaurant
	} else if company, ok := periodEvent["company_id"].(int64); ok && company != 0 {
		companyID := company
		query += " AND id IN (SELECT customuser_id FROM cre_customuser_companies WHERE company_id = $1::bigint)"
		args = append(args, &companyID)
		filterType = "company"
		filterID = company
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
		
		if filterType == "branch" {
			user.Branches = []int64{filterID}
		} else if filterType == "restaurant" {
			user.Restaurants = []int64{filterID}
		} else if filterType == "company" {
			user.Companies = []int64{filterID}
		}
		
		users = append(users, user)
	}

	// Cache the result
	// usersJSON, err := json.Marshal(users)
	// if err == nil {
	// 	pd.p.cache.Set(ctx, cacheKey, usersJSON, 3600*time.Second) // 1-hour TTL
	// }

	return users, nil
}

// FetchRules retrieves rules based on scope from periodEvent
func (pd *PayrollDB) FetchRules(ctx context.Context, periodEvent map[string]interface{}) ([]models.Rule, []int64, error) {
	fmt.Println("Starting FetchRules with periodEvent:", periodEvent)

	// Step 1: Derive complete scope hierarchy
	var branchID, restaurantID, companyID *int64
	if v, ok := periodEvent["branch_id"]; ok {
		if id, ok := v.(int64); ok {
			branchID = &id
			fmt.Println("Branch ID found:", *branchID)
		}
	}
	if v, ok := periodEvent["restaurant_id"]; ok {
		if id, ok := v.(int64); ok {
			restaurantID = &id
			fmt.Println("Restaurant ID found:", *restaurantID)
		}
	}
	if v, ok := periodEvent["company_id"]; ok {
		if id, ok := v.(int64); ok {
			companyID = &id
			fmt.Println("Company ID found:", *companyID)
		}
	}

	// Derive missing scope details from branch or restaurant
	if branchID != nil {
		var rID, cID int64
		err := pd.p.pool.QueryRow(ctx, `
			SELECT restaurant_id, company_id FROM cre_branch WHERE id = $1
		`, *branchID).Scan(&rID, &cID)
		if err == nil {
			restaurantID = &rID
			companyID = &cID
			fmt.Println("Derived Restaurant ID:", *restaurantID, "Company ID:", *companyID)
		} else if err != sql.ErrNoRows {
			fmt.Println("Error deriving scope from branch:", err)
			return nil, nil, err
		}
	} else if restaurantID != nil {
		var cID int64
		err := pd.p.pool.QueryRow(ctx, `
			SELECT company_id FROM cre_restaurant WHERE id = $1
		`, *restaurantID).Scan(&cID)
		if err == nil {
			companyID = &cID
			fmt.Println("Derived Company ID:", *companyID)
		} else if err != sql.ErrNoRows {
			fmt.Println("Error deriving scope from restaurant:", err)
			return nil, nil, err
		}
	}
	fmt.Println("Final scope: BranchID:", branchID, "RestaurantID:", restaurantID, "CompanyID:", companyID)

	// Step 2: Fetch rules in separate sets
	var branchRules, restaurantRules, companyRules []models.Rule

	if branchID != nil {
		rows, err := pd.p.pool.Query(ctx, `
			SELECT id, name, rule_type, amount, scope, company_id, restaurant_id, branch_id, priority, is_active
			FROM payroll_rule
			WHERE is_active = true AND effective_from <= $1 AND branch_id = $2
			ORDER BY priority DESC
		`, time.Now(), *branchID)
		if err != nil {
			fmt.Println("Error fetching branch rules:", err)
			return nil, nil, err
		}
		defer rows.Close()
		branchRules = scanRules(rows)
		fmt.Println("Fetched", len(branchRules), "branch rules")
	}

	if restaurantID != nil {
		rows, err := pd.p.pool.Query(ctx, `
			SELECT id, name, rule_type, amount, scope, company_id, restaurant_id, branch_id, priority, is_active
			FROM payroll_rule
			WHERE is_active = true AND effective_from <= $1 AND restaurant_id = $2
			ORDER BY priority DESC
		`, time.Now(), *restaurantID)
		if err != nil {
			fmt.Println("Error fetching restaurant rules:", err)
			return nil, nil, err
		}
		defer rows.Close()
		restaurantRules = scanRules(rows)
		fmt.Println("Fetched", len(restaurantRules), "restaurant rules")
	}

	if companyID != nil {
		rows, err := pd.p.pool.Query(ctx, `
			SELECT id, name, rule_type, amount, scope, company_id, restaurant_id, branch_id, priority, is_active
			FROM payroll_rule
			WHERE is_active = true AND effective_from <= $1 AND company_id = $2
			ORDER BY priority DESC
		`, time.Now(), *companyID)
		if err != nil {
			fmt.Println("Error fetching company rules:", err)
			return nil, nil, err
		}
		defer rows.Close()
		companyRules = scanRules(rows)
		fmt.Println("Fetched", len(companyRules), "company rules")
	}

	// Step 3: Filter rules by matching scope IDs
	if branchID != nil {
		branchRules = filterByScopeID(branchRules, branchID, restaurantID, companyID)
		fmt.Println("Filtered branch rules count:", len(branchRules))
	}
	if restaurantID != nil {
		restaurantRules = filterByScopeID(restaurantRules, branchID, restaurantID, companyID)
		fmt.Println("Filtered restaurant rules count:", len(restaurantRules))
	}
	if companyID != nil {
		companyRules = filterByScopeID(companyRules, branchID, restaurantID, companyID)
		fmt.Println("Filtered company rules count:", len(companyRules))
	}

	// Step 4: Apply priority-based deduplication
	if branchRules != nil {
		restaurantRules = removeMatchingRules(restaurantRules, branchRules)
		companyRules = removeMatchingRules(companyRules, branchRules)
	}
	if restaurantRules != nil {
		companyRules = removeMatchingRules(companyRules, restaurantRules)
	}
	fmt.Println("After deduplication - Branch:", len(branchRules), "Restaurant:", len(restaurantRules), "Company:", len(companyRules))

	// Step 5: Combine and extract rule IDs
	allRules := append(branchRules, restaurantRules...)
	allRules = append(allRules, companyRules...)
	var ruleIDs []int64
	ruleMap := make(map[int64]struct{})
	for _, rule := range allRules {
		if _, exists := ruleMap[rule.ID]; !exists {
			ruleMap[rule.ID] = struct{}{}
			ruleIDs = append(ruleIDs, rule.ID)
		}
	}
	fmt.Println("Combined rules count:", len(allRules), "Unique rule IDs:", len(ruleIDs))

	return allRules, ruleIDs, nil
}

// filterByScopeID filters rules by matching branch/restaurant/company IDs from periodEvent
func filterByScopeID(rules []models.Rule, branchID, restaurantID, companyID *int64) []models.Rule {
	var filtered []models.Rule
	for _, rule := range rules {
		// Check branch mismatch
		if branchID != nil && rule.BranchID != nil && *rule.BranchID != *branchID {
			fmt.Println("Removing rule", rule.ID, "due to branch ID mismatch:", *rule.BranchID, "!=", *branchID)
			continue
		}
		// Check restaurant mismatch
		if restaurantID != nil && rule.RestaurantID != nil && *rule.RestaurantID != *restaurantID {
			fmt.Println("Removing rule", rule.ID, "due to restaurant ID mismatch:", *rule.RestaurantID, "!=", *restaurantID)
			continue
		}
		// Check company mismatch
		if companyID != nil && rule.CompanyID != nil && *rule.CompanyID != *companyID {
			fmt.Println("Removing rule", rule.ID, "due to company ID mismatch:", *rule.CompanyID, "!=", *companyID)
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

// scanRules scans rows into a slice of Rule models
func scanRules(rows pgx.Rows) []models.Rule {
	var rules []models.Rule
	for rows.Next() {
		var rule models.Rule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Type, &rule.Amount,
			&rule.Scope, &rule.CompanyID, &rule.RestaurantID, &rule.BranchID, &rule.Priority, &rule.IsActive); err != nil {
			fmt.Println("Scan error:", err)
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

// removeMatchingRules removes rules from target set that match IDs in source set
func removeMatchingRules(target, source []models.Rule) []models.Rule {
	sourceMap := make(map[int64]struct{})
	for _, r := range source {
		sourceMap[r.ID] = struct{}{}
	}
	var filtered []models.Rule
	for _, r := range target {
		if _, exists := sourceMap[r.ID]; !exists {
			filtered = append(filtered, r)
		}
	}
	return filtered
}


// FetchRuleTargets retrieves targets based on ruleIDs and associates with users
func (pd *PayrollDB) FetchRuleTargets(ctx context.Context, ruleIDs []int64, users []models.CustomUser) ([]models.MappedTarget, []models.UnmappedTarget, []models.RuleTarget, error) {
	fmt.Println("Starting FetchRuleTargets with ruleIDs:", ruleIDs, "users count:", len(users))

	// Step 1: Fetch rule targets for the given ruleIDs
	query, args, err := buildRuleTargetsQuery(ruleIDs)
	if err != nil {
		fmt.Println("Error building query:", err)
		return nil, nil, nil, err
	}

	rows, err := pd.p.pool.Query(ctx, query, args...)
	if err != nil {
		fmt.Println("Error fetching rule targets:", err)
		return nil, nil, nil, err
	}
	defer rows.Close()

	var allTargets []models.RuleTarget
	for rows.Next() {
		var target models.RuleTarget
		if err := rows.Scan(&target.RuleID, &target.TargetType, &target.TargetValue, &target.BranchID); err != nil {
			fmt.Println("Scan error:", err)
			return nil, nil, nil, err
		}
		allTargets = append(allTargets, target)
	}
	fmt.Println("Fetched", len(allTargets), "rule targets")

	// Step 2: Associate targets with users
	mappedTargets := []models.MappedTarget{}
	userMap := make(map[int64]models.CustomUser) // Map user ID to user
	roleMap := make(map[string][]int64)         // Map role to user IDs
	for _, user := range users {
		userMap[user.ID] = user
		if user.Role != "" { // Single role as string
			roleMap[user.Role] = append(roleMap[user.Role], user.ID)
		}
	}

	usedRuleIDs := make(map[int64]struct{})
	mappedUserIDs := make(map[int64]struct{})
	for _, target := range allTargets {
		if _, ok := usedRuleIDs[target.RuleID]; !ok {
			var associatedUserIDs []int64
			if target.TargetType == "user" {
				if userID, err := parseInt64(target.TargetValue); err == nil {
					if _, exists := userMap[userID]; exists {
						associatedUserIDs = []int64{userID}
						mappedUserIDs[userID] = struct{}{}
						fmt.Println("Mapped user", userID, "to rule", target.RuleID)
					}
				}
			} else if target.TargetType == "role" {
				if userIDs, exists := roleMap[target.TargetValue]; exists {
					associatedUserIDs = userIDs
					for _, userID := range userIDs {
						mappedUserIDs[userID] = struct{}{}
						fmt.Println("Mapped user", userID, "to rule", target.RuleID, "via role", target.TargetValue)
					}
				}
			}
			if len(associatedUserIDs) > 0 {
				mappedTargets = append(mappedTargets, models.MappedTarget{
					RuleID:         target.RuleID,
					AssociatedUserIDs: associatedUserIDs,
				})
				usedRuleIDs[target.RuleID] = struct{}{}
			}
		}
	}

	// Identify unmapped users and rules
	var unmappedTargets []models.UnmappedTarget
	remainingRuleIDs := difference(ruleIDs, usedRuleIDs)
	for _, user := range users {
		if _, mapped := mappedUserIDs[user.ID]; !mapped {
			unmappedTargets = append(unmappedTargets, models.UnmappedTarget{
				UserID:  user.ID,
				RuleIDs: remainingRuleIDs,
			})
			fmt.Println("Unmapped user", user.ID, "with rules:", remainingRuleIDs)
		}
	}

	// Step 3: Organize and return
	fmt.Println("Mapped targets count:", len(mappedTargets), "Unmapped targets count:", len(unmappedTargets), "All targets count:", len(allTargets))
	return mappedTargets, unmappedTargets, allTargets, nil
}

// buildRuleTargetsQuery constructs a parameterized query for ruleIDs
func buildRuleTargetsQuery(ruleIDs []int64) (string, []interface{}, error) {
	if len(ruleIDs) == 0 {
		return "", nil, fmt.Errorf("no rule IDs provided")
	}

	query := "SELECT rule_id, target_type, target_value, branch_id FROM payroll_ruletarget WHERE rule_id IN ("
	args := make([]interface{}, len(ruleIDs))
	for i, id := range ruleIDs {
		args[i] = id
		query += fmt.Sprintf("$%d,", i+1)
	}
	query = query[:len(query)-1] + ")"

	return query, args, nil
}

// parseInt64 converts string to int64, handling potential errors
func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// difference returns elements in a that are not in b
func difference(a []int64, b map[int64]struct{}) []int64 {
	var diff []int64
	for _, item := range a {
		if _, exists := b[item]; !exists {
			diff = append(diff, item)
		}
	}
	return diff
}

// FetchOverrides retrieves overrides based on ruleIDs and periodID, applying actions and user matching
func (pd *PayrollDB) FetchOverrides(ctx context.Context, ruleIDs []int64, periodID int64, rules []models.Rule, branchID *int64) ([]models.MatchedOverride, []models.Override, []models.Override, []models.Rule, error) {
	fmt.Println("Starting FetchOverrides with ruleIDs:", ruleIDs, "periodID:", periodID, "branchID:", branchID)

	// Step 1: Fetch overrides for the given ruleIDs without periodID filter
	query, args, err := buildOverridesQuery(ruleIDs)
	if err != nil {
		fmt.Println("Error building query:", err)
		return nil, nil, nil, nil, err
	}

	rows, err := pd.p.pool.Query(ctx, query, args...)
	if err != nil {
		fmt.Println("Error fetching overrides:", err)
		return nil, nil, nil, nil, err
	}
	defer rows.Close()

	var totalOverrides []models.Override
	for rows.Next() {
		var override models.Override
		var amount sql.NullFloat64
		if err := rows.Scan(&override.RuleID, &override.PeriodID, &override.UserID, &override.OverrideType,
			&amount, &override.BranchID, &override.Notes); err != nil {
			fmt.Println("Scan error:", err)
			return nil, nil, nil, nil, err
		}
		if amount.Valid {
			override.Amount = amount.Float64
		}
		totalOverrides = append(totalOverrides, override)
	}
	fmt.Println("Fetched", len(totalOverrides), "total overrides")

	// Post-Fetch Filtering: Remove overrides with mismatched period_id (if period_id is set) and branch_id (if branchID is set)
	var filteredOverrides []models.Override
	for _, override := range totalOverrides {
		// Check period mismatch
		if override.PeriodID != nil && *override.PeriodID != periodID {
			fmt.Println("Removing override for rule", override.RuleID, "due to mismatched period_id", *override.PeriodID)
			continue
		}
		// Check branch mismatch only if branchID is not nil
		if branchID != nil && override.BranchID != nil && *branchID != *override.BranchID {
			fmt.Println("Removing override for rule", override.RuleID, "due to mismatched branch_id", *override.BranchID)
			continue
		}
		filteredOverrides = append(filteredOverrides, override)
	}
	fmt.Println("Filtered to", len(filteredOverrides), "overrides after period and branch checks")

	// Step 2: Apply actions based on override_type
	ruleMap := make(map[int64]*models.Rule)
	for i := range rules {
		ruleMap[rules[i].ID] = &rules[i]
	}

	var processedOverrides []models.Override
	for _, override := range filteredOverrides {
		if rule, exists := ruleMap[override.RuleID]; exists {
			switch override.OverrideType {
			case "replace":
				fmt.Println("Replacing rule", override.RuleID, "with override")
		
				override.Type = rule.Type // Copy type before deletion
				override.Name = rule.Name // Copy name before deletion
				// delete(ruleMap, override.RuleID) // Remove associated rule 
				processedOverrides = append(processedOverrides, override)
			case "add":
				fmt.Println("Adding override to rule", override.RuleID)
				// rule.Amount += override.Amount
				processedOverrides = append(processedOverrides, override)
			case "subtract":
				fmt.Println("Subtracting override from rule", override.RuleID)
				// rule.Amount -= override.Amount
				processedOverrides = append(processedOverrides, override)
			default:
				processedOverrides = append(processedOverrides, override)
			}
		} else {
			processedOverrides = append(processedOverrides, override)
		}
	}
	fmt.Println("Processed", len(processedOverrides), "overrides after actions")

	// Step 3: Match user-specific overrides
	var matchedOverrides []models.MatchedOverride
	userMap := make(map[int64]struct{})
	// fmt.Printf("before loop - override: %v\n", processedOverrides)
	for i := 0; i < len(processedOverrides); i++ {
		override := processedOverrides[i]
		// fmt.Printf("override: %v\n", override)
		if override.UserID != nil {
			userID := *override.UserID
			if _, exists := userMap[userID]; !exists {
				fmt.Println("Matching override for user", userID, "with rule", override.RuleID)
				matchedOverrides = append(matchedOverrides, models.MatchedOverride{
					Override: override,
					UserID:   userID,
				})
				userMap[userID] = struct{}{}
			}
			// Remove the current override from processedOverrides
			processedOverrides = removeOverride(processedOverrides, override)
			i-- // Adjust index since slice is shortened
		}
	}
	fmt.Println("Matched", len(matchedOverrides), "user-specific overrides")

	// Step 4: Organize and return
	var updatedRules []models.Rule
	for _, rule := range ruleMap {
		updatedRules = append(updatedRules, *rule)
	}
	fmt.Println("Unmatched overrides count:", len(processedOverrides), "Total overrides count:", len(totalOverrides), "Updated rules count:", len(updatedRules))
	return matchedOverrides, processedOverrides, totalOverrides, updatedRules, nil
}

// buildOverridesQuery constructs a parameterized query for ruleIDs
func buildOverridesQuery(ruleIDs []int64) (string, []interface{}, error) {
	if len(ruleIDs) == 0 {
		return "", nil, fmt.Errorf("no rule IDs provided")
	}

	query := "SELECT rule_id, period_id, user_id, override_type, amount, branch_id, notes FROM payroll_override WHERE rule_id IN ("
	args := make([]interface{}, len(ruleIDs)+1) // Only for ruleIDs and time.Now()
	for i, id := range ruleIDs {
		args[i] = id
		query += fmt.Sprintf("$%d,", i+1)
	}
	query = query[:len(query)-1] + ") AND effective_from <= $" + fmt.Sprint(len(ruleIDs)+1) +
		" AND (expires_at IS NULL OR expires_at >= $" + fmt.Sprint(len(ruleIDs)+1) + ")"
	args[len(ruleIDs)] = time.Now()

	return query, args, nil
}

// removeOverride removes an override from the slice
func removeOverride(overrides []models.Override, target models.Override) []models.Override {
    var result []models.Override
    for _, ov := range overrides {
        // Compare UserID value if not nil, otherwise compare other fields
        userMatch := (ov.UserID == nil && target.UserID == nil) || (ov.UserID != nil && target.UserID != nil && *ov.UserID == *target.UserID)
        if !(ov.RuleID == target.RuleID && userMatch && ov.OverrideType == target.OverrideType) {
            result = append(result, ov)
        }
    }
    return result
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
	fmt.Printf("Event: %+v\n", event)
	if branch, ok := event["branch_id"].(int64); ok {
		return fmt.Sprintf("users:branch:%d", branch)
	} else if restaurant, ok := event["restaurant_id"].(int64); ok {
		return fmt.Sprintf("users:restaurant:%d", restaurant)
	} else if company, ok := event["company_id"].(int64); ok {
		return fmt.Sprintf("users:company:%d", company)
	}
	return "users:default"
}

// StorePayrollRecord stores a payroll record in the database within a transaction
func (pd *PayrollDB) StorePayrollRecord(ctx context.Context, record models.Record) error {
	tx, err := pd.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var recordID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO payroll_record (user_id, branch_id, period_id, base_salary, total_bonus, total_deduction, net_pay, status, generated_at, bonus_details, deduction_details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb)
		RETURNING id
	`, record.UserID, record.BranchID, record.PeriodID, record.BaseSalary, record.TotalBonus, record.TotalDeduction,
		record.NetPay, record.Status, record.GeneratedAt, record.BonusDetails, record.DeductionDetails).Scan(&recordID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}