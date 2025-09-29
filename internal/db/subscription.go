package db

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

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
        FROM subscription_plan WHERE is_active = TRUE
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
    // Try cache first
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
        return nil, fmt.Errorf("plan %s not found in cache", planID)
    }

    // Fallback to database
    var plan models.Plan
    err = sdb.Pool.QueryRow(ctx, `
        SELECT plan_id, name, billing_type, monthly_price, included_credits, grace_credits, grace_days, is_active, created_at
        FROM subscription_plan WHERE plan_id = $1 AND is_active = TRUE
    `, planID).Scan(
        &plan.PlanID, &plan.Name, &plan.BillingType, &plan.MonthlyPrice, &plan.IncludedCredits,
        &plan.GraceCredits, &plan.GraceDays, &plan.IsActive, &plan.CreatedAt)
    if err == pgx.ErrNoRows {
        return nil, fmt.Errorf("plan %s not found or inactive", planID)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get plan: %v", err)
    }

    // Refresh cache after DB fetch
    if err := sdb.CacheAllPlans(ctx); err != nil {
        // Log error but don't fail the request
        fmt.Printf("Failed to refresh plans cache: %v\n", err)
    }

    return &plan, nil
}

// CheckSubscriptionExists checks if a subscription exists
func (sdb *SubscriptionDB) CheckSubscriptionExists(ctx context.Context, subscriptionID uuid.UUID) (bool, error) {
    var exists bool
    err := sdb.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM subscription_subscription WHERE subscription_id = $1)", subscriptionID).Scan(&exists)
    if err != nil {
        return false, fmt.Errorf("failed to check subscription existence: %v", err)
    }
    return exists, nil
}

// CreateSubscription creates a new subscription
func (sdb *SubscriptionDB) CreateSubscription(ctx context.Context, tx pgx.Tx, subscription *models.Subscription, featureIDs []uuid.UUID) error {
    _, err := tx.Exec(ctx, `
        INSERT INTO subscription_subscription (subscription_id, entity_type, entity_id, plan_id, status, start_date, current_period_start, plan_name, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, subscription.SubscriptionID, subscription.EntityType, subscription.EntityID, subscription.PlanID,
        subscription.Status, subscription.StartDate, subscription.CurrentPeriodStart,
        subscription.PlanName, subscription.CreatedAt, subscription.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to insert subscription: %v", err)
    }

    for _, featureID := range featureIDs {
        _, err = tx.Exec(ctx, `
            INSERT INTO subscription_subscription_features (subscription_id, feature_id)
            VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, subscription.SubscriptionID, featureID)
        if err != nil {
            return fmt.Errorf("failed to insert feature %s: %v", featureID, err)
        }
        _, err = tx.Exec(ctx, `
            INSERT INTO subscription_history (history_id, subscription_id, event_type, new_feature_id, created_at)
            VALUES ($1, $2, 'feature_added', $3, NOW())
        `, uuid.New(), subscription.SubscriptionID, featureID)
        if err != nil {
            return fmt.Errorf("failed to log feature addition: %v", err)
        }
    }

    _, err = tx.Exec(ctx, `
        INSERT INTO subscription_history (history_id, subscription_id, event_type, created_at)
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
    err := sdb.Pool.QueryRow(ctx, "SELECT status FROM subscription_subscription WHERE subscription_id = $1", subscriptionID).Scan(&status)
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
        INSERT INTO subscription_history (history_id, subscription_id, event_type, old_status, new_status, created_at)
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
    err := sdb.Pool.QueryRow(ctx, "SELECT status, auto_renew FROM subscription_subscription WHERE subscription_id = $1", subscriptionID).Scan(&status, &autoRenew)
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
        INSERT INTO subscription_history (history_id, subscription_id, event_type, old_status, new_status, created_at)
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
    err := sdb.Pool.QueryRow(ctx, "SELECT plan_id, status FROM subscription_subscription WHERE subscription_id = $1", subscriptionID).Scan(&oldPlanID, &status)
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
        INSERT INTO subscription_history (history_id, subscription_id, event_type, old_plan_id, new_plan_id, created_at)
        VALUES ($1, $2, 'plan_changed', $3, $4, NOW())
    `, uuid.New(), subscriptionID, oldPlanID, newPlanID)
    if err != nil {
        return uuid.UUID{}, "", fmt.Errorf("failed to log plan change: %v", err)
    }

    return oldPlanID, status, nil
}

// LogError logs an error to subscription_history
func (sdb *SubscriptionDB) LogError(ctx context.Context, subscriptionID uuid.UUID, errorMsg, streamID string) error {
    _, err := sdb.Pool.Exec(ctx, `
        INSERT INTO subscription_history (history_id, subscription_id, event_type, notes, created_at)
        VALUES ($1, $2, 'error', $3, NOW())
    `, uuid.New(), subscriptionID, fmt.Sprintf("Failed: %s (Stream ID: %s)", errorMsg, streamID))
    if err != nil {
        return fmt.Errorf("failed to log error: %v", err)
    }
    return nil
}