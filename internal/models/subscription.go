package models

import (
    "time"

    "github.com/google/uuid"
)

// BillingType represents the subscription billing type
type BillingType string

const (
    BillingTypeMonthlyFixed BillingType = "monthly_fixed"
    BillingTypePayPerOrder  BillingType = "pay_per_order"
)

// SubscriptionStatus represents the subscription status
type SubscriptionStatus string

const (
    StatusTrial       SubscriptionStatus = "trial"
    StatusActive      SubscriptionStatus = "active"
    StatusGracePeriod SubscriptionStatus = "grace_period"
    StatusSuspended   SubscriptionStatus = "suspended"
    StatusCanceled    SubscriptionStatus = "canceled"
)

// EventType represents the history event type
type EventType string

const (
    EventCreated       EventType = "created"
    EventRenewed       EventType = "renewed"
    EventCanceled      EventType = "canceled"
    EventPlanChanged   EventType = "plan_changed"
    EventStatusChanged EventType = "status_changed"
    EventFeatureAdded  EventType = "feature_added"
    EventFeatureRemoved EventType = "feature_removed"
    EventError         EventType = "error"
)

// EntityType represents the entity type
type EntityType string

const (
    EntityCompany    EntityType = "company"
    EntityBranch     EntityType = "branch"
    EntityRestaurant EntityType = "restaurant"
)

// Plan represents a subscription plan
type Plan struct {
    PlanID         uuid.UUID
    Name           string
    BillingType    BillingType
    MonthlyPrice   float64
    IncludedCredits int
    GraceCredits   int
    GraceDays      int
    IsActive       bool
    CreatedAt      time.Time
}

// Feature represents an add-on feature
type Feature struct {
    FeatureID   uuid.UUID
    Name        string
    Description string
    Price       float64
    IsActive    bool
    CreatedAt   time.Time
}

// Subscription represents a customer subscription
type Subscription struct {
    SubscriptionID     uuid.UUID
    EntityType        EntityType
    EntityID          int
    PlanID            uuid.UUID
    Features          []uuid.UUID
    Status            SubscriptionStatus
    StartDate         time.Time
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   *time.Time
    AutoRenew         bool
    CancelAtPeriodEnd bool
    Balance           int
    PlanName          string
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// History represents an audit log entry
type History struct {
    HistoryID      uuid.UUID
    SubscriptionID uuid.UUID
    EventType      EventType
    OldStatus      *SubscriptionStatus
    NewStatus      *SubscriptionStatus
    OldPlanID      *uuid.UUID
    NewPlanID      *uuid.UUID
    OldFeatureID   *uuid.UUID
    NewFeatureID   *uuid.UUID
    Notes          string
    CreatedAt      time.Time
}

// SubscriptionRequest represents a request from Redis Stream
type SubscriptionRequest struct {
    Type           string    `json:"type"`
    RequestID      string    `json:"request_id"`
    SubscriptionID string    `json:"subscription_id"`
    EntityType     string    `json:"entity_type"`
    EntityID       int       `json:"entity_id"`
    PlanID         string    `json:"plan_id"`
    FeatureIDs     []string  `json:"feature_ids"`
}