package db

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
	"log"
	"strings"

    "carousel/internal/cache"
    "carousel/internal/models"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// SubscriptionDB handles database operations for subscriptions
type SubscriptionDB struct {
    Pool  *pgxpool.Pool
    Cache *cache.Redis
}

// NewSubscriptionDB initializes the database handler
func NewSubscriptionDB(pool *pgxpool.Pool, cache *cache.Redis) *SubscriptionDB {
    return &SubscriptionDB{Pool: pool, Cache: cache}
}

// CacheAllPlans caches all active plans in Redis
func (sdb *SubscriptionDB) CacheAllPlans(ctx context.Context) error {
    rows, err := sdb.Pool.Query(ctx, `
        SELECT plan_id, name, billing_type, monthly_price, included_credits, grace_credits, grace_days, is_active, created_at
        FROM subscriptions_plan WHERE is_active = TRUE
    `)
    if err != nil {
        return fmt.Errorf("failed to query plans: %v", err)
    }
    defer rows.Close()

    var plans []models.Plan
    for rows.Next() {
        var plan models.Plan
        err := rows.Scan(
            &plan.PlanID, &plan.Name, &plan.BillingType, &plan.MonthlyPrice, &plan.IncludedCredits,
            &plan.GraceCredits, &plan.GraceDays, &plan.IsActive, &plan.CreatedAt)
        if err != nil {
            return fmt.Errorf("failed to scan plan: %v", err)
        }
        plans = append(plans, plan)
    }

    data, err := json.Marshal(plans)
    if err != nil {
        return fmt.Errorf("failed to marshal plans: %v", err)
    }
    return sdb.Cache.Set(ctx, "plans:active", data, 24*time.Hour)
}

// GetPlan retrieves a plan by ID, checking cache first
func (sdb *SubscriptionDB) GetPlan(ctx context.Context, planID uuid.UUID) (*models.Plan, error) {
    cached, err := sdb.Cache.Get(ctx, "plans:active")
    if err == nil {
        var plans []models.Plan
        if err := json.Unmarshal([]byte(cached), &plans); err != nil {
            return nil, fmt.Errorf("failed to unmarshal plans: %v", err)
        }
        for _, plan := range plans {
            if plan.PlanID == planID {
                return &plan, nil
            }
        }
    }

    var plan models.Plan
    err = sdb.Pool.QueryRow(ctx, `
        SELECT plan_id, name, billing_type, monthly_price, included_credits, grace_credits, grace_days, is_active, created_at
        FROM subscriptions_plan WHERE plan_id = $1 AND is_active = TRUE
    `, planID).Scan(
        &plan.PlanID, &plan.Name, &plan.BillingType, &plan.MonthlyPrice, &plan.IncludedCredits,
        &plan.GraceCredits, &plan.GraceDays, &plan.IsActive, &plan.CreatedAt)
    if err == pgx.ErrNoRows {
        return nil, fmt.Errorf("plan %s not found or inactive", planID)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get plan: %v", err)
    }

    if err := sdb.CacheAllPlans(ctx); err != nil {
        fmt.Printf("Failed to refresh plans cache: %v\n", err)
    }

    return &plan, nil
}

// CacheAllFeatures caches all active features in Redis
func (sdb *SubscriptionDB) CacheAllFeatures(ctx context.Context) error {
    rows, err := sdb.Pool.Query(ctx, `
        SELECT feature_id, name, description, price, is_active, created_at
        FROM subscriptions_feature WHERE is_active = TRUE
    `)
    if err != nil {
        return fmt.Errorf("failed to query features: %v", err)
    }
    defer rows.Close()

    var features []models.Feature
    for rows.Next() {
        var feature models.Feature
        err := rows.Scan(
            &feature.FeatureID, &feature.Name, &feature.Description, &feature.Price,
            &feature.IsActive, &feature.CreatedAt)
        if err != nil {
            return fmt.Errorf("failed to scan feature: %v", err)
        }
        features = append(features, feature)
    }

    data, err := json.Marshal(features)
    if err != nil {
        return fmt.Errorf("failed to marshal features: %v", err)
    }
    return sdb.Cache.Set(ctx, "features:active", data, 24*time.Hour)
}

// GetFeature retrieves a feature by ID, checking cache first
func (sdb *SubscriptionDB) GetFeature(ctx context.Context, featureID uuid.UUID) (*models.Feature, error) {
    cached, err := sdb.Cache.Get(ctx, "features:active")
    if err == nil {
        var features []models.Feature
        if err := json.Unmarshal([]byte(cached), &features); err != nil {
            return nil, fmt.Errorf("failed to unmarshal features: %v", err)
        }
        for _, feature := range features {
            if feature.FeatureID == featureID {
                return &feature, nil
            }
        }
    }

    var feature models.Feature
    err = sdb.Pool.QueryRow(ctx, `
        SELECT feature_id, name, description, price, is_active, created_at
        FROM subscriptions_feature WHERE feature_id = $1 AND is_active = TRUE
    `, featureID).Scan(
        &feature.FeatureID, &feature.Name, &feature.Description, &feature.Price,
        &feature.IsActive, &feature.CreatedAt)
    if err == pgx.ErrNoRows {
        return nil, fmt.Errorf("feature %s not found or inactive", featureID)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get feature: %v", err)
    }

    if err := sdb.CacheAllFeatures(ctx); err != nil {
        fmt.Printf("Failed to refresh features cache: %v\n", err)
    }

    return &feature, nil
}

// CheckSubscriptionExists checks if a subscription exists
func (sdb *SubscriptionDB) CheckSubscriptionExists(ctx context.Context, subscriptionID uuid.UUID) (bool, error) {
    cached, err := sdb.Cache.Get(ctx, "subscription:"+subscriptionID.String())
    if err == nil && cached == "exists" {
        return true, nil
    }
    var exists bool
    err = sdb.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM subscriptions_subscription WHERE subscription_id = $1)", subscriptionID).Scan(&exists)
    if err != nil {
        return false, fmt.Errorf("failed to check subscription: %v", err)
    }
    if exists {
        sdb.Cache.Set(ctx, "subscription:"+subscriptionID.String(), []byte("exists"), 24*time.Hour)
    }
    return exists, nil
}

// CreateSubscription creates a new subscription
func (sdb *SubscriptionDB) CreateSubscription(ctx context.Context, tx pgx.Tx, subscription *models.Subscription, featureIDs []uuid.UUID) error {
    // Validate features
    for _, featureID := range featureIDs {
        feature, err := sdb.GetFeature(ctx, featureID)
        if err != nil {
            return fmt.Errorf("invalid feature %s: %v", featureID, err)
        }
        if !feature.IsActive {
            return fmt.Errorf("feature %s is not active", featureID)
        }
    }

    // Get plan for billing type and included credits
    plan, err := sdb.GetPlan(ctx, subscription.PlanID)
    if err != nil {
        return fmt.Errorf("failed to get plan: %v", err)
    }

    // Set balance based on included_credits for pay_per_order
    if plan.BillingType == models.BillingTypePayPerOrder {
        subscription.Balance = plan.IncludedCredits
    }

    // Set current_period_end based on billing type
    if plan.BillingType == models.BillingTypeMonthlyFixed {
		currentPeriodEnd := subscription.CurrentPeriodStart.AddDate(0, 1, 0)
		subscription.CurrentPeriodEnd = &currentPeriodEnd
	} else if plan.BillingType == models.BillingTypeYearlyFixed {
		currentPeriodEnd := subscription.CurrentPeriodStart.AddDate(1, 0, 0)
		subscription.CurrentPeriodEnd = &currentPeriodEnd
	} else {
		subscription.CurrentPeriodEnd = nil
	}

    // Set auto_renew to false
    subscription.AutoRenew = false

	// Set trial_end_date based on plan's trial_days
	if plan.TrialDays > 0 {
		trialEndDate := subscription.StartDate.AddDate(0, 0, plan.TrialDays)
		subscription.TrialEndDate = &trialEndDate
	} else {
		subscription.TrialEndDate = nil
	}

    _, err = tx.Exec(ctx, `
		INSERT INTO subscriptions_subscription (
			subscription_id, entity_type, entity_id, plan_id, status, start_date, trial_end_date,
			current_period_start, current_period_end, auto_renew, cancel_at_period_end, balance, plan_name, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, subscription.SubscriptionID, subscription.EntityType, subscription.EntityID, subscription.PlanID,
		subscription.Status, subscription.StartDate, subscription.TrialEndDate,
		subscription.CurrentPeriodStart, subscription.CurrentPeriodEnd, subscription.AutoRenew,
		subscription.CancelAtPeriodEnd, subscription.Balance, subscription.PlanName,
		subscription.CreatedAt, subscription.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to insert subscription: %v", err)
    }

    for _, featureID := range featureIDs {
        _, err = tx.Exec(ctx, `
            INSERT INTO subscriptions_subscription_features (subscription_id, feature_id)
            VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, subscription.SubscriptionID, featureID)
        if err != nil {
            return fmt.Errorf("failed to insert feature %s: %v", featureID, err)
        }
        _, err = tx.Exec(ctx, `
            INSERT INTO subscriptions_history (history_id, subscription_id, event_type, new_feature_id, created_at)
            VALUES ($1, $2, 'feature_added', $3, NOW())
        `, uuid.New(), subscription.SubscriptionID, featureID)
        if err != nil {
            return fmt.Errorf("failed to log feature addition: %v", err)
        }
    }

    _, err = tx.Exec(ctx, `
        INSERT INTO subscriptions_history (history_id, subscription_id, event_type, created_at)
        VALUES ($1, $2, 'created', NOW())
    `, uuid.New(), subscription.SubscriptionID)
    if err != nil {
        return fmt.Errorf("failed to log subscription creation: %v", err)
    }

    return nil
}

// CancelSubscription cancels a subscription
func (sdb *SubscriptionDB) CancelSubscription(ctx context.Context, tx pgx.Tx, subscriptionID uuid.UUID) (models.SubscriptionStatus, error) {
    var status models.SubscriptionStatus
    err := sdb.Pool.QueryRow(ctx, "SELECT status FROM subscriptions_subscription WHERE subscription_id = $1", subscriptionID).Scan(&status)
    if err == pgx.ErrNoRows {
        return "", fmt.Errorf("subscription %s not found", subscriptionID)
    }
    if err != nil {
        return "", fmt.Errorf("failed to check subscription: %v", err)
    }
    if status == models.StatusCanceled {
        return status, nil
    }

    _, err = tx.Exec(ctx, `
        UPDATE subscription_subscription
        SET status = 'canceled', updated_at = NOW()
        WHERE subscription_id = $1
    `, subscriptionID)
    if err != nil {
        return "", fmt.Errorf("failed to cancel subscription: %v", err)
    }

    _, err = tx.Exec(ctx, `
        INSERT INTO subscriptions_history (history_id, subscription_id, event_type, old_status, new_status, created_at)
        VALUES ($1, $2, 'canceled', $3, 'canceled', NOW())
    `, uuid.New(), subscriptionID, status)
    if err != nil {
        return "", fmt.Errorf("failed to log cancellation: %v", err)
    }

    return status, nil
}

// RenewSubscription renews a subscription
func (sdb *SubscriptionDB) RenewSubscription(ctx context.Context, tx pgx.Tx, subscriptionID uuid.UUID) (models.SubscriptionStatus, bool, error) {
    var status models.SubscriptionStatus
    var autoRenew bool
    err := sdb.Pool.QueryRow(ctx, "SELECT status, auto_renew FROM subscriptions_subscription WHERE subscription_id = $1", subscriptionID).Scan(&status, &autoRenew)
    if err == pgx.ErrNoRows {
        return "", false, fmt.Errorf("subscription %s not found", subscriptionID)
    }
    if err != nil {
        return "", false, fmt.Errorf("failed to check subscription: %v", err)
    }
    if status != models.StatusActive && status != models.StatusGracePeriod {
        return status, autoRenew, fmt.Errorf("subscription %s not eligible for renewal (status: %s)", subscriptionID, status)
    }
    if !autoRenew {
        return status, autoRenew, fmt.Errorf("subscription %s has auto_renew disabled", subscriptionID)
    }

    _, err = tx.Exec(ctx, `
        UPDATE subscription_subscription
        SET current_period_start = NOW(), current_period_end = NOW() + INTERVAL '30 days', status = 'active', updated_at = NOW()
        WHERE subscription_id = $1
    `, subscriptionID)
    if err != nil {
        return "", false, fmt.Errorf("failed to renew subscription: %v", err)
    }

    _, err = tx.Exec(ctx, `
        INSERT INTO subscriptions_history (history_id, subscription_id, event_type, old_status, new_status, created_at)
        VALUES ($1, $2, 'renewed', $3, 'active', NOW())
    `, uuid.New(), subscriptionID, status)
    if err != nil {
        return "", false, fmt.Errorf("failed to log renewal: %v", err)
    }

    return status, autoRenew, nil
}

// ChangeSubscriptionPlan changes a subscription's plan
func (sdb *SubscriptionDB) ChangeSubscriptionPlan(ctx context.Context, tx pgx.Tx, subscriptionID, newPlanID uuid.UUID, newPlanName string) (uuid.UUID, models.SubscriptionStatus, error) {
    var oldPlanID uuid.UUID
    var status models.SubscriptionStatus
    err := sdb.Pool.QueryRow(ctx, "SELECT plan_id, status FROM subscriptions_subscription WHERE subscription_id = $1", subscriptionID).Scan(&oldPlanID, &status)
    if err == pgx.ErrNoRows {
        return uuid.UUID{}, "", fmt.Errorf("subscription %s not found", subscriptionID)
    }
    if err != nil {
        return uuid.UUID{}, "", fmt.Errorf("failed to check subscription: %v", err)
    }
    if status != models.StatusActive && status != models.StatusGracePeriod {
        return oldPlanID, status, fmt.Errorf("subscription %s not eligible for plan change (status: %s)", subscriptionID, status)
    }
    if oldPlanID == newPlanID {
        return oldPlanID, status, nil
    }

    _, err = tx.Exec(ctx, `
        UPDATE subscription_subscription
        SET plan_id = $1, plan_name = $2, updated_at = NOW()
        WHERE subscription_id = $3
    `, newPlanID, newPlanName, subscriptionID)
    if err != nil {
        return uuid.UUID{}, "", fmt.Errorf("failed to change plan: %v", err)
    }

    _, err = tx.Exec(ctx, `
        INSERT INTO subscriptions_history (history_id, subscription_id, event_type, old_plan_id, new_plan_id, created_at)
        VALUES ($1, $2, 'plan_changed', $3, $4, NOW())
    `, uuid.New(), subscriptionID, oldPlanID, newPlanID)
    if err != nil {
        return uuid.UUID{}, "", fmt.Errorf("failed to log plan change: %v", err)
    }

    return oldPlanID, status, nil
}

// LogError logs an error to subscription_history
func (sdb *SubscriptionDB) LogError(ctx context.Context, subscriptionID uuid.UUID, errorMsg, streamID string) error {
    // log.Printf("%s: Logging error: %s, StreamID: %s",subscriptionID, errorMsg, streamID)
    notes := fmt.Sprintf("Failed: %s (Stream ID: %s)", errorMsg, streamID)
    // Try inserting with subscription_id first
    _, err := sdb.Pool.Exec(ctx, `
        INSERT INTO subscriptions_history (history_id, subscription_id, event_type, notes, created_at)
        VALUES ($1, $2, $3, $4, NOW())
    `, uuid.New(), subscriptionID, models.EventError, notes)
    if err != nil && strings.Contains(err.Error(), "SQLSTATE 23503") {
        // Retry without subscription_id if foreign key constraint is violated
        _, err = sdb.Pool.Exec(ctx, `
            INSERT INTO subscriptions_history (history_id, event_type, notes, created_at)
            VALUES ($1, $2, $3, NOW())
        `, uuid.New(), models.EventError, notes)
    }
    if err != nil {
        log.Printf("Failed to log error to database: %v", err)
        return fmt.Errorf("failed to log error: %v", err)
    }
    return nil
}