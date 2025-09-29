package services

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "carousel/internal/db"
    "carousel/internal/models"

    "github.com/google/uuid"
)

const (
    StreamName      = "rms.stream"
    GroupName       = "subscription_group"
    DLQStreamName   = "rms.dlq.stream"
    MaxRetries      = 3
    MaxStreamLength = 10000
)

// SubscriptionProcessor handles subscription-related messages
type SubscriptionProcessor struct {
    db       *db.SubscriptionDB
    handlers map[string]func(context.Context, models.SubscriptionRequest) (bool, error)
}

// NewSubscriptionProcessor initializes the processor
func NewSubscriptionProcessor(postgres *db.Postgres) *SubscriptionProcessor {
    sp := &SubscriptionProcessor{
        db:       db.NewSubscriptionDB(postgres.Pool, postgres.Cache),
        handlers: make(map[string]func(context.Context, models.SubscriptionRequest) (bool, error)),
    }
    sp.handlers["subscription.create"] = sp.handleCreate
    sp.handlers["subscription.cancel"] = sp.handleCancel
    sp.handlers["subscription.renew"] = sp.handleRenew
    sp.handlers["subscription.change"] = sp.handleChange
    return sp
}

// Start begins processing the Redis Stream
func (sp *SubscriptionProcessor) Start(ctx context.Context) error {
    // Cache all plans on startup
    if err := sp.db.CacheAllPlans(ctx); err != nil {
        log.Printf("Failed to cache plans on startup: %v", err)
    }

    err := sp.db.Cache.CreateConsumerGroup(ctx, StreamName, GroupName)
    if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
        return fmt.Errorf("failed to create consumer group: %v", err)
    }

    consumerID := uuid.New().String()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            entries, err := sp.db.Cache.XReadGroup(ctx, GroupName, consumerID, StreamName)
            if err != nil {
                log.Printf("Error reading stream: %v", err)
                time.Sleep(1 * time.Second)
                continue
            }

            for _, entry := range entries[0].Messages {
                sp.processMessage(ctx, entry.ID, entry.Values)
            }
        }
    }
}

// processMessage processes a single stream message
func (sp *SubscriptionProcessor) processMessage(ctx context.Context, streamID string, values map[string]interface{}) {
    var req models.SubscriptionRequest
    data, err := json.Marshal(values)
    if err != nil {
        sp.db.LogError(ctx, uuid.UUID{}, fmt.Sprintf("Invalid JSON: %v", err), streamID)
        return
    }
    if err := json.Unmarshal(data, &req); err != nil {
        sp.db.LogError(ctx, uuid.UUID{}, fmt.Sprintf("Invalid JSON: %v", err), streamID)
        return
    }

    handler, exists := sp.handlers[req.Type]
    if !exists {
        sp.db.LogError(ctx, uuid.UUID{}, fmt.Sprintf("Unknown message type: %s", req.Type), streamID)
        sp.db.Cache.XAdd(ctx, DLQStreamName, values)
        return
    }

    success := false
    for attempt := 1; attempt <= MaxRetries; attempt++ {
        success, err = handler(ctx, req)
        if success {
            sp.db.Cache.XAck(ctx, StreamName, GroupName, streamID)
            sp.db.Cache.XTrimMaxLen(ctx, StreamName, MaxStreamLength)
            break
        }
        if attempt == MaxRetries {
            sp.db.Cache.XAdd(ctx, DLQStreamName, values)
            subscriptionID, _ := uuid.Parse(req.SubscriptionID)
            sp.db.LogError(ctx, subscriptionID, fmt.Sprintf("Failed after %d retries: %v", MaxRetries, err), streamID)
            break
        }
        time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
    }
}

// handleCreate processes subscription creation
func (sp *SubscriptionProcessor) handleCreate(ctx context.Context, req models.SubscriptionRequest) (bool, error) {
    subscriptionID, err := uuid.Parse(req.SubscriptionID)
    if err != nil {
        return false, fmt.Errorf("invalid subscription_id: %v", err)
    }
    planID, err := uuid.Parse(req.PlanID)
    if err != nil {
        return false, fmt.Errorf("invalid plan_id: %v", err)
    }
    featureIDs := make([]uuid.UUID, len(req.FeatureIDs))
    for i, fid := range req.FeatureIDs {
        featureID, err := uuid.Parse(fid)
        if err != nil {
            return false, fmt.Errorf("invalid feature_id %s: %v", fid, err)
        }
        featureIDs[i] = featureID
    }

    // Check idempotency
    exists, err := sp.db.CheckSubscriptionExists(ctx, subscriptionID)
    if err != nil {
        return false, err
    }
    if exists {
        log.Printf("Subscription %s already exists", subscriptionID)
        return true, nil
    }

    // Validate plan
    plan, err := sp.db.GetPlan(ctx, planID)
    if err != nil {
        return false, err
    }

    // Begin transaction
    tx, err := sp.db.Pool.Begin(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback(ctx)

    // Create subscription
    subscription := &models.Subscription{
        SubscriptionID:     subscriptionID,
        EntityType:        models.EntityType(req.EntityType),
        EntityID:          req.EntityID,
        PlanID:            planID,
        Status:            models.StatusTrial,
        StartDate:         time.Now(),
        CurrentPeriodStart: time.Now(),
        PlanName:          plan.Name,
        CreatedAt:         time.Now(),
        UpdatedAt:         time.Now(),
    }
    err = sp.db.CreateSubscription(ctx, tx, subscription, featureIDs)
    if err != nil {
        return false, err
    }

    // Commit transaction
    err = tx.Commit(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return true, nil
}

// handleCancel processes subscription cancellation
func (sp *SubscriptionProcessor) handleCancel(ctx context.Context, req models.SubscriptionRequest) (bool, error) {
    subscriptionID, err := uuid.Parse(req.SubscriptionID)
    if err != nil {
        return false, fmt.Errorf("invalid subscription_id: %v", err)
    }

    tx, err := sp.db.Pool.Begin(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback(ctx)

    _, err = sp.db.CancelSubscription(ctx, tx, subscriptionID)
    if err != nil {
        return false, err
    }

    err = tx.Commit(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return true, nil
}

// handleRenew processes subscription renewal
func (sp *SubscriptionProcessor) handleRenew(ctx context.Context, req models.SubscriptionRequest) (bool, error) {
    subscriptionID, err := uuid.Parse(req.SubscriptionID)
    if err != nil {
        return false, fmt.Errorf("invalid subscription_id: %v", err)
    }

    tx, err := sp.db.Pool.Begin(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback(ctx)

    _, _, err = sp.db.RenewSubscription(ctx, tx, subscriptionID)
    if err != nil {
        return false, err
    }

    err = tx.Commit(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return true, nil
}

// handleChange processes subscription plan change
func (sp *SubscriptionProcessor) handleChange(ctx context.Context, req models.SubscriptionRequest) (bool, error) {
    subscriptionID, err := uuid.Parse(req.SubscriptionID)
    if err != nil {
        return false, fmt.Errorf("invalid subscription_id: %v", err)
    }
    newPlanID, err := uuid.Parse(req.PlanID)
    if err != nil {
        return false, fmt.Errorf("invalid plan_id: %v", err)
    }

    // Validate plan
    plan, err := sp.db.GetPlan(ctx, newPlanID)
    if err != nil {
        return false, err
    }

    tx, err := sp.db.Pool.Begin(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback(ctx)

    _, _, err = sp.db.ChangeSubscriptionPlan(ctx, tx, subscriptionID, newPlanID, plan.Name)
    if err != nil {
        return false, err
    }

    err = tx.Commit(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return true, nil
}